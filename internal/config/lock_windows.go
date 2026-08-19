//go:build windows

package config

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
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
	ol := new(windows.Overlapped)
	if err := windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, ol); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &fileLock{f: f}, nil
}

func (l *fileLock) release() {
	if l == nil {
		return
	}
	_ = windows.UnlockFileEx(windows.Handle(l.f.Fd()), 0, 1, 0, new(windows.Overlapped))
	_ = l.f.Close()
}
