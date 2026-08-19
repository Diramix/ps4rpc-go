package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ps4rpc/internal/bot"
	"ps4rpc/internal/config"
	"ps4rpc/internal/ps4"
	"ps4rpc/internal/rpcsvc"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	maxLogLines       = 500
	probeEverySeconds = 10
)

type tab int

const (
	tabDashboard tab = iota
	tabSettings
	tabMappings
)

var tabNames = []string{"Dashboard", "Settings", "Mapped"}

type logMsg string

type tickMsg time.Time

type ps4Msg bool

type Model struct {
	cfg     *config.Config
	version string

	tabIdx tab

	svc *rpcsvc.Service
	bot *bot.Bot

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

	ticks     int
	botOnline bool
	ps4Online bool
	status    string
	err       string

	fingerprint string
	appliedRPC  rpcKey
	appliedBot  config.Bot

	width  int
	height int
	body   int
	ready  bool
}

func New(cfg *config.Config, version string, startTab int) *Model {
	logCh := make(chan string, 256)
	logf := func(format string, args ...any) {
		select {
		case logCh <- fmt.Sprintf(format, args...):
		default:
		}
	}

	in := textinput.New()
	in.Prompt = "› "
	in.CharLimit = 256

	m := &Model{
		cfg:     cfg,
		version: version,
		svc:     rpcsvc.New(cfg, logf),
		logCh:   logCh,
		fields:  settingsFields(),
		input:   in,
		vp:      viewport.New(80, 10),
	}
	m.cursor = m.firstSelectable()
	m.fingerprint = config.Fingerprint()
	if startTab >= 0 && startTab <= int(tabMappings) {
		m.tabIdx = tab(startTab)
	}
	redirectStdLog(logCh)
	return m
}

func (m *Model) Init() tea.Cmd {
	m.reconcile()
	return tea.Batch(m.listenLogs(), tickCmd(), m.probePS4())
}

func (m *Model) listenLogs() tea.Cmd {
	return func() tea.Msg { return logMsg(<-m.logCh) }
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (m *Model) probePS4() tea.Cmd {
	ip := m.cfg.Var.IP
	return func() tea.Msg {
		if ip == "" {
			return ps4Msg(false)
		}
		online, _ := ps4.CheckPS4(ip)
		return ps4Msg(online)
	}
}

func (m *Model) setPS4Online(online bool) {
	if online == m.ps4Online {
		return
	}
	m.ps4Online = online
	if online {
		m.log("ps4: %s is online", m.cfg.Var.IP)
	} else {
		m.log("ps4: %s is unreachable", m.cfg.Var.IP)
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
		m.logs = append(m.logs, line)
	}
	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
	m.vp.SetContent(strings.Join(m.logs, "\n"))
	m.vp.GotoBottom()
}

func (m *Model) startRPC() {
	if m.svc.Status().Running {
		return
	}
	if m.cfg.Var.IP == "" {
		m.err = "set the console IP first"
		return
	}
	if err := m.svc.Start(context.Background()); err != nil {
		m.err = err.Error()
	}
}

func (m *Model) stopRPC() {
	m.log("rpc: stopping")
	m.svc.Stop()
}

func (m *Model) startBot() {
	if m.botOnline {
		return
	}
	if m.cfg.Bot.Token == "" {
		m.err = "bot token is not set"
		return
	}
	b, err := bot.New(m.cfg.Bot, m.cfg.Var.IP)
	if err != nil {
		m.err = "bot: " + err.Error()
		return
	}
	if err := b.Start(); err != nil {
		m.err = "bot: " + err.Error()
		return
	}
	m.bot = b
	m.botOnline = true
	m.log("bot: started")
}

func (m *Model) stopBot() {
	if m.bot != nil {
		m.bot.Stop()
		m.bot = nil
	}
	m.botOnline = false
	m.log("bot: stopped")
}

func (m *Model) Shutdown() {
	m.svc.Stop()
	if m.bot != nil {
		m.bot.Stop()
		m.bot = nil
	}
}
