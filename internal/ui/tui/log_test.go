package tui

import (
	"regexp"
	"strings"
	"testing"
)

var stamp = regexp.MustCompile(`^\d\d:\d\d:\d\d `)

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
