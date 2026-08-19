package tui

import (
	"encoding/json"
	"testing"
	"time"

	"ps4rpc/internal/config"
	"ps4rpc/internal/daemon"
	"ps4rpc/internal/ipc"
)

func fakePresence(t *testing.T, st daemon.PresenceStatus, history []string) *ipc.Server {
	t.Helper()
	srv, err := ipc.Serve(daemon.RolePresence, func(method string, _ json.RawMessage) (any, error) {
		switch method {
		case ipc.MethodStatus:
			return st, nil
		case ipc.MethodLogs:
			return history, nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func TestModelPicksUpALiveDaemon(t *testing.T) {
	runtime := t.TempDir()
	t.Setenv("PS4RPC_RUNTIME_DIR", runtime)
	config.SetDir(t.TempDir())

	want := daemon.PresenceStatus{Running: true, PS4Online: true, GameName: "Bloodborne"}
	srv := fakePresence(t, want, []string{"rpc: started"})

	cfg := config.Default()
	cfg.Core.IP = "127.0.0.1"
	m := New(cfg, "test", 0)
	defer m.Shutdown()

	m.refreshStatus()
	if m.rpcStatus != want {
		t.Fatalf("status not taken from the daemon: %+v", m.rpcStatus)
	}
	if m.botStatus.Running {
		t.Error("bot reported as running while no bot daemon exists")
	}

	if got := drainLog(t, m); got != "rpc: started" {
		t.Fatalf("log history not pulled, got %q", got)
	}

	srv.Broadcast(ipc.EventLog, "rpc: connected")
	if got := drainLog(t, m); got != "rpc: connected" {
		t.Fatalf("broadcast not streamed into the UI, got %q", got)
	}
}

func TestStatusResetsWhenTheDaemonDisappears(t *testing.T) {
	t.Setenv("PS4RPC_RUNTIME_DIR", t.TempDir())
	config.SetDir(t.TempDir())

	srv := fakePresence(t, daemon.PresenceStatus{Running: true}, nil)

	cfg := config.Default()
	cfg.Core.IP = "127.0.0.1"
	m := New(cfg, "test", 0)
	defer m.Shutdown()

	m.refreshStatus()
	if !m.rpcStatus.Running {
		t.Fatal("status not picked up while the daemon was alive")
	}

	_ = srv.Close()
	m.refreshStatus()
	if m.rpcStatus.Running {
		t.Fatal("stale running status kept after the daemon went away")
	}
}

func drainLog(t *testing.T, m *Model) string {
	t.Helper()
	select {
	case line := <-m.logCh:
		return line
	case <-time.After(2 * time.Second):
		t.Fatal("no log line arrived")
		return ""
	}
}
