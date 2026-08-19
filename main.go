package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"ps4rpc/internal/service/daemon"
	"ps4rpc/internal/ui/cli"
	"ps4rpc/internal/ui/tui"
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
			cli.OpenConfigDir()
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
		exit(cli.Start())
	case "stop":
		exit(cli.Stop(args[1:]))
	case "status":
		cli.PrintStatus(os.Stdout)
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

func runTUI() {
	cfg, err := cli.LoadConfig()
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
		cli.PrintStatus(os.Stdout)
		return
	}
	if err := tui.Run(cfg, version, startTab); err != nil {
		cli.Fail(os.Stderr, fmt.Errorf("tui: %w", err))
		os.Exit(1)
	}
}
