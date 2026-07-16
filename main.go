package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"ps4rpc/internal/config"
	"ps4rpc/internal/discord"
	"ps4rpc/internal/ps4"
	"ps4rpc/internal/tmdb"
)

var version = "dev"

var (
	cfg   *config.Config
	rpc   *discord.Client
	timer = time.Now().Unix()
)

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
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Println(version)
			return
		}
	}

	readConfig()
	rpc = discord.ConnectRetry(clientID(cfg.Var.ClientID))

	prevTitleID := ""
	devAppChanged := false
	for {
		titleID, gameType, ok := ps4.GetTitleID(cfg.Var.IP)
		if !ok {
			fmt.Println("get_title_id():  PS4 not found, sleeping")
			for !ps4.TestForPS4(cfg.Var.IP) {
				sleepFor()
			}
			continue
		}
		fmt.Printf("get_title_id():  (%q, %q)\n", titleID, gameType)

		if prevTitleID == titleID {
			fmt.Println("reusing previous presence data")
		} else {
			name, image := checkMapped(titleID, gameType)
			prevTitleID = titleID

			if cfg.Var.UseDevapps {
				devAppChanged = changeDevApp(titleID, devAppChanged)
			}

			if titleID == "main_menu" {
				_ = rpc.Clear()
			} else {
				_ = rpc.Close()
				rpc = discord.ConnectRetry(clientID(cfg.Var.ClientID))
				updatePresence(name, image, titleID)
			}
		}
		fmt.Println()
		time.Sleep(time.Duration(cfg.Var.WaitTime) * time.Second)
	}
}

func readConfig() {
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
				fmt.Printf("readConfig():   ps4 sleeping on '%s', waiting %d seconds\n", cfg.Var.IP, sleepSeconds())
				time.Sleep(time.Duration(sleepSeconds()) * time.Second)
			}
		}
	}
}

func checkMapped(titleID, gameType string) (name, image string) {
	image = titleID
	for _, m := range cfg.Mapped {
		if m.TitleID == titleID {
			fmt.Printf("check_mapped():  (%q, %q)\n", m.Name, m.Image)
			return m.Name, m.Image
		}
	}
	if titleID == "main_menu" {
		return "", titleID
	}

	fmt.Println("check_mapped():  game has not been mapped yet.")
	switch gameType {
	case "PS4":
		name, image = tmdb.GetPS4GameInfo(titleID)
	case "PS1/2":
		name, image = tmdb.GetClassicGameInfo(titleID, cfg.Var.RetroCovers)
	default:
		name, image = tmdb.GetOtherGameInfo(titleID)
	}
	_ = cfg.AppendMapped(config.Mapped{TitleID: titleID, Name: name, Image: image})
	return name, image
}

func changeDevApp(titleID string, changed bool) bool {
	for _, app := range cfg.Devapps {
		if app.TitleID == titleID && app.TitleID != "" {
			fmt.Println("change_dev_app():    changing to new developer app")
			_ = rpc.Close()
			rpc = discord.ConnectRetry(app.DevID)
			return true
		}
	}
	if changed {
		fmt.Println("change_dev_app():    reverting to default developer app")
		_ = rpc.Close()
		rpc = discord.ConnectRetry(clientID(cfg.Var.ClientID))
		return false
	}
	return changed
}

func updatePresence(name, image, titleID string) {
	a := discord.Activity{LargeImage: image, LargeText: titleID}
	if cfg.Var.UseAppname {
		a.Details = name
	} else {
		a.Name = name
	}
	if cfg.Var.ShowTimer {
		a.Start = timer
	}
	if err := rpc.Update(a); err != nil {
		fmt.Printf("Error with Discord: %v\n", err)
		rpc = discord.ConnectRetry(clientID(cfg.Var.ClientID))
	}
}

func sleepSeconds() int {
	if cfg.Var.Hibernate {
		return cfg.Var.HibernateTime
	}
	return cfg.Var.WaitTime
}

func sleepFor() {
	if cfg.Var.Hibernate {
		time.Sleep(time.Duration(cfg.Var.HibernateTime) * time.Second)
	} else {
		time.Sleep(time.Duration(cfg.Var.WaitTime) * time.Second)
	}
}

func clientID(id int64) string {
	return strconv.FormatInt(id, 10)
}
