package tui

import (
	"io"
	"log"

	"ps4rpc/internal/app/config"

	tea "github.com/charmbracelet/bubbletea"
)

type logWriter struct{ ch chan string }

func (w logWriter) Write(p []byte) (int, error) {
	select {
	case w.ch <- string(p):
	default:
	}
	return len(p), nil
}

func redirectStdLog(ch chan string) {
	log.SetFlags(0)
	log.SetOutput(io.Writer(logWriter{ch: ch}))
}

func (m *Model) firstSelectable() int {
	for i, f := range m.fields {
		if f.selectable() {
			return i
		}
	}
	return 0
}

func (m *Model) moveCursor(delta int) {
	n := len(m.fields)
	for i := 0; i < n; i++ {
		m.cursor += delta
		if m.cursor < 0 {
			m.cursor = n - 1
		}
		if m.cursor >= n {
			m.cursor = 0
		}
		if m.fields[m.cursor].selectable() {
			return
		}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.ready = true
		return m, nil

	case logMsg:
		m.appendLog(string(msg))
		return m, m.listenLogs()

	case configMsg:
		return m, m.onExternalConfig(msg.cfg)

	case historyMsg:
		if msg.err != nil {
			m.historyErr = msg.err.Error()
		} else {
			m.historyErr = ""
			m.history = msg.sessions
		}
		return m, nil

	case tickMsg:
		m.ticks++
		m.refreshStatus()
		cmds := []tea.Cmd{tickCmd(), m.watchConfig()}
		if m.rpcStatus.Running {
			m.setPS4Online(m.rpcStatus.PS4Online)
		} else if m.ticks%probeEverySeconds == 0 {
			cmds = append(cmds, m.probePS4())
		}
		if m.tabIdx == tabHistory && m.ticks%historyRefreshEverySeconds == 0 {
			cmds = append(cmds, m.refreshHistory())
		}
		return m, tea.Batch(cmds...)

	case ps4Msg:
		m.setPS4Online(bool(msg))
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.handleEditKey(msg)
	}

	switch keyName(msg) {
	case "ctrl+c", "q":
		m.Shutdown()
		return m, tea.Quit
	case "tab", "right":
		m.tabIdx = (m.tabIdx + 1) % tabCount
		return m, m.onTabChanged()
	case "shift+tab", "left":
		m.tabIdx = (m.tabIdx + tabCount - 1) % tabCount
		return m, m.onTabChanged()
	case "1":
		m.tabIdx = tabDashboard
		return m, m.onTabChanged()
	case "2":
		m.tabIdx = tabSettings
		return m, m.onTabChanged()
	case "3":
		m.tabIdx = tabMappings
		return m, m.onTabChanged()
	case "4":
		m.tabIdx = tabHistory
		return m, m.onTabChanged()
	}

	switch m.tabIdx {
	case tabDashboard:
		return m.handleDashboardKey(msg)
	case tabSettings:
		return m.handleSettingsKey(msg)
	case tabMappings:
		return m.handleMappingsKey(msg)
	case tabHistory:
		return m.handleHistoryKey(msg)
	}
	return m, nil
}

func (m *Model) onTabChanged() tea.Cmd {
	if m.tabIdx == tabHistory {
		return m.refreshHistory()
	}
	return nil
}

func (m *Model) handleHistoryKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyName(msg) {
	case "up", "k":
		m.moveHistoryRow(-1)
	case "down", "j":
		m.moveHistoryRow(1)
	}
	return m, nil
}

func (m *Model) moveHistoryRow(delta int) {
	n := len(m.history)
	if n == 0 {
		return
	}
	m.rowHistory = (m.rowHistory + delta + n) % n
}

func (m *Model) handleDashboardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyName(msg) {
	case "s":
		m.err = ""
		m.cfg.Core.Enabled = !m.rpcStatus.Running
		return m, m.applyNow()
	case "c":
		m.logs = nil
		m.vp.SetContent("")
		return m, nil
	case "up", "k":
		m.vp.ScrollUp(1)
	case "down", "j":
		m.vp.ScrollDown(1)
	case "pgup":
		m.vp.HalfPageUp()
	case "pgdown":
		m.vp.HalfPageDown()
	}
	return m, nil
}

func (m *Model) handleSettingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := m.fields[m.cursor]
	switch keyName(msg) {
	case "up", "k":
		m.moveCursor(-1)
	case "down", "j":
		m.moveCursor(1)
	case "v":
		if f.kind == fieldSecret {
			m.secret = !m.secret
		}
	case " ", "enter":
		m.err = ""
		switch f.kind {
		case fieldToggle:
			f.toggle(m.cfg)
			return m, m.applyNow()
		case fieldText, fieldSecret:
			m.editing = true
			m.input.SetValue(f.get(m.cfg))
			m.input.CursorEnd()
			m.input.EchoMode = 0
			return m, m.input.Focus()
		}
	}
	return m, nil
}

func (m *Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyName(msg) {
	case "esc":
		m.editing = false
		m.input.Blur()
		return m, nil
	case "enter":
		if err := m.commitEdit(m.input.Value()); err != nil {
			m.err = err.Error()
			return m, nil
		}
		m.editing = false
		m.err = ""
		m.input.Blur()
		return m, m.applyNow()
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) commitEdit(v string) error {
	if m.tabIdx == tabSettings {
		return m.fields[m.cursor].set(m.cfg, v)
	}
	return m.commitRowEdit(v)
}

func (m *Model) handleMappingsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch keyName(msg) {
	case "up", "k":
		m.moveRow(-1)
	case "down", "j":
		m.moveRow(1)
	case "h":
		if m.col > 0 {
			m.col--
		}
	case "l":
		if m.col < 2 {
			m.col++
		}
	case "a":
		m.addRow()
		return m, m.applyNow()
	case "d":
		m.deleteRow()
		return m, m.applyNow()
	case " ", "enter":
		m.err = ""
		if len(m.cfg.Mapped) == 0 {
			return m, nil
		}
		m.editing = true
		m.input.SetValue(m.cellValue())
		m.input.CursorEnd()
		return m, m.input.Focus()
	}
	return m, nil
}

func (m *Model) moveRow(delta int) {
	n := len(m.cfg.Mapped)
	if n == 0 {
		return
	}
	m.rowMapped = (m.rowMapped + delta + n) % n
}

func (m *Model) cellValue() string {
	mp := m.cfg.Mapped[m.rowMapped]
	switch m.col {
	case 1:
		return mp.Name
	case 2:
		return mp.Image
	}
	return mp.TitleID
}

func (m *Model) commitRowEdit(v string) error {
	if len(m.cfg.Mapped) == 0 {
		return nil
	}
	mp := &m.cfg.Mapped[m.rowMapped]
	switch m.col {
	case 1:
		mp.Name = v
	case 2:
		mp.Image = v
	default:
		mp.TitleID = v
	}
	return nil
}

func (m *Model) addRow() {
	m.cfg.Mapped = append(m.cfg.Mapped, config.Mapped{})
	m.rowMapped = len(m.cfg.Mapped) - 1
}

func (m *Model) deleteRow() {
	if len(m.cfg.Mapped) == 0 {
		return
	}
	m.cfg.Mapped = append(m.cfg.Mapped[:m.rowMapped], m.cfg.Mapped[m.rowMapped+1:]...)
	if m.rowMapped >= len(m.cfg.Mapped) {
		m.rowMapped = max(0, len(m.cfg.Mapped)-1)
	}
}
