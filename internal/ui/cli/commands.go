package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"ps4rpc/internal/app/config"
	"ps4rpc/internal/service/daemon"
)

func LoadConfig() (*config.Config, error) {
	config.SetDir(config.DefaultDir())
	c, existed, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	if !existed {
		_ = c.Save()
	}
	return c, nil
}

func Start() error {
	if _, err := LoadConfig(); err != nil {
		return err
	}
	if err := daemon.EnsureEnabled(); err != nil {
		return err
	}
	PrintStatus(os.Stdout)
	return nil
}

func Stop(roles []string) error {
	if len(roles) == 0 {
		roles = daemon.Roles
	}
	for _, role := range roles {
		if err := daemon.Shutdown(role); err != nil {
			return err
		}
		fmt.Println(DoneLine(role, "stopped"))
	}
	return nil
}

func PrintStatus(w io.Writer) {
	if st, ok := daemon.PresenceState(); ok {
		game := st.GameName
		if game == "" {
			game = "-"
		}
		fmt.Fprintln(w, StatusLine(daemon.RolePresence, st.Running, runState(st.Running),
			Flag("ps4", st.PS4Online), Flag("discord", st.DiscordOK), KV("game", game)))
	} else {
		fmt.Fprintln(w, StatusLine(daemon.RolePresence, false, "not running"))
	}

	if st, ok := daemon.BotState(); ok {
		fmt.Fprintln(w, StatusLine(daemon.RoleBot, st.Running, runState(st.Running),
			Flag("token", st.HasToken)))
	} else {
		fmt.Fprintln(w, StatusLine(daemon.RoleBot, false, "not running"))
	}
}

func runState(running bool) string {
	if running {
		return "running"
	}
	return "idle"
}

func OpenConfigDir() {
	dir := config.DefaultDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		Fail(os.Stderr, err)
		return
	}
	fmt.Println(dir)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		Fail(os.Stderr, err)
	}
}
