package ps4

import (
	"testing"
	"time"

	"ps4rpc/internal/app/cache"
)

func TestUnreachableConsoleSkipsTheDial(t *testing.T) {
	store, err := cache.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Put(IconPath("CUSA00512"), []byte("png")); err != nil {
		t.Fatal(err)
	}
	c := New("127.0.0.1")
	c.SetCache(store)

	if _, _, err := c.Icon("CUSA00512"); err != nil {
		t.Fatal(err)
	}
	if !c.unreachable() {
		t.Fatal("a failed dial did not mark the console unreachable")
	}

	c.cache = map[string]cachedFile{}
	start := time.Now()
	if _, meta, err := c.Icon("CUSA00512"); err != nil || meta.Live {
		t.Fatalf("second read: meta=%+v err=%v", meta, err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("second read took %s, the dial was not skipped", elapsed)
	}
}
