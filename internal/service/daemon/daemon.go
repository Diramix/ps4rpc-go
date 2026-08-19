package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"ps4rpc/internal/app/autostart"
	"ps4rpc/internal/app/config"
	"ps4rpc/internal/service/ipc"
)

const (
	RolePresence = "presence"
	RoleBot      = "bot"
)

var Roles = []string{RolePresence, RoleBot}

const (
	spawnTimeout = 5 * time.Second
	spawnPoll    = 50 * time.Millisecond

	shutdownTimeout = 30 * time.Second
)

func Running(role string) bool { return ipc.Ping(role) }

func Spawn(role string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(ipc.RuntimeDir(), 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(filepath.Join(ipc.RuntimeDir(), role+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}

	cmd := detachedCommand(exe, role)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
		logFile.Close()
	}()
	return nil
}

func Ensure(role string) error {
	reconcileMu.Lock()
	defer reconcileMu.Unlock()
	return ensure(role)
}

func ensure(role string) error {
	if Running(role) {
		return nil
	}
	if err := Spawn(role); err != nil {
		return fmt.Errorf("daemon: spawn %s: %w", role, err)
	}
	deadline := time.Now().Add(spawnTimeout)
	for time.Now().Before(deadline) {
		if Running(role) {
			return nil
		}
		time.Sleep(spawnPoll)
	}
	return fmt.Errorf("daemon: %s did not come up, see %s",
		role, filepath.Join(ipc.RuntimeDir(), role+".log"))
}

func Wanted(cfg *config.Config, role string) bool {
	switch role {
	case RolePresence:
		return cfg.Core.Enabled && cfg.Core.IP != ""
	case RoleBot:
		return cfg.Bot.Enabled && cfg.Bot.Token != "" && cfg.Core.IP != ""
	}
	return false
}

var reconcileMu sync.Mutex

func EnsureEnabled() error {
	reconcileMu.Lock()
	defer reconcileMu.Unlock()

	config.SetDir(config.DefaultDir())
	cfg, _, err := config.Load()
	if err != nil {
		return fmt.Errorf("daemon: config: %w", err)
	}

	firstErr := autostart.Sync(cfg.Core.Autostart)

	for _, role := range Roles {
		var err error
		if Wanted(cfg, role) {
			err = startRole(role)
		} else {
			err = stopRole(role)
		}
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func startRole(role string) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = ensure(role); err != nil {
			return err
		}
		if err = Notify(role, ipc.MethodReload); err == nil && Running(role) {
			return nil
		}
	}
	if err == nil {
		err = fmt.Errorf("daemon: %s keeps stopping right after start", role)
	}
	return err
}

func stopRole(role string) error {
	if !Running(role) {
		return nil
	}
	return shutdown(role)
}

func Notify(role, method string) error {
	c, err := ipc.Dial(role)
	if err != nil {
		return nil
	}
	defer c.Close()
	return c.Call(method, nil, nil)
}
