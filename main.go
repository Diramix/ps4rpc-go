package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"ps4rpc/internal/cli"
	"ps4rpc/internal/config"
	"ps4rpc/internal/ps4"
	"ps4rpc/internal/rpcsvc"
	"ps4rpc/internal/tui"
)

var version = "dev"

var cfg *config.Config

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
	headless := false
	for _, arg := range os.Args[1:] {
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
		case "--headless", "-headless":
			headless = true
		}
	}

	if !headless && tui.IsTTY() {
		runTUI()
		return
	}
	runHeadless()
}

func runTUI() {
	config.SetDir(config.DefaultDir())
	c, existed, err := config.Load()
	if err != nil {
		fmt.Printf("config: %v\n", err)
	}
	cfg = c

	startTab := 0
	if !existed || cfg.Var.IP == "" {
		_ = cfg.Save()
		startTab = 1
	}

	if err := tui.Run(cfg, version, startTab); err != nil {
		fmt.Printf("tui: %v\n", err)
		os.Exit(1)
	}
}

func runHeadless() {
	readConfig()
	if cfg.Var.IP == "" {
		fmt.Println("readConfig():   no PS4 IP configured, nothing to do")
		os.Exit(1)
	}

	svc := rpcsvc.New(cfg, nil)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	svc.Run(ctx)
}

func openConfigDir() {
	dir := config.DefaultDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Printf("openConfigDir(): %v\n", err)
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
		fmt.Printf("openConfigDir(): %v\n", err)
	}
}

func readConfig() {
	config.SetDir(config.DefaultDir())
	c, existed, err := config.Load()
	if err != nil {
		fmt.Printf("readConfig():   error with config file: %v\n", err)
	}
	cfg = c
	if !existed {
		cfg.Var.IP = ps4.PromptUser()
		_ = cfg.Save()
		return
	}
	if !ps4.TestForPS4(cfg.Var.IP) {
		if cfg.Var.IP == "" {
			cfg.Var.IP = ps4.PromptUser()
			_ = cfg.Save()
		} else {
			for !ps4.TestForPS4(cfg.Var.IP) {
				fmt.Printf("readConfig():   ps4 sleeping on '%s', waiting %d seconds\n", cfg.Var.IP, cfg.Var.WaitTime)
				time.Sleep(time.Duration(cfg.Var.WaitTime) * time.Second)
			}
		}
	}
}
