package tui

import (
	"fmt"
	"strings"
	"time"

	"ps4rpc/internal/app/config"
	"ps4rpc/internal/service/daemon"
	"ps4rpc/internal/service/history"
	"ps4rpc/internal/source/ps4"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	maxLogLines                = 500
	probeEverySeconds          = 10
	historyRefreshEverySeconds = 5
)

type tab int

const (
	tabDashboard tab = iota
	tabSettings
	tabMappings
	tabHistory
)

var tabNames = []string{"Dashboard", "Settings", "Mapped", "History"}

const tabCount = 4

type logMsg string

type tickMsg time.Time

type ps4Msg bool

type Model struct {
	cfg     *config.Config
	version string

	tabIdx tab

	presence *conn
	botConn  *conn

	rpcStatus daemon.PresenceStatus
	botStatus daemon.BotStatus

	logCh chan string
	logs  []string
	vp    viewport.Model

	fields  []field
	cursor  int
	editing bool
	input   textinput.Model
	secret  bool

	rowMapped int
	col       int

	history    []history.Session
	rowHistory int
	historyErr string

	ticks     int
	ps4Online bool
	status    string
	err       string

	fingerprint string

	width  int
	height int
	body   int
	ready  bool
}

func New(cfg *config.Config, version string, startTab int) *Model {
	logCh := make(chan string, 256)

	in := textinput.New()
	in.Prompt = "› "
	in.CharLimit = 256

	m := &Model{
		cfg:      cfg,
		version:  version,
		presence: newConn(daemon.RolePresence, logCh),
		botConn:  newConn(daemon.RoleBot, logCh),
		logCh:    logCh,
		fields:   settingsFields(),
		input:    in,
		vp:       viewport.New(80, 10),
	}
	m.cursor = m.firstSelectable()
	m.fingerprint = config.Fingerprint()
	if startTab >= 0 && startTab <= int(tabHistory) {
		m.tabIdx = tab(startTab)
	}
	redirectStdLog(logCh)
	return m
}

func (m *Model) Init() tea.Cmd {
	m.refreshStatus()
	return tea.Batch(m.listenLogs(), tickCmd(), m.probePS4(), m.ensureDaemons(), m.refreshHistory())
}

type historyMsg struct {
	sessions []history.Session
	err      error
}

func (m *Model) refreshHistory() tea.Cmd {
	ip := m.cfg.Core.IP
	return func() tea.Msg {
		sessions, err := history.Load(ip)
		return historyMsg{sessions: sessions, err: err}
	}
}

func (m *Model) listenLogs() tea.Cmd {
	return func() tea.Msg { return logMsg(<-m.logCh) }
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *Model) probePS4() tea.Cmd {
	ip := m.cfg.Core.IP
	return func() tea.Msg {
		if ip == "" {
			return ps4Msg(false)
		}
		return ps4Msg(ps4.New(ip).Check() == nil)
	}
}

func (m *Model) setPS4Online(online bool) {
	if online == m.ps4Online {
		return
	}
	m.ps4Online = online
	if online {
		m.log("ps4: %s is online", m.cfg.Core.IP)
	} else {
		m.log("ps4: %s is unreachable", m.cfg.Core.IP)
	}
}

func (m *Model) log(format string, args ...any) {
	select {
	case m.logCh <- fmt.Sprintf(format, args...):
	default:
	}
}

func (m *Model) appendLog(s string) {
	for _, line := range strings.Split(strings.TrimRight(s, "\n"), "\n") {
		m.logs = append(m.logs, formatLog(line))
	}
	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
	m.setLogContent()
	m.vp.GotoBottom()
}

func (m *Model) setLogContent() {
	m.vp.SetContent(wrapLog(m.logs, m.vp.Width))
}

func (m *Model) Shutdown() {
	m.presence.drop()
	m.botConn.drop()
}
