//go:build windows

package ipc

import (
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/Microsoft/go-winio"
)

func RuntimeDir() string {
	if v := os.Getenv("PS4RPC_RUNTIME_DIR"); v != "" {
		return v
	}
	return filepath.Join(os.TempDir(), "ps4rpc-go")
}

func addr(name string) string {
	return `\\.\pipe\ps4rpc-go-` + name
}

func dial(name string) (net.Conn, error) {
	timeout := dialTimeout
	return winio.DialPipe(addr(name), &timeout)
}

func listen(name string) (net.Listener, error) {
	if err := os.MkdirAll(RuntimeDir(), 0o700); err != nil {
		return nil, err
	}
	ln, err := winio.ListenPipe(addr(name), nil)
	if err != nil {
		if Ping(name) {
			return nil, fmt.Errorf("ipc: %s is already running", name)
		}
		return nil, err
	}
	return ln, nil
}

func cleanup(string) {}
