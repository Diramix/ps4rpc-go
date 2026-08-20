package tui

import (
	"strings"
	"testing"
	"time"

	"ps4rpc/internal/app/config"

	tea "github.com/charmbracelet/bubbletea"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("PS4RPC_RUNTIME_DIR", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	config.SetDir(t.TempDir())
	cfg := config.Default()
	cfg.Core.IP = "127.0.0.1"
	cfg.Bot.Token = "secret-token"
	cfg.Mapped = []config.Mapped{{TitleID: "CUSA10249", Name: "Bloodborne", Image: "icon.png"}}

	m := New(cfg, "test", 0)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 34})
	return m
}

func key(s string) tea.KeyMsg {
	if r := []rune(s); len(r) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: r}
	}
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "ctrl+s":
		return tea.KeyMsg{Type: tea.KeyCtrlS}
	}
	panic("unknown key " + s)
}

func send(m *Model, keys ...string) {
	for _, k := range keys {
		m.Update(key(k))
	}
}

func focusField(t *testing.T, m *Model, label string) {
	t.Helper()
	for i, f := range m.fields {
		if f.label == label {
			m.cursor = i
			return
		}
	}
	t.Fatalf("no settings field labelled %q", label)
}

func TestViewRendersEveryTab(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()

	m.err = "something went wrong"

	sizes := []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 100, Height: 34}, {Width: 200, Height: 60}}
	for _, size := range sizes {
		m.Update(size)
		for i := range tabNames {
			m.tabIdx = tab(i)
			out := m.View()
			if !strings.Contains(out, "PS4RPC") {
				t.Errorf("%dx%d tab %d: header missing", size.Width, size.Height, i)
			}
			lines := strings.Split(out, "\n")
			if len(lines) != size.Height {
				t.Errorf("%dx%d tab %d: got %d lines, want exactly %d",
					size.Width, size.Height, i, len(lines), size.Height)
			}
			for _, line := range lines {
				if got := len([]rune(line)); got > size.Width {
					t.Errorf("%dx%d tab %d: line too wide (%d): %q", size.Width, size.Height, i, got, line)
				}
			}
		}
	}
}

func TestTokenIsMaskedUntilRevealed(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()
	m.tabIdx = tabSettings

	if strings.Contains(m.View(), "secret-token") {
		t.Fatal("token rendered in clear by default")
	}
	m.secret = true
	if !strings.Contains(m.View(), "secret-token") {
		t.Fatal("token not revealed after toggling")
	}
}

func TestToggleIsSavedImmediately(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()
	m.tabIdx = tabSettings

	focusField(t, m, "Rich Presence")
	send(m, " ")
	if m.cfg.Core.Enabled {
		t.Error("toggle did not flip the value")
	}
	if m.err != "" {
		t.Fatalf("apply reported an error: %q", m.err)
	}

	saved, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Core.Enabled {
		t.Error("toggle was not written to disk without ctrl+s")
	}
}

func TestStatusFallsBackToStoppedWithoutDaemons(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()

	m.refreshStatus()
	if m.rpcStatus.Running || m.botStatus.Running {
		t.Fatal("status reported as running while no daemon is listening")
	}
}

func TestExternalConfigEditIsPickedUp(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()

	onDisk := m.cfg.Clone()
	onDisk.Core.IP = "10.9.9.9"
	onDisk.Core.Enabled = false
	if err := onDisk.Save(); err != nil {
		t.Fatal(err)
	}

	msg := m.watchConfig()()
	cm, ok := msg.(configMsg)
	if !ok {
		t.Fatalf("watcher did not report the change, got %T", msg)
	}
	m.onExternalConfig(cm.cfg)

	if m.cfg.Core.IP != "10.9.9.9" || m.cfg.Core.Enabled {
		t.Fatalf("config not reloaded: %+v", m.cfg.Core)
	}
	if msg := m.watchConfig()(); msg != nil {
		t.Errorf("watcher fired on an unchanged config: %T", msg)
	}
}

