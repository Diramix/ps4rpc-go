package ps4

import (
	"testing"
	"time"

	"ps4rpc/internal/app/cache"
)

func offlineClient(t *testing.T) (*Client, *cache.Store) {
	t.Helper()
	store, err := cache.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c := New("127.0.0.1")
	c.SetCache(store)
	return c, store
}

func TestIconFallsBackToCache(t *testing.T) {
	c, store := offlineClient(t)
	if err := store.Put(IconPath("CUSA00512"), []byte("png bytes")); err != nil {
		t.Fatal(err)
	}

	data, meta, err := c.Icon("CUSA00512")
	if err != nil {
		t.Fatalf("Icon with an unreachable console: %v", err)
	}
	if string(data) != "png bytes" {
		t.Errorf("data = %q", data)
	}
	if meta.Live {
		t.Error("cached data reported as live")
	}
	if !meta.Stale() {
		t.Error("cached data not reported as stale")
	}
}

func TestDownloadWithoutCacheStillFails(t *testing.T) {
	c := New("127.0.0.1")
	if _, _, err := c.Icon("CUSA00512"); err == nil {
		t.Fatal("expected an error without a cache and without a console")
	}
}

func TestUsersFallBackToCache(t *testing.T) {
	c, store := offlineClient(t)
	if err := store.Put(usersKey, []byte(`["1234abcd"]`)); err != nil {
		t.Fatal(err)
	}
	acc, err := c.AccountID()
	if err != nil {
		t.Fatalf("AccountID offline: %v", err)
	}
	if acc != "1234abcd" {
		t.Errorf("account = %q", acc)
	}
}

func TestMetaMergePrefersStale(t *testing.T) {
	live := liveMeta()
	stale := Meta{Fetched: live.Fetched.Add(-time.Hour)}
	if got := live.Merge(stale); !got.Stale() {
		t.Error("merge lost the stale marker")
	}
	if got := stale.Merge(live); !got.Stale() {
		t.Error("merge lost the stale marker in the other direction")
	}
	if got := live.Merge(Meta{}); !got.Live {
		t.Error("merging with an empty meta dropped liveness")
	}
}
