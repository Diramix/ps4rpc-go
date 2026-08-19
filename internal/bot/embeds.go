package bot

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"

	"ps4rpc/internal/ps4info"
)

const (
	libPageSize    = 10
	trophyPageSize = 8
)

type rendered struct {
	embed      *discordgo.MessageEmbed
	files      []*discordgo.File
	components []discordgo.MessageComponent
}

type author struct {
	name   string
	avatar []byte
}

func (a author) attach(e *discordgo.MessageEmbed, r *rendered) {
	if a.name == "" {
		return
	}
	ma := &discordgo.MessageEmbedAuthor{Name: a.name}
	if len(a.avatar) > 0 {
		r.files = append(r.files, &discordgo.File{
			Name:        "avatar.png",
			ContentType: "image/png",
			Reader:      bytes.NewReader(a.avatar),
		})
		ma.IconURL = "attachment://avatar.png"
	}
	e.Author = ma
}

func iconFile(data []byte) ([]*discordgo.File, string) {
	if len(data) == 0 {
		return nil, ""
	}
	return []*discordgo.File{{
		Name:        "icon.png",
		ContentType: "image/png",
		Reader:      bytes.NewReader(data),
	}}, "attachment://icon.png"
}

func pageRow(prevID, nextID string, page, pages int) []discordgo.MessageComponent {
	if pages <= 1 {
		return nil
	}
	return []discordgo.MessageComponent{discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Emoji: &discordgo.ComponentEmoji{Name: "◀"}, Style: discordgo.SecondaryButton, CustomID: prevID, Disabled: page == 0},
		discordgo.Button{Emoji: &discordgo.ComponentEmoji{Name: "▶"}, Style: discordgo.SecondaryButton, CustomID: nextID, Disabled: page >= pages-1},
	}}}
}

func statusEmbed(st ps4info.Status, games []ps4info.Game, uptime string, icon []byte, name string, auth author) rendered {
	e := &discordgo.MessageEmbed{}
	switch st.State {
	case ps4info.StateGame:
		e.Color = colorPS4
		e.Title = "🟢 Playing"
		title := name
		if title == "" {
			title = st.TitleID
		}
		e.Description = "**" + title + "**"
		e.Fields = append(e.Fields,
			&discordgo.MessageEmbedField{Name: "Title ID", Value: st.TitleID, Inline: true},
			&discordgo.MessageEmbedField{Name: "Playing for", Value: elapsed(st.SessionStart), Inline: true},
		)
	case ps4info.StateMenu:
		e.Color = colorPS4
		e.Title = "🟢 In the main menu"
	default:
		e.Color = colorOffline
		e.Title = "😴 Offline / asleep"
	}

	if st.Online {
		e.Fields = append(e.Fields,
			&discordgo.MessageEmbedField{Name: "📀 Used", Value: humanBytes(ps4info.TotalSize(games)), Inline: true},
			&discordgo.MessageEmbedField{Name: "⏱ Uptime", Value: valueOr(uptime), Inline: true},
		)
	}

	files, url := iconFile(icon)
	if url != "" {
		e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: url}
	}
	e.Footer = &discordgo.MessageEmbedFooter{Text: "PlayStation®4"}

	r := rendered{embed: e, files: files}
	auth.attach(e, &r)
	return r
}

func valueOr(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func sortGames(games []ps4info.Game, mode string) {
	switch mode {
	case "size":
		sort.Slice(games, func(i, j int) bool { return games[i].Size > games[j].Size })
	case "recent":
		sort.Slice(games, func(i, j int) bool { return games[i].LastPlayed.After(games[j].LastPlayed) })
	default:
		sort.Slice(games, func(i, j int) bool { return strings.ToLower(games[i].Name) < strings.ToLower(games[j].Name) })
	}
}

func libraryEmbed(games []ps4info.Game, mode string, page int) rendered {
	sortGames(games, mode)
	pages := (len(games) + libPageSize - 1) / libPageSize
	if pages == 0 {
		pages = 1
	}
	page = clamp(page, 0, pages-1)

	var sb strings.Builder
	start := page * libPageSize
	end := min(start+libPageSize, len(games))
	for i := start; i < end; i++ {
		g := games[i]
		fmt.Fprintf(&sb, "**%s**\n`%s` · %s · %s\n\n", g.Name, g.TitleID, humanBytes(g.Size), discordRel(g.LastPlayed))
	}
	e := &discordgo.MessageEmbed{
		Title:       "📚 Game library",
		Description: sb.String(),
		Color:       colorPS4,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d · %d games · %s", page+1, pages, len(games), humanBytes(ps4info.TotalSize(games))),
		},
	}
	prev := fmt.Sprintf("lib:%s:%d", mode, page-1)
	next := fmt.Sprintf("lib:%s:%d", mode, page+1)
	return rendered{embed: e, components: pageRow(prev, next, page, pages)}
}

