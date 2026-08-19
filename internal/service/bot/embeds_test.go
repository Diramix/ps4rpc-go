package bot

import (
	"strings"
	"testing"
	"time"

	"ps4rpc/internal/source/ps4"
)

func sampleGames() []ps4.Game {
	return []ps4.Game{
		{TitleID: "CUSA00512", Name: "Beyond: Two Souls™", Category: "gd", Size: 37579763712, LastPlayed: time.Now().Add(-2 * time.Hour)},
		{TitleID: "CUSA01113", Name: "Gravity Rush™ Remastered", Category: "gd", Size: 9886642176, LastPlayed: time.Now().Add(-48 * time.Hour)},
	}
}

func TestStatusEmbedGame(t *testing.T) {
	st := ps4.Status{Online: true, State: ps4.StateGame, TitleID: "CUSA00512", SessionStart: time.Now().Add(-time.Hour)}
	r := statusEmbed(st, sampleGames(), "3h 21m", nil, "Beyond: Two Souls™", author{name: "diram1x"})
	if r.embed == nil || !strings.Contains(r.embed.Description, "Beyond") {
		t.Fatalf("bad status embed: %+v", r.embed)
	}
}

func TestLibraryEmbedPaging(t *testing.T) {
	r := libraryEmbed(sampleGames(), "size", 0)
	if r.embed.Description == "" {
		t.Fatal("empty library")
	}
	if !strings.Contains(r.embed.Footer.Text, "Page 1/1") {
		t.Errorf("footer = %q", r.embed.Footer.Text)
	}
}

func TestTrophyEmbed(t *testing.T) {
	title := ps4.TrophyTitle{CommID: "NPWR13348_00", Name: "The Last of Us™ Part II", Progress: 9, Unlocked: 2, Total: 28, Platinum: 1, Gold: 7, Silver: 9, Bronze: 11}
	trophies := []ps4.Trophy{
		{ID: 1, Grade: ps4.GradeGold, Unlocked: true, Name: "What I Had to Do", Description: "Complete the story", TimeUnlocked: time.Now()},
		{ID: 2, Grade: ps4.GradeBronze, Unlocked: false, Hidden: true, Name: "Secret", Description: "hidden"},
	}
	r := trophyEmbed(title, trophies, "all", 0, nil)
	if !strings.Contains(r.embed.Description, "🏆 1") {
		t.Errorf("missing counts: %q", r.embed.Description)
	}
	if !strings.Contains(r.embed.Description, "Hidden trophy") {
		t.Errorf("hidden trophy not masked: %q", r.embed.Description)
	}
	if !strings.Contains(r.embed.Description, "✅") || !strings.Contains(r.embed.Description, "🔒") {
		t.Errorf("missing unlock marks: %q", r.embed.Description)
	}
}

func TestTrophyFilterLocked(t *testing.T) {
	all := []ps4.Trophy{
		{ID: 1, Unlocked: true, Name: "a"},
		{ID: 2, Unlocked: false, Name: "b"},
	}
	if got := filterTrophies(all, "locked"); len(got) != 1 || got[0].ID != 2 {
		t.Fatalf("locked filter = %+v", got)
	}
	if got := filterTrophies(all, "unlocked"); len(got) != 1 || got[0].ID != 1 {
		t.Fatalf("unlocked filter = %+v", got)
	}
}
