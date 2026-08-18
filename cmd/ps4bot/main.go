package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ps4rpc/internal/bot"
	"ps4rpc/internal/config"
)

var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	ipFlag := flag.String("ip", "", "override PS4 IP address")
	flag.Parse()
	if *showVersion {
		fmt.Println(version)
		return
	}

	config.SetDir(config.DefaultDir())
	cfg, existed, err := config.Load()
	if err != nil {
		log.Printf("config: %v", err)
	}
	if !existed {
		log.Printf("config: no config found in %s, writing defaults; set bot.token and var.ip", config.Dir)
		_ = cfg.Save()
	}

	ip := cfg.Var.IP
	if *ipFlag != "" {
		ip = *ipFlag
	}

	b, err := bot.New(cfg.Bot, ip)
	if err != nil {
		log.Fatalf("bot: %v", err)
	}
	if err := b.Start(); err != nil {
		log.Fatalf("bot start: %v", err)
	}
	defer b.Stop()

	log.Printf("ps4bot %s running (PS4 %s). Ctrl+C to stop.", version, ip)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("shutting down")
}
