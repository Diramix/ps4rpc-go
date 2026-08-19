//go:build !windows

package ipc

import (
	"crypto/sha256"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

func RuntimeDir() string {
	if v := os.Getenv("PS4RPC_RUNTIME_DIR"); v != "" {
		return v
	}
	if v := os.Getenv("XDG_RUNTIME_DIR"); v != "" {
		return filepath.Join(v, "ps4rpc-go")
	}
	return filepath.Join(os.TempDir(), "ps4rpc-go-"+strconv.Itoa(os.Getuid()))
}

const maxSockPath = 100

func addr(name string) string {
	path := filepath.Join(RuntimeDir(), name+".sock")
	if len(path) <= maxSockPath {
		return path
	}
	sum := sha256.Sum256([]byte(RuntimeDir()))
	return filepath.Join(os.TempDir(), fmt.Sprintf("ps4rpc-%x-%s.sock", sum[:4], name))
}

func dial(name string) (net.Conn, error) {
	return net.DialTimeout("unix", addr(name), dialTimeout)
}

func listen(name string) (net.Listener, error) {
	if err := os.MkdirAll(RuntimeDir(), 0o700); err != nil {
		return nil, err
	}
	path := addr(name)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	if _, err := os.Stat(path); err == nil {
		if Ping(name) {
			return nil, fmt.Errorf("ipc: %s is already running", name)
		}
		_ = os.Remove(path)
	}
	return net.Listen("unix", path)
}

func cleanup(name string) {
	_ = os.Remove(addr(name))
}
