package daemon

import (
	"fmt"
	"time"

	"ps4rpc/internal/service/ipc"
)

func PresenceState() (st PresenceStatus, ok bool) {
	return query[PresenceStatus](RolePresence)
}

func BotState() (st BotStatus, ok bool) {
	return query[BotStatus](RoleBot)
}

func query[T any](role string) (out T, ok bool) {
	c, err := ipc.Dial(role)
	if err != nil {
		return out, false
	}
	defer c.Close()
	if err := c.Call(ipc.MethodStatus, nil, &out); err != nil {
		return out, false
	}
	return out, true
}

func Shutdown(role string) error {
	reconcileMu.Lock()
	defer reconcileMu.Unlock()
	return shutdown(role)
}

func shutdown(role string) error {
	if err := Notify(role, ipc.MethodShutdown); err != nil {
		return err
	}
	deadline := time.Now().Add(shutdownTimeout)
	for time.Now().Before(deadline) {
		if !Running(role) {
			return nil
		}
		time.Sleep(spawnPoll)
	}
	return fmt.Errorf("daemon: %s is still winding down", role)
}
