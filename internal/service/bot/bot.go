package bot

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"ps4rpc/internal/app/config"
	"ps4rpc/internal/source/ps4"
)

type Bot struct {
	session *discordgo.Session
	cfg     config.Bot
	ip      string

	info *ps4.Client

	registered []*discordgo.ApplicationCommand

	mu         sync.Mutex
	psnName    string
	psnAvatar  []byte
	avatarDone bool
}

func New(cfg config.Bot, ip string) (*Bot, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("bot token is empty; set bot.token in the config")
	}
	if ip == "" {
		return nil, fmt.Errorf("PS4 IP is empty; set var.ip in the config")
	}
	s, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, err
	}
	s.Identify.Intents = discordgo.IntentsGuilds

	b := &Bot{
		session: s,
		cfg:     cfg,
		ip:      ip,
		info:    ps4.New(ip),
	}
	b.info.SetAccount(cfg.AccountID)
	s.AddHandler(b.onInteraction)
	s.AddHandler(func(_ *discordgo.Session, r *discordgo.Ready) {
		log.Printf("bot: logged in as %s#%s", r.User.Username, r.User.Discriminator)
	})
	return b, nil
}

func (b *Bot) Start() error {
	if err := b.session.Open(); err != nil {
		return err
	}
	appID := b.session.State.User.ID
	cmds := commandDefs()
	registered, err := b.session.ApplicationCommandBulkOverwrite(appID, b.cfg.GuildID, cmds)
	if err != nil {
		if b.cfg.GuildID != "" {
			return fmt.Errorf("failed to register commands to guild %s - is the bot a member of that server? Leave bot.guild_id empty for global commands: %w", b.cfg.GuildID, err)
		}
		return fmt.Errorf("failed to register commands: %w", err)
	}
	b.registered = registered
	scope := "globally (all servers + DMs)"
	if b.cfg.GuildID != "" {
		scope = "on guild " + b.cfg.GuildID
	}
	log.Printf("bot: registered %d commands %s", len(registered), scope)
	return nil
}

func (b *Bot) Stop() {
	b.session.Close()
}

func (b *Bot) iconForTitle(titleID string) []byte {
	if titleID == "" {
		return nil
	}
	data, err := b.info.Icon(titleID)
	if err != nil {
		return nil
	}
	return data
}

func (b *Bot) iconForName(name string) []byte {
	games, err := b.info.Games()
	if err != nil {
		return nil
	}
	want := strings.ToLower(strings.TrimSpace(name))
	var best string
	for _, g := range games {
		gn := strings.ToLower(g.Name)
		if gn == want {
			best = g.TitleID
			break
		}
		if best == "" && (strings.Contains(gn, want) || strings.Contains(want, gn)) {
			best = g.TitleID
		}
	}
	return b.iconForTitle(best)
}

func (b *Bot) author() author {
	b.mu.Lock()
	name, avatar, done := b.psnName, b.psnAvatar, b.avatarDone
	b.mu.Unlock()

	if name == "" {
		if u, err := b.info.Username(); err == nil && u != "" {
			name = u
			b.mu.Lock()
			b.psnName = u
			b.mu.Unlock()
		}
	}
	if !done {
		avatar, _ = b.info.Avatar()
		b.mu.Lock()
		b.psnAvatar, b.avatarDone = avatar, true
		b.mu.Unlock()
	}
	return author{name: name, avatar: avatar}
}
