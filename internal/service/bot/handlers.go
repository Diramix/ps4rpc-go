package bot

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ps4rpc/internal/source/ps4"
)

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !b.allowed(i) {
		b.deny(s, i)
		return
	}
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		b.handleCommand(s, i)
	case discordgo.InteractionApplicationCommandAutocomplete:
		b.handleAutocomplete(s, i)
	case discordgo.InteractionMessageComponent:
		b.handleComponent(s, i)
	}
}

func (b *Bot) allowed(i *discordgo.InteractionCreate) bool {
	if b.cfg.OwnerID == "" {
		return true
	}
	return interactionUserID(i) == b.cfg.OwnerID
}

func interactionUserID(i *discordgo.InteractionCreate) string {
	if i.Member != nil && i.Member.User != nil {
		return i.Member.User.ID
	}
	if i.User != nil {
		return i.User.ID
	}
	return ""
}

func (b *Bot) deny(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Printf("bot: DENIED user=%q (owner=%q) type=%v", interactionUserID(i), b.cfg.OwnerID, i.Type)
	if i.Type == discordgo.InteractionApplicationCommandAutocomplete {
		_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{},
		})
		return
	}
	b.ephemeral(s, i, "⛔ Only the owner can use this bot.")
}

func optString(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, o := range opts {
		if o.Name == name {
			return o.StringValue()
		}
	}
	return ""
}

func (b *Bot) defer_(s *discordgo.Session, i *discordgo.InteractionCreate) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
}

func (b *Bot) edit(s *discordgo.Session, i *discordgo.InteractionCreate, r rendered) {
	edit := &discordgo.WebhookEdit{Embeds: &[]*discordgo.MessageEmbed{r.embed}}
	if r.components != nil {
		edit.Components = &r.components
	}
	if r.files != nil {
		edit.Files = r.files
		edit.Attachments = &[]*discordgo.MessageAttachment{}
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, edit); err != nil {
		log.Printf("bot: edit response: %v", err)
	}
}

func (b *Bot) fail(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	e := &discordgo.MessageEmbed{Description: "⚠️ " + msg, Color: colorOffline}
	b.edit(s, i, rendered{embed: e})
}

func (b *Bot) ephemeral(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg, Flags: discordgo.MessageFlagsEphemeral},
	})
}

func (b *Bot) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	switch data.Name {
	case "info":
		b.defer_(s, i)
		b.edit(s, i, b.buildStatus())
	case "library":
		b.defer_(s, i)
		b.cmdLibrary(s, i, optString(data.Options, "sort"))
	case "game":
		b.defer_(s, i)
		b.cmdGame(s, i, optString(data.Options, "name"))
	case "recent":
		b.defer_(s, i)
		b.cmdRecent(s, i)
	case "shot":
		b.defer_(s, i)
		b.cmdShot(s, i)
	case "trophy":
		b.cmdTrophy(s, i, data.Options)
	}
}

func (b *Bot) buildStatus() rendered {
	st, _ := b.info.Status()
	var games []ps4.Game
	var uptime string
	var name string
	var icon []byte
	if st.Online {
		games, _ = b.info.Games()
		if d, err := b.info.Uptime(); err == nil {
			uptime = humanDuration(d)
		}
		if st.State == ps4.StateGame {
			for _, g := range games {
				if g.TitleID == st.TitleID {
					name = g.Name
				}
			}
			icon = b.iconForTitle(st.TitleID)
		}
	}
	return statusEmbed(st, games, uptime, icon, name, b.author())
}

func (b *Bot) cmdLibrary(s *discordgo.Session, i *discordgo.InteractionCreate, mode string) {
	games, err := b.info.Games()
	if err != nil {
		b.fail(s, i, "failed to read the library: "+err.Error())
		return
	}
	b.edit(s, i, libraryEmbed(games, mode, 0))
}

func (b *Bot) cmdGame(s *discordgo.Session, i *discordgo.InteractionCreate, name string) {
	games, err := b.info.Games()
	if err != nil {
		b.fail(s, i, err.Error())
		return
	}
	g, ok := findGame(games, name)
	if !ok {
		b.fail(s, i, "game not found: "+name)
		return
	}
	playtime, starts := b.playtimeOf(g.TitleID)
	b.edit(s, i, gameEmbed(g, playtime, starts, b.iconForTitle(g.TitleID), b.author()))
}

