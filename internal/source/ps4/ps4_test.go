package ps4

import (
	"os"
	"testing"
)

func TestQueryLibrary(t *testing.T) {
	data, err := os.ReadFile("testdata/app.db")
	if err != nil {
		t.Fatal(err)
	}
	db, cleanup, err := openDB(data)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	games, err := queryLibrary(db)
	if err != nil {
		t.Fatal(err)
	}
	if len(games) == 0 {
		t.Fatal("no games parsed")
	}
	var found *Game
	for i := range games {
		if games[i].TitleID == "CUSA00512" {
			found = &games[i]
		}
	}
	if found == nil {
		t.Fatal("CUSA00512 not found")
	}
	if found.Name != "Beyond: Two Souls™" {
		t.Errorf("name = %q", found.Name)
	}
	if !found.IsGame() {
		t.Errorf("category %q not detected as game", found.Category)
	}
	if found.Size <= 0 {
		t.Errorf("size = %d", found.Size)
	}
	if found.LastPlayed.IsZero() {
		t.Errorf("last played not parsed")
	}
}

func TestQueryTrophyTitles(t *testing.T) {
	data, err := os.ReadFile("testdata/trophy_local.db")
	if err != nil {
		t.Fatal(err)
	}
	db, cleanup, err := openDB(data)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	titles, err := queryTrophyTitles(db)
	if err != nil {
		t.Fatal(err)
	}
	var tlou *TrophyTitle
	for i := range titles {
		if titles[i].CommID == "NPWR13348_00" {
			tlou = &titles[i]
		}
	}
	if tlou == nil {
		t.Fatal("TLOU2 title not found")
	}
	if tlou.Total != 28 || tlou.Platinum != 1 || tlou.Gold != 7 || tlou.Silver != 9 || tlou.Bronze != 11 {
		t.Errorf("counts wrong: %+v", tlou)
	}
}

func TestQueryTrophies(t *testing.T) {
	data, err := os.ReadFile("testdata/trophy_local.db")
	if err != nil {
		t.Fatal(err)
	}
	db, cleanup, err := openDB(data)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	tr, err := queryTrophies(db, "NPWR13348_00")
	if err != nil {
		t.Fatal(err)
	}
	if len(tr) != 28 {
		t.Fatalf("got %d trophies", len(tr))
	}
	var unlocked int
	var platinum int
	for _, x := range tr {
		if x.Unlocked {
			unlocked++
		}
		if x.Grade == GradePlatinum {
			platinum++
		}
		if x.Name == "" {
			t.Errorf("trophy %d has empty name", x.ID)
		}
	}
	if unlocked != 2 {
		t.Errorf("unlocked = %d, want 2", unlocked)
	}
	if platinum != 1 {
		t.Errorf("platinum count = %d, want 1", platinum)
	}
}
