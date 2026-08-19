package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPutGetRoundTrip(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	want := []byte("sqlite payload")
	if err := s.Put("/system_data/priv/mms/app.db", want); err != nil {
		t.Fatal(err)
	}
	got, fetched, ok := s.Get("/system_data/priv/mms/app.db")
	if !ok {
		t.Fatal("key not found right after Put")
	}
	if string(got) != string(want) {
		t.Errorf("data = %q", got)
	}
	if time.Since(fetched) > time.Minute {
		t.Errorf("fetched = %v", fetched)
	}
}

func TestSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("/user/appmeta/CUSA00512/icon0.png", []byte("png")); err != nil {
		t.Fatal(err)
	}

	again, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	data, _, ok := again.Get("/user/appmeta/CUSA00512/icon0.png")
	if !ok || string(data) != "png" {
		t.Fatalf("lost the entry after reopen: %q %v", data, ok)
	}
	if files, bytes := again.Stats(); files != 1 || bytes != 3 {
		t.Errorf("stats = %d files, %d bytes", files, bytes)
	}
}

func TestMissingKey(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := s.Get("/nope"); ok {
		t.Error("missing key reported as present")
	}
	if _, ok := s.Age("/nope"); ok {
		t.Error("age reported for a missing key")
	}
}

func TestBrokenIndexIsIgnored(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, blobDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, indexName), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatalf("Open on a broken index: %v", err)
	}
	if err := s.Put("/a", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := s.Get("/a"); !ok {
		t.Error("store unusable after a broken index")
	}
}

func TestNewest(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !s.Newest().IsZero() {
		t.Error("empty store reports a newest entry")
	}
	if err := s.Put("/a", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if s.Newest().IsZero() {
		t.Error("newest is zero after Put")
	}
}

func TestNilStoreIsInert(t *testing.T) {
	var s *Store
	if _, _, ok := s.Get("/a"); ok {
		t.Error("nil store returned data")
	}
	if err := s.Put("/a", []byte("x")); err != nil {
		t.Errorf("Put on a nil store: %v", err)
	}
}

func TestOpenPrunesOrphanedIndexEntries(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Put("/some/key", []byte("data")); err != nil {
		t.Fatal(err)
	}
	files, ok := s.index["/some/key"]
	if !ok {
		t.Fatal("missing index entry after Put")
	}
	if err := os.Remove(filepath.Join(dir, blobDir, files.File)); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := reopened.Get("/some/key"); ok {
		t.Error("Get returned data for an orphaned entry")
	}
	if _, ok := reopened.index["/some/key"]; ok {
		t.Error("orphaned entry not pruned from index")
	}
}

func TestBlobNameIsReadable(t *testing.T) {
	if got := blobName("/user/appmeta/CUSA00512/icon0.png"); !strings.HasPrefix(got, "user_appmeta_CUSA00512_icon0.png-") {
		t.Errorf("blobName = %q", got)
	}
}

func TestBlobNameAvoidsCollisions(t *testing.T) {
	a := blobName("/a/b")
	b := blobName("/a_b")
	if a == b {
		t.Errorf("blobName collision: %q == %q", a, b)
	}
}
