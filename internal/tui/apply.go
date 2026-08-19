package tui

import (
	"ps4rpc/internal/config"

	tea "github.com/charmbracelet/bubbletea"
)

type configMsg struct{ cfg *config.Config }

func (m *Model) applyNow() {
	if err := m.cfg.Save(); err != nil {
		m.err = "save: " + err.Error()
		return
	}
	m.fingerprint = config.Fingerprint()
	m.reconcile()
}

func (m *Model) reconcile() {
	m.svc.SetConfig(m.cfg)

	wantRPC := m.cfg.Var.Enabled && m.cfg.Var.IP != ""
	running := m.svc.Status().Running
	switch {
	case wantRPC && running && m.rpcKey() != m.appliedRPC:
		m.log("rpc: settings changed, restarting")
		m.svc.Stop()
		m.startRPC()
	case wantRPC && !running:
		m.startRPC()
	case !wantRPC && running:
		m.stopRPC()
	}
	m.appliedRPC = m.rpcKey()

	wantBot := m.cfg.Bot.Enabled && m.cfg.Bot.Token != ""
	switch {
	case wantBot && m.botOnline && m.cfg.Bot != m.appliedBot:
		m.log("bot: settings changed, restarting")
		m.stopBot()
		m.startBot()
	case wantBot && !m.botOnline:
		m.startBot()
	case !wantBot && m.botOnline:
		m.stopBot()
	}
	m.appliedBot = m.cfg.Bot
}

type rpcKey struct {
	ip       string
	clientID int64
}

func (m *Model) rpcKey() rpcKey {
	return rpcKey{ip: m.cfg.Var.IP, clientID: m.cfg.Var.ClientID}
}

func (m *Model) watchConfig() tea.Cmd {
	known := m.fingerprint
	return func() tea.Msg {
		if config.Fingerprint() == known {
			return nil
		}
		cfg, existed, err := config.Load()
		if err != nil || !existed {
			return nil
		}
		return configMsg{cfg: cfg}
	}
}

func (m *Model) onExternalConfig(cfg *config.Config) {
	m.fingerprint = config.Fingerprint()
	if cfg.Equal(m.cfg) {
		return
	}
	m.cfg = cfg
	if m.rowMapped >= len(m.cfg.Mapped) {
		m.rowMapped = max(0, len(m.cfg.Mapped)-1)
	}
	m.log("config: reloaded from disk")
	m.reconcile()
}
