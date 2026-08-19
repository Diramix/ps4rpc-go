package autostart

import (
	"os"
	"path/filepath"
)

const appName = "ps4rpc"

func Sync(on bool) error {
	if !on {
		return Set(false)
	}
	if cur, err := registered(); err == nil && cur == exePath() {
		return nil
	}
	return Set(true)
}

func exePath() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		return resolved
	}
	return exe
}
