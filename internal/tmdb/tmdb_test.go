package tmdb

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"testing"
)

func TestSearchClassicEmbedded(t *testing.T) {
	name, ok := searchClassic("data/ps1_games.md", "SLUS01272")
	if !ok {
		t.Fatal("expected to find SLUS01272 in embedded PS1 list")
	}
	if name != "007 - THE WORLD IS NOT ENOUGH" {
		t.Fatalf("unexpected name %q", name)
	}
	if _, ok := searchClassic("data/ps2_games.md", "SLPS25527"); !ok {
		t.Fatal("expected to find SLPS25527 in embedded PS2 list")
	}
}

func TestHashMatchesPython(t *testing.T) {
	tid := "CUSA10249_00"
	mac := hmac.New(sha1.New, tmdbKey)
	mac.Write([]byte(tid))
	got := strings.ToUpper(hex.EncodeToString(mac.Sum(nil)))
	want := "A66FC3584B3B7071C47CC1EF7BB20A4A358E4575"
	if got != want {
		t.Fatalf("hash mismatch:\n got %s\nwant %s", got, want)
	}
}