func gameEmbed(g ps4info.Game, playtime string, starts int, icon []byte, auth author) rendered {
	e := &discordgo.MessageEmbed{
		Title: g.Name,
		Color: colorPS4,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Title ID", Value: g.TitleID, Inline: true},
			{Name: "Size", Value: humanBytes(g.Size), Inline: true},
			{Name: "Last played", Value: discordDate(g.LastPlayed), Inline: true},
		},
	}
	if playtime != "" {
		v := playtime
		if starts > 0 {
			v = fmt.Sprintf("%s · %d launches", playtime, starts)
		}
		e.Fields = append(e.Fields, &discordgo.MessageEmbedField{Name: "Playtime", Value: v, Inline: true})
	}
	files, url := iconFile(icon)
	if url != "" {
		e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: url}
	}
	r := rendered{embed: e, files: files}
	auth.attach(e, &r)
	return r
}

func recentEmbed(games []ps4info.Game) rendered {
	sortGames(games, "recent")
	var sb strings.Builder
	for i := 0; i < len(games) && i < 10; i++ {
		g := games[i]
		fmt.Fprintf(&sb, "**%s** - %s\n", g.Name, discordRel(g.LastPlayed))
	}
	if sb.Len() == 0 {
		sb.WriteString("no data")
	}
	e := &discordgo.MessageEmbed{Title: "Recently played", Description: sb.String(), Color: colorPS4}
	return rendered{embed: e}
}

func filterTrophies(all []ps4info.Trophy, filter string) []ps4info.Trophy {
	if filter == "" || filter == "all" {
		return all
	}
	out := all[:0:0]
	for _, t := range all {
		if (filter == "unlocked") == t.Unlocked {
			out = append(out, t)
		}
	}
	return out
}

func trophyEmbed(title ps4info.TrophyTitle, all []ps4info.Trophy, filter string, page int, icon []byte) rendered {
	list := filterTrophies(all, filter)
	pages := (len(list) + trophyPageSize - 1) / trophyPageSize
	if pages == 0 {
		pages = 1
	}
	page = clamp(page, 0, pages-1)

	var sb strings.Builder
	fmt.Fprintf(&sb, "%s  **%d%%**  ·  %d/%d\n🏆 %d · 🥇 %d · 🥈 %d · 🥉 %d\n\n",
		progressBar(title.Progress), title.Progress, title.Unlocked, title.Total,
		title.Platinum, title.Gold, title.Silver, title.Bronze)

	start := page * trophyPageSize
	end := min(start+trophyPageSize, len(list))
	for i := start; i < end; i++ {
		t := list[i]
		mark := "🔒"
		when := ""
		if t.Unlocked {
			mark = "✅"
			when = " · " + discordDate(t.TimeUnlocked)
		}
		name, desc := t.Name, t.Description
		if t.Hidden && !t.Unlocked {
			name, desc = "Hidden trophy", "?"
		}
		fmt.Fprintf(&sb, "%s %s **%s**%s\n%s\n\n", gradeEmoji(t.Grade), mark, name, when, desc)
	}

	e := &discordgo.MessageEmbed{
		Title:       "🏆 " + title.Name,
		Description: sb.String(),
		Color:       colorTrophy,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Page %d/%d · last trophy", page+1, pages),
		},
	}
	if !title.LastUnlocked.IsZero() {
		e.Timestamp = title.LastUnlocked.Format("2006-01-02T15:04:05Z07:00")
	}
	files, url := iconFile(icon)
	if url != "" {
		e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: url}
	}
	prev := fmt.Sprintf("trp:%s:%s:%d", title.CommID, orAll(filter), page-1)
	next := fmt.Sprintf("trp:%s:%s:%d", title.CommID, orAll(filter), page+1)
	return rendered{embed: e, files: files, components: pageRow(prev, next, page, pages)}
}

func orAll(s string) string {
	if s == "" {
		return "all"
	}
	return s
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
