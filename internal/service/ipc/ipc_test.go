package ipc

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func serveTest(t *testing.T, name string, h Handler) *Server {
	t.Helper()
	t.Setenv("PS4RPC_RUNTIME_DIR", t.TempDir())
	srv, err := Serve(name, h)
	if err != nil {
		t.Fatalf("serve: %v", err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func TestCallRoundTrip(t *testing.T) {
	serveTest(t, "role", func(method string, params json.RawMessage) (any, error) {
		switch method {
		case "echo":
			var s string
			if err := json.Unmarshal(params, &s); err != nil {
				return nil, err
			}
			return s + "!", nil
		case "boom":
			return nil, errors.New("exploded")
		}
		return nil, errors.New("unknown")
	})

	c, err := Dial("role")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var out string
	if err := c.Call("echo", "hi", &out); err != nil {
		t.Fatal(err)
	}
	if out != "hi!" {
		t.Fatalf("got %q, want %q", out, "hi!")
	}

	if err := c.Call("boom", nil, nil); err == nil || err.Error() != "exploded" {
		t.Fatalf("handler error not propagated: %v", err)
	}
	if err := c.Call(MethodPing, nil, nil); err != nil {
		t.Fatalf("ping: %v", err)
	}
}

func TestBroadcastReachesSubscribers(t *testing.T) {
	srv := serveTest(t, "role", func(string, json.RawMessage) (any, error) { return nil, nil })

	c, err := Dial("role")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.Call(MethodSubscribe, nil, nil); err != nil {
		t.Fatal(err)
	}

	srv.Broadcast(EventLog, "a line")
	select {
	case ev := <-c.Events():
		if ev.Name != EventLog || ev.Str() != "a line" {
			t.Fatalf("unexpected event %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event received")
	}
}

func TestServeReplacesAStaleSocket(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PS4RPC_RUNTIME_DIR", dir)

	stale := filepath.Join(dir, "role.sock")
	if err := os.WriteFile(stale, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	srv, err := Serve("role", func(string, json.RawMessage) (any, error) { return nil, nil })
	if err != nil {
		t.Fatalf("stale socket was not cleaned up: %v", err)
	}
	defer srv.Close()

	if !Ping("role") {
		t.Fatal("server does not answer after taking over the address")
	}
}

func TestServeRefusesASecondInstance(t *testing.T) {
	serveTest(t, "role", func(string, json.RawMessage) (any, error) { return nil, nil })
	if _, err := Serve("role", nil); err == nil {
		t.Fatal("a second daemon was allowed to bind the same address")
	}
}

func TestCallFailsAfterTheServerGoesAway(t *testing.T) {
	srv := serveTest(t, "role", func(string, json.RawMessage) (any, error) { return nil, nil })

	c, err := Dial("role")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = srv.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.Call(MethodPing, nil, nil); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("call still succeeds after the server closed")
}
