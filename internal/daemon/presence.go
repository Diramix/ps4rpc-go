package daemon

import (
	"context"

	"ps4rpc/internal/config"
	"ps4rpc/internal/rpcsvc"
)

type PresenceStatus struct {
	Running   bool   `json:"running"`
	PS4Online bool   `json:"ps4_online"`
	DiscordOK bool   `json:"discord_ok"`
	TitleID   string `json:"title_id"`
	GameName  string `json:"game_name"`
	IP        string `json:"ip"`
	Err       string `json:"err,omitempty"`
}

type rpcKey struct {
	ip       string
	clientID int64
}

type presence struct {
	logs *logbuf
	svc  *rpcsvc.Service

	cfg     *config.Config
	applied rpcKey
	err     string
}

func RunPresence() error {
	logs := &logbuf{}
	logs.captureStdLog()
	p := &presence{logs: logs, cfg: config.Default()}
	p.svc = rpcsvc.New(p.cfg, logs.printf)
	return serve(RolePresence, logs, p)
}

func (p *presence) status() any {
	st := p.svc.Status()
	return PresenceStatus{
		Running:   st.Running,
		PS4Online: st.PS4Online,
		DiscordOK: st.DiscordOK,
		TitleID:   st.TitleID,
		GameName:  st.GameName,
		IP:        p.cfg.Core.IP,
		Err:       p.err,
	}
}

func (p *presence) reload(cfg *config.Config) {
	p.cfg = cfg
	p.svc.SetConfig(cfg)

	want := Wanted(cfg, RolePresence)
	running := p.svc.Status().Running
	key := rpcKey{ip: cfg.Core.IP, clientID: cfg.Core.ClientID}

	switch {
	case want && running && key != p.applied:
		p.logs.printf("rpc: settings changed, restarting")
		p.svc.Stop()
		p.start()
	case want && !running:
		p.start()
	case !want && running:
		p.logs.printf("rpc: stopping")
		p.svc.Stop()
	}
	p.applied = key
}

func (p *presence) start() {
	p.err = ""
	if err := p.svc.Start(context.Background()); err != nil {
		p.err = err.Error()
		p.logs.printf("rpc: %v", err)
		return
	}
	p.logs.printf("rpc: started for %s", p.cfg.Core.IP)
}

func (p *presence) stop() { p.svc.Stop() }
