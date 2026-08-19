package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"

	"ps4rpc/internal/app/cache"
	"ps4rpc/internal/app/config"
	"ps4rpc/internal/source/ps4"
)

type Bot struct {
	session *discordgo.Session
	cfg     config.Bot
	cache   config.Cache
	ip      string

	info *ps4.Client

	stop context.CancelFunc

	registered []*discordgo.ApplicationCommand

	mu         sync.Mutex
	psnName    string
	psnAvatar  []byte
	authorMeta ps4.Meta
}

func New(cfg config.Bot, cacheCfg config.Cache, ip string) (*Bot, error) {
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
		cache:   cacheCfg,
		ip:      ip,
		info:    ps4.New(ip),
	}
	b.info.SetAccount(cfg.AccountID)
	if cacheCfg.Enabled {
		store, err := cache.Open(config.CacheDir())
		if err != nil {
			log.Printf("bot: cache disabled: %v", err)
		} else {
			b.info.SetCache(store)
			files, bytes := store.Stats()
			log.Printf("bot: cache at %s (%d files, %s)", store.Dir(), files, humanBytes(bytes))
		}
	}
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
	registered, err := b.session.ApplicationCommandBulkOverwrite(appID, "", cmds)
	if err != nil {
		return fmt.Errorf("failed to register commands: %w", err)
	}
	b.registered = registered
	log.Printf("bot: registered %d commands globally (all servers + DMs)", len(registered))

	if b.info.Cache() != nil {
		ctx, cancel := context.WithCancel(context.Background())
		b.stop = cancel
		go b.warmLoop(ctx)
	}
	return nil
}

func (b *Bot) Stop() {
	if b.stop != nil {
		b.stop()
		b.stop = nil
	}
	b.session.Close()
}

func (b *Bot) iconForTitle(titleID string) ([]byte, ps4.Meta) {
	if titleID == "" {
		return nil, ps4.Meta{}
	}
	data, meta, err := b.info.Icon(titleID)
	if err != nil {
		return nil, ps4.Meta{}
	}
	return data, meta
}

func (b *Bot) iconForName(name string) ([]byte, ps4.Meta) {
	games, _, err := b.info.Games()
	if err != nil {
		return nil, ps4.Meta{}
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

func (b *Bot) author() (author, ps4.Meta) {
	b.mu.Lock()
	name, avatar, meta := b.psnName, b.psnAvatar, b.authorMeta
	b.mu.Unlock()

	if name == "" || meta.Stale() {
		if u, m, err := b.info.Username(); err == nil && u != "" {
			name, meta = u, m
			b.mu.Lock()
			b.psnName, b.authorMeta = name, meta
			b.mu.Unlock()
		}
	}
	if len(avatar) == 0 || meta.Stale() {
		if a, m, err := b.info.Avatar(); err == nil {
			avatar, meta = a, meta.Merge(m)
			b.mu.Lock()
			b.psnAvatar, b.authorMeta = avatar, meta
			b.mu.Unlock()
		}
	}
	return author{name: name, avatar: avatar}, meta
}
