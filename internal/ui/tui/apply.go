package tui

import (
	"ps4rpc/internal/app/config"

	tea "github.com/charmbracelet/bubbletea"
)

type configMsg struct{ cfg *config.Config }

func (m *Model) applyNow() tea.Cmd {
	if err := m.cfg.Save(); err != nil {
		m.err = "save: " + err.Error()
		return nil
	}
	m.fingerprint = config.Fingerprint()
	return m.ensureDaemons()
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

func (m *Model) onExternalConfig(cfg *config.Config) tea.Cmd {
	m.fingerprint = config.Fingerprint()
	if cfg.Equal(m.cfg) {
		return nil
	}
	m.cfg = cfg
	if m.rowMapped >= len(m.cfg.Mapped) {
		m.rowMapped = max(0, len(m.cfg.Mapped)-1)
	}
	m.log("config: reloaded from disk")
	return m.ensureDaemons()
}
