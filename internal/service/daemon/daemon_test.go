package daemon

import (
	"testing"

	"ps4rpc/internal/app/config"
	"ps4rpc/internal/service/presence"
)

func TestWantedFollowsTheConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Core.IP = "10.0.0.2"
	cfg.Bot.Token = "token"
	cfg.Bot.Enabled = true

	if !Wanted(cfg, RolePresence) || !Wanted(cfg, RoleBot) {
		t.Fatal("both roles should be wanted with a full config")
	}

	cfg.Core.IP = ""
	if Wanted(cfg, RolePresence) || Wanted(cfg, RoleBot) {
		t.Error("no role should be wanted without a console IP")
	}

	cfg.Core.IP = "10.0.0.2"
	cfg.Core.Enabled = false
	cfg.Bot.Token = ""
	if Wanted(cfg, RolePresence) {
		t.Error("presence should not be wanted when disabled")
	}
	if Wanted(cfg, RoleBot) {
		t.Error("bot should not be wanted without a token")
	}
}

func TestBotReloadTracksTheDesiredState(t *testing.T) {
	b := &botDaemon{logs: &logbuf{}, cfg: config.Default()}

	cfg := config.Default()
	cfg.Core.IP = "10.0.0.2"
	cfg.Bot.Token = ""
	cfg.Bot.Enabled = true
	b.reload(cfg)
	if b.bot != nil {
		t.Fatal("bot started without a token")
	}

	cfg = cfg.Clone()
	cfg.Bot.Token = "token"
	cfg.Bot.Enabled = false
	b.reload(cfg)
	if b.bot != nil {
		t.Fatal("bot started while disabled")
	}
	if !b.status().(BotStatus).HasToken {
		t.Error("status lost the token flag")
	}
}

func TestPresenceStatusReportsStoppedByDefault(t *testing.T) {
	p := &presenceRunner{logs: &logbuf{}, cfg: config.Default()}
	p.svc = presence.New(p.cfg, p.logs.printf)

	st := p.status().(PresenceStatus)
	if st.Running {
		t.Fatal("a freshly built presence daemon reports itself as running")
	}
}

func TestLogbufKeepsOnlyTheTail(t *testing.T) {
	l := &logbuf{}
	for i := 0; i < maxLogLines+50; i++ {
		l.printf("line %d", i)
	}
	if got := len(l.history()); got != maxLogLines {
		t.Fatalf("history holds %d lines, want %d", got, maxLogLines)
	}
}

func TestLogbufSplitsMultilineWrites(t *testing.T) {
	l := &logbuf{}
	l.add("first\nsecond\n")
	if got := l.history(); len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("multiline write not split: %q", got)
	}
}
