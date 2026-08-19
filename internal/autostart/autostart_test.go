//go:build !windows && !darwin

package autostart

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func TestSetWritesADesktopEntryPointingAtThisBinary(t *testing.T) {
	isolate(t)

	if on, err := Enabled(); err != nil || on {
		t.Fatalf("Enabled() = %v, %v; want false, nil", on, err)
	}
	if err := Set(true); err != nil {
		t.Fatalf("Set(true): %v", err)
	}

	on, err := Enabled()
	if err != nil || !on {
		t.Fatalf("Enabled() = %v, %v; want true, nil", on, err)
	}

	raw, err := os.ReadFile(Location())
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "Exec=") || !strings.Contains(body, " start\n") {
		t.Fatalf("entry does not run the start command:\n%s", body)
	}
	got, err := registered()
	if err != nil {
		t.Fatalf("registered(): %v", err)
	}
	if got != exePath() {
		t.Fatalf("registered() = %q, want %q", got, exePath())
	}
}

func TestSyncIsIdempotentAndRepairsAStaleEntry(t *testing.T) {
	isolate(t)

	if err := Sync(true); err != nil {
		t.Fatalf("Sync(true): %v", err)
	}
	first, err := os.ReadFile(Location())
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if err := Sync(true); err != nil {
		t.Fatalf("Sync(true) again: %v", err)
	}
	second, err := os.ReadFile(Location())
	if err != nil {
		t.Fatalf("read entry: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("Sync rewrote an up-to-date entry")
	}

	stale := strings.Replace(string(second), exePath(), filepath.Join(t.TempDir(), "old-ps4rpc"), 1)
	if err := os.WriteFile(Location(), []byte(stale), 0o644); err != nil {
		t.Fatalf("write stale entry: %v", err)
	}
	if err := Sync(true); err != nil {
		t.Fatalf("Sync(true) on stale: %v", err)
	}
	got, err := registered()
	if err != nil {
		t.Fatalf("registered(): %v", err)
	}
	if got != exePath() {
		t.Fatalf("stale entry not repaired: got %q, want %q", got, exePath())
	}
}

func TestSetFalseRemovesTheEntryAndIsSafeToRepeat(t *testing.T) {
	isolate(t)

	if err := Set(true); err != nil {
		t.Fatalf("Set(true): %v", err)
	}
	if err := Set(false); err != nil {
		t.Fatalf("Set(false): %v", err)
	}
	if on, err := Enabled(); err != nil || on {
		t.Fatalf("Enabled() = %v, %v; want false, nil", on, err)
	}
	if err := Set(false); err != nil {
		t.Fatalf("Set(false) twice: %v", err)
	}
	if err := Sync(false); err != nil {
		t.Fatalf("Sync(false): %v", err)
	}
}