func TestInvalidValueIsRejected(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()
	m.tabIdx = tabSettings
	m.cursor = m.firstSelectable()

	send(m, "enter")
	if !m.editing {
		t.Fatal("enter did not start editing")
	}
	m.input.SetValue("")
	send(m, "enter")
	if !m.editing || m.err == "" {
		t.Fatalf("empty IP should be rejected: editing=%v err=%q", m.editing, m.err)
	}

	m.input.SetValue("10.0.0.9")
	send(m, "enter")
	if m.editing || m.cfg.Core.IP != "10.0.0.9" {
		t.Fatalf("valid IP not applied: %q", m.cfg.Core.IP)
	}
}

func TestMappingRowAddEditDelete(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()
	m.tabIdx = tabMappings

	send(m, "a")
	if len(m.cfg.Mapped) != 2 {
		t.Fatalf("add failed: %d rows", len(m.cfg.Mapped))
	}
	send(m, "enter")
	m.input.SetValue("CUSA00001")
	send(m, "enter")
	if m.cfg.Mapped[1].TitleID != "CUSA00001" {
		t.Fatalf("cell edit failed: %+v", m.cfg.Mapped[1])
	}
	send(m, "d")
	if len(m.cfg.Mapped) != 1 {
		t.Fatalf("delete failed: %d rows", len(m.cfg.Mapped))
	}

	saved, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Mapped) != 1 {
		t.Errorf("mapped rows not persisted: %+v", saved.Mapped)
	}
}

func TestTickSchedulesTheConfigWatcher(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()

	edited := m.cfg.Clone()
	edited.Core.IP = "10.1.2.3"
	if err := edited.Save(); err != nil {
		t.Fatal(err)
	}

	_, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("tick returned no commands")
	}
	msgs := collect(cmd)
	for _, msg := range msgs {
		if cm, ok := msg.(configMsg); ok {
			m.onExternalConfig(cm.cfg)
			if m.cfg.Core.IP != "10.1.2.3" {
				t.Fatalf("reload did not apply: %+v", m.cfg.Core)
			}
			return
		}
	}
	t.Fatalf("no configMsg among %d commands from the tick", len(msgs))
}

func collect(cmd tea.Cmd) []tea.Msg {
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var out []tea.Msg
	for _, c := range batch {
		if c == nil {
			continue
		}
		out = append(out, collect(c)...)
	}
	return out
}

func TestShortcutsWorkOnACyrillicLayout(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()
	m.tabIdx = tabMappings

	before := len(m.cfg.Mapped)
	send(m, "ф")
	if len(m.cfg.Mapped) != before+1 {
		t.Fatalf("ф did not act as a: %d rows", len(m.cfg.Mapped))
	}
	send(m, "в")
	if len(m.cfg.Mapped) != before {
		t.Fatalf("в did not act as d: %d rows", len(m.cfg.Mapped))
	}

	for in, want := range map[string]string{"ф": "a", "Ф": "A", "с": "c", "й": "q", "ы": "s"} {
		if got := keyName(key(in)); got != want {
			t.Errorf("keyName(%q) = %q, want %q", in, got, want)
		}
	}
	if got := keyName(key("a")); got != "a" {
		t.Errorf("keyName(\"a\") = %q", got)
	}
}

func TestTypingCyrillicIsNotRemapped(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()
	m.tabIdx = tabSettings
	m.cursor = m.firstSelectable()

	send(m, "enter")
	m.input.SetValue("")
	send(m, "ф")
	if got := m.input.Value(); got != "ф" {
		t.Fatalf("edit mode swallowed the rune: %q", got)
	}
}

func TestAutostartToggleIsPersisted(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()
	m.tabIdx = tabSettings

	if !m.cfg.Core.Autostart {
		t.Fatal("autostart should start out on")
	}
	focusField(t, m, "Autostart")
	send(m, " ")
	if m.cfg.Core.Autostart {
		t.Fatal("toggle did not flip the value")
	}
	if m.err != "" {
		t.Fatalf("apply reported an error: %q", m.err)
	}

	saved, _, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if saved.Core.Autostart {
		t.Fatal("autostart was not written to disk")
	}
}
