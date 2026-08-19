package daemon

import (
	"ps4rpc/internal/app/config"
	"ps4rpc/internal/service/bot"
)

type BotStatus struct {
	Running  bool   `json:"running"`
	HasToken bool   `json:"has_token"`
	Err      string `json:"err,omitempty"`
}

type botDaemon struct {
	logs *logbuf

	cfg          *config.Config
	bot          *bot.Bot
	applied      config.Bot
	appliedCache config.Cache
	err          string
}

func RunBot() error {
	logs := &logbuf{}
	logs.captureStdLog()
	return serve(RoleBot, logs, &botDaemon{logs: logs, cfg: config.Default()})
}

func (b *botDaemon) status() any {
	return BotStatus{
		Running:  b.bot != nil,
		HasToken: b.cfg.Bot.Token != "",
		Err:      b.err,
	}
}

func (b *botDaemon) reload(cfg *config.Config) {
	b.cfg = cfg
	want := Wanted(cfg, RoleBot)
	key := cfg.Bot
	key.Enabled = true
	cacheKey := cfg.Cache

	switch {
	case want && b.bot != nil && (key != b.applied || cacheKey != b.appliedCache):
		b.logs.printf("bot: settings changed, restarting")
		b.stop()
		b.start()
	case want && b.bot == nil:
		b.start()
	case !want && b.bot != nil:
		b.stop()
	}
	b.applied = key
	b.appliedCache = cacheKey
}

func (b *botDaemon) start() {
	b.err = ""
	instance, err := bot.New(b.cfg.Bot, b.cfg.Cache, b.cfg.Core.IP)
	if err != nil {
		b.fail(err)
		return
	}
	if err := instance.Start(); err != nil {
		b.fail(err)
		return
	}
	b.bot = instance
	b.logs.printf("bot: started")
}

func (b *botDaemon) fail(err error) {
	b.err = err.Error()
	b.logs.printf("bot: %v", err)
}

func (b *botDaemon) stop() {
	if b.bot == nil {
		return
	}
	b.bot.Stop()
	b.bot = nil
	b.logs.printf("bot: stopped")
}