func (b *Bot) playtimeOf(titleID string) (string, int) {
	d, starts, ok := b.info.Playtime(titleID)
	if !ok || d <= 0 {
		return "", 0
	}
	return humanDuration(d), starts
}

func (b *Bot) cmdRecent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	games, err := b.info.Games()
	if err != nil {
		b.fail(s, i, err.Error())
		return
	}
	b.edit(s, i, recentEmbed(games))
}

func (b *Bot) cmdShot(s *discordgo.Session, i *discordgo.InteractionCreate) {
	name, data, err := b.info.LatestScreenshot()
	if err != nil {
		b.fail(s, i, "no screenshots or clips found")
		return
	}
	edit := &discordgo.WebhookEdit{
		Files:       []*discordgo.File{{Name: name, Reader: bytes.NewReader(data)}},
		Attachments: &[]*discordgo.MessageAttachment{},
	}
	if _, err := s.InteractionResponseEdit(i.Interaction, edit); err != nil {
		log.Printf("bot: shot edit: %v", err)
	}
}

func (b *Bot) cmdTrophy(s *discordgo.Session, i *discordgo.InteractionCreate, opts []*discordgo.ApplicationCommandInteractionDataOption) {
	b.defer_(s, i)
	game := optString(opts, "game")
	filter := optString(opts, "filter")
	title, err := b.info.FindTitle(game)
	if err != nil {
		b.fail(s, i, "no trophies found for: "+game)
		return
	}
	trophies, err := b.info.Trophies(title.CommID)
	if err != nil {
		b.fail(s, i, err.Error())
		return
	}
	b.edit(s, i, trophyEmbed(title, trophies, filter, 0, b.iconForName(title.Name)))
}

func (b *Bot) handleAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	var focused string
	for _, o := range data.Options {
		if o.Focused {
			focused = o.StringValue()
		}
	}

	var choices []*discordgo.ApplicationCommandOptionChoice
	if data.Name == "trophy" {
		titles, _ := b.info.SearchTitles(focused, 25)
		for _, t := range titles {
			choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: truncate(t.Name, 100), Value: t.Name})
		}
	} else {
		games, _ := b.info.Games()
		f := strings.ToLower(focused)
		for _, g := range games {
			if f == "" || strings.Contains(strings.ToLower(g.Name), f) {
				choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: truncate(g.Name, 100), Value: g.Name})
			}
			if len(choices) >= 25 {
				break
			}
		}
	}
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

func (b *Bot) handleComponent(s *discordgo.Session, i *discordgo.InteractionCreate) {
	id := i.MessageComponentData().CustomID

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{Type: discordgo.InteractionResponseDeferredMessageUpdate})

	parts := strings.Split(id, ":")
	var r rendered
	switch parts[0] {
	case "lib":
		mode, page := parts[1], atoiSafe(parts[2])
		games, _ := b.info.Games()
		r = libraryEmbed(games, mode, page)
	case "trp":
		comm, filter, page := parts[1], parts[2], atoiSafe(parts[3])
		title, err := b.findTitleByComm(comm)
		if err != nil {
			return
		}
		trophies, _ := b.info.Trophies(comm)
		r = trophyEmbed(title, trophies, filter, page, b.iconForName(title.Name))
	default:
		return
	}
	b.edit(s, i, r)
}

func (b *Bot) findTitleByComm(comm string) (ps4.TrophyTitle, error) {
	titles, err := b.info.TrophyTitles()
	if err != nil {
		return ps4.TrophyTitle{}, err
	}
	for _, t := range titles {
		if t.CommID == comm {
			return t, nil
		}
	}
	return ps4.TrophyTitle{}, fmt.Errorf("not found")
}

func findGame(games []ps4.Game, name string) (ps4.Game, bool) {
	want := strings.ToLower(strings.TrimSpace(name))
	for _, g := range games {
		if strings.EqualFold(g.Name, name) || g.TitleID == name {
			return g, true
		}
	}
	for _, g := range games {
		if strings.Contains(strings.ToLower(g.Name), want) {
			return g, true
		}
	}
	return ps4.Game{}, false
}

func atoiSafe(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
