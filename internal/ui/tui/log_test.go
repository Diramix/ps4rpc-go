package tui

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

var stamp = regexp.MustCompile(`^\d\d:\d\d:\d\d `)

var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

func stripANSI(s string) string { return ansiSeq.ReplaceAllString(s, "") }

func TestFormatLogSplitsTheSourceOffTheMessage(t *testing.T) {
	out := formatLog("ps4: 192.168.31.114 is online")
	if !stamp.MatchString(out) {
		t.Fatalf("no timestamp: %q", out)
	}
	rest := stamp.ReplaceAllString(out, "")
	if !strings.HasPrefix(rest, "ps4 ") || !strings.Contains(rest, "192.168.31.114 is online") {
		t.Fatalf("source not separated: %q", rest)
	}
	if strings.Contains(rest, "ps4:") {
		t.Fatalf("colon kept after the source: %q", rest)
	}
}

func TestFormatLogKeepsLinesWithoutASource(t *testing.T) {
	out := stamp.ReplaceAllString(formatLog("reusing previous presence data"), "")
	if out != "reusing previous presence data" {
		t.Fatalf("plain line was rewritten: %q", out)
	}
}

func TestSourceDetectionIgnoresProse(t *testing.T) {
	cases := map[string]bool{
		"ps4": true, "config": true, "rpc-svc": true,
		"": false, "two words": false, "Bloodborne": false,
		"a very long prefix": false, "12:30": false,
	}
	for in, want := range cases {
		if got := isSource(in); got != want {
			t.Errorf("isSource(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestLooksLikeValue(t *testing.T) {
	for _, s := range []string{"192.168.31.114", "6", "cusa10249", "560801"} {
		if !looksLikeValue(s) {
			t.Errorf("looksLikeValue(%q) = false", s)
		}
	}
	for _, s := range []string{"online", "started", "ps4rpc#2669", ""} {
		if looksLikeValue(s) {
			t.Errorf("looksLikeValue(%q) = true", s)
		}
	}
}

func TestLongLogLinesDoNotOverflowTheScreen(t *testing.T) {
	m := newTestModel(t)
	defer m.Shutdown()

	sizes := []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 100, Height: 30}, {Width: 200, Height: 50}}
	long := "bot: failed to register commands: HTTP 401 Unauthorized, the token in bot.lua was rejected by Discord, check it and restart the bot service"
	for _, size := range sizes {
		m.Update(size)
		for i := 0; i < 40; i++ {
			m.appendLog(long)
		}
		lines := strings.Split(m.View(), "\n")
		if len(lines) != size.Height {
			t.Errorf("%dx%d: view is %d lines, want exactly %d",
				size.Width, size.Height, len(lines), size.Height)
		}
		for i, line := range lines {
			if got := len([]rune(stripANSI(line))); got > size.Width {
				t.Errorf("%dx%d: line %d is %d cells wide: %q", size.Width, size.Height, i, got, line)
			}
		}
	}
}
