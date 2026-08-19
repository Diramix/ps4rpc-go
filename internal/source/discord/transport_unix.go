//go:build !windows

package discord

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func dialIPC() (net.Conn, error) {
	base := ipcBase()
	var lastErr error
	for i := 0; i < 10; i++ {
		path := filepath.Join(base, fmt.Sprintf("discord-ipc-%d", i))
		conn, err := net.DialTimeout("unix", path, 3*time.Second)
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("no discord-ipc socket found")
	}
	return nil, lastErr
}

func ipcBase() string {
	for _, env := range []string{"XDG_RUNTIME_DIR", "TMPDIR", "TMP", "TEMP"} {
		if v := os.Getenv(env); v != "" {
			if candidate := firstExisting(v); candidate != "" {
				return candidate
			}
			return v
		}
	}
	return "/tmp"
}

func firstExisting(base string) string {
	for _, sub := range []string{
		"",
		"app/com.discordapp.Discord",
		"snap.discord",
	} {
		dir := filepath.Join(base, sub)
		if _, err := os.Stat(filepath.Join(dir, "discord-ipc-0")); err == nil {
			return dir
		}
	}
	return ""
}
