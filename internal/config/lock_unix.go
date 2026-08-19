//go:build !windows

package config

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

type fileLock struct{ f *os.File }

func acquire() (*fileLock, error) {
	if err := os.MkdirAll(Dir, 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(Dir, ".lock"), os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &fileLock{f: f}, nil
}

func (l *fileLock) release() {
	if l == nil {
		return
	}
	_ = unix.Flock(int(l.f.Fd()), unix.LOCK_UN)
	_ = l.f.Close()
}
