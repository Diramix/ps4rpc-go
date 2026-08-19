package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"runtime/debug"

	"ps4rpc/internal/cli"
	"ps4rpc/internal/config"
	"ps4rpc/internal/daemon"
	"ps4rpc/internal/tui"
)

var version = "dev"

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			rev := s.Value
			if len(rev) > 7 {
				rev = rev[:7]
			}
			return "dev-" + rev
		}
	}
	return version
}

func main() {
	version = resolveVersion()

	args := os.Args[1:]
	for _, arg := range args {
		switch arg {
		case "--version", "-v":
			fmt.Println(version)
			return
		case "--help", "-h":
			cli.PrintHelp(os.Stdout, version)
			return
		case "--config", "-c":
			openConfigDir()
			return
		}
	}

	command := ""
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case daemon.RolePresence, "--headless", "-headless":
		exit(daemon.RunPresence())
	case daemon.RoleBot:
		exit(daemon.RunBot())
	case "start":
		exit(startDaemons())
	case "stop":
		exit(stopDaemons(args[1:]))
	case "status":
		printStatus()
	case "":
		runTUI()
	default:
		cli.Fail(os.Stderr, fmt.Errorf("unknown command %q, try --help", command))
		os.Exit(2)
	}
}

func exit(err error) {
	if err != nil {
		cli.Fail(os.Stderr, err)
		os.Exit(1)
	}
}

func startDaemons() error {
	if _, err := loadConfig(); err != nil {
		return err
	}
	if err := daemon.EnsureEnabled(); err != nil {
		return err
	}
	printStatus()
	return nil
}

func stopDaemons(roles []string) error {
	if len(roles) == 0 {
		roles = daemon.Roles
	}
	for _, role := range roles {
		if err := daemon.Shutdown(role); err != nil {
			return err
		}
		fmt.Println(cli.DoneLine(role, "stopped"))
	}
	return nil
}

func printStatus() {
	if st, ok := daemon.PresenceState(); ok {
		game := st.GameName
		if game == "" {
			game = "-"
		}
		fmt.Println(cli.StatusLine(daemon.RolePresence, st.Running, runState(st.Running),
			cli.Flag("ps4", st.PS4Online), cli.Flag("discord", st.DiscordOK), cli.KV("game", game)))
	} else {
		fmt.Println(cli.StatusLine(daemon.RolePresence, false, "not running"))
	}

	if st, ok := daemon.BotState(); ok {
		fmt.Println(cli.StatusLine(daemon.RoleBot, st.Running, runState(st.Running),
			cli.Flag("token", st.HasToken)))
	} else {
		fmt.Println(cli.StatusLine(daemon.RoleBot, false, "not running"))
	}
}

func runState(running bool) string {
	if running {
		return "running"
	}
	return "idle"
}

func loadConfig() (*config.Config, error) {
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

func runTUI() {
	cfg, err := loadConfig()
	if err != nil {
		cli.Fail(os.Stderr, err)
		os.Exit(1)
	}

	startTab := 0
	if cfg.Core.IP == "" {
		startTab = 1
	}

	if err := daemon.EnsureEnabled(); err != nil {
		cli.Fail(os.Stderr, err)
	}

	if !tui.IsTTY() {
		printStatus()
		return
	}
	if err := tui.Run(cfg, version, startTab); err != nil {
		cli.Fail(os.Stderr, fmt.Errorf("tui: %w", err))
		os.Exit(1)
	}
}

func openConfigDir() {
	dir := config.DefaultDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		cli.Fail(os.Stderr, err)
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
		cli.Fail(os.Stderr, err)
	}
}
