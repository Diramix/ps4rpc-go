package bot

import (
	"context"
	"testing"
	"time"

	"ps4rpc/internal/app/config"
	"ps4rpc/internal/source/ps4"
)

func TestWarmReportsAnOfflineConsole(t *testing.T) {
	b := &Bot{info: ps4.New("127.0.0.1")}
	if b.warm(context.Background()) {
		t.Fatal("warm reported success without a console")
	}
}

func TestRefreshEveryFallsBackToTheDefault(t *testing.T) {
	b := &Bot{cache: config.Cache{Refresh: 0}}
	if got := b.refreshEvery(); got != defaultRefresh {
		t.Errorf("refreshEvery = %s", got)
	}
	b.cache.Refresh = 5
	if got := b.refreshEvery(); got != 5*time.Minute {
		t.Errorf("refreshEvery = %s", got)
	}
}

func TestWarmLoopRetriesQuicklyWhileOffline(t *testing.T) {
	b := &Bot{info: ps4.New("127.0.0.1"), cache: config.Cache{Refresh: 60}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		b.warmLoop(ctx)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("warmLoop did not stop on context cancel")
	}
	if probeEvery >= defaultRefresh {
		t.Errorf("the offline probe (%s) is not shorter than the refresh period (%s)", probeEvery, defaultRefresh)
	}
}
