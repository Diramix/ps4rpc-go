package config

import (
	"os"
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)

	if err := os.WriteFile(RpcPath, []byte(`return {
    client_id = "858345055966461973",
    wait_time = 15,
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(BotPath, []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(DevPath, []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(MappedPath, []byte(`return {
    { titleid = "CUSA10249", name = "The Last of Us, Part II = Remastered", image = "http://example/icon0.png" },
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path, []byte(`return {
    var = { ip = "192.168.31.114" },
    rpc = require("rpc"),
    bot = require("bot"),
    dev = require("dev"),
    mapped = require("mapped"),
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, existed, err := Load()
	if err != nil || !existed {
		t.Fatalf("load: existed=%v err=%v", existed, err)
	}
	if cfg.Core.IP != "192.168.31.114" || cfg.Core.WaitTime != 15 {
		t.Fatalf("var parsed wrong: %+v", cfg.Core)
	}
	if cfg.Core.ClientID != 858345055966461973 {
		t.Fatalf("client_id parsed wrong: %d", cfg.Core.ClientID)
	}
	if len(cfg.Mapped) != 1 || cfg.Mapped[0].Name != "The Last of Us, Part II = Remastered" {
		t.Fatalf("mapped parsed wrong: %+v", cfg.Mapped)
	}

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	cfg2, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Core != cfg.Core || cfg2.Mapped[0] != cfg.Mapped[0] {
		t.Fatalf("round-trip mismatch:\n%+v\n%+v", cfg, cfg2)
	}
}

func TestToggleDefaultsAndRoundTrip(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)

	write := func(path, body string) {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(RpcPath, "return { wait_time = 10 }\n")
	write(BotPath, "return {}\n")
	write(DevPath, `return { { devid = "1", titleid = "CUSA1" } }`+"\n")
	write(MappedPath, "return {}\n")
	write(Path, `return {
    var = { ip = "10.0.0.5" },
    rpc = require("rpc"),
    bot = require("bot"),
    dev = require("dev"),
    mapped = require("mapped"),
}
`)

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Core.Enabled {
		t.Errorf("rpc should default to enabled: %+v", cfg.Core)
	}
	if cfg.Bot.Enabled {
		t.Errorf("bot should default to disabled")
	}
	if len(cfg.Devapps) != 1 || cfg.Devapps[0].TitleID != "CUSA1" {
		t.Errorf("dev app parsed wrong: %+v", cfg.Devapps)
	}

	rpcFile, err := os.ReadFile(RpcPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"enabled ="} {
		if !strings.Contains(string(rpcFile), key) {
			t.Errorf("migrated rpc.lua missing %q\n%s", key, rpcFile)
		}
	}

	cfg.Core.Enabled = false
	cfg.Bot.Enabled = true
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	cfg2, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.Core != cfg.Core || cfg2.Bot != cfg.Bot || cfg2.Devapps[0] != cfg.Devapps[0] {
		t.Fatalf("toggle round-trip mismatch:\n%+v\n%+v", cfg, cfg2)
	}
}

func TestMigrateAddsMissingFiles(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)

	if err := os.WriteFile(RpcPath, []byte(`return {
    wait_time = 42,
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(Path, []byte(`return {
    var = { ip = "10.0.0.5" },
    rpc = require("rpc"),
    bot = require("bot"),
    dev = require("dev"),
    mapped = require("mapped"),
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(BotPath, []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(DevPath, []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(MappedPath, []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, existed, err := Load()
	if err != nil || !existed {
		t.Fatalf("load: existed=%v err=%v", existed, err)
	}
	if cfg.Core.IP != "10.0.0.5" || cfg.Core.WaitTime != 42 {
		t.Fatalf("user values not preserved: %+v", cfg.Core)
	}

	bot, err := os.ReadFile(BotPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"token =", "owner_id =", "guild_id =", "account_id ="} {
		if !strings.Contains(string(bot), key) {
			t.Errorf("migrated bot.lua missing %q\n%s", key, bot)
		}
	}

	before, _ := os.Stat(BotPath)
	if _, _, err := Load(); err != nil {
		t.Fatal(err)
	}
	after, _ := os.Stat(BotPath)
	if !before.ModTime().Equal(after.ModTime()) {
		t.Errorf("canonical file was rewritten on second load")
	}
}

func TestLegacyVarSectionIsMigratedToCore(t *testing.T) {
	dir := t.TempDir()
	SetDir(dir)

	if err := os.WriteFile(Path, []byte(`return {
    var = { ip = "10.0.0.7" },
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Core.IP != "10.0.0.7" {
		t.Fatalf("legacy section not read: %+v", cfg.Core)
	}

	raw, err := os.ReadFile(Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "core = {") || strings.Contains(string(raw), "var = {") {
		t.Fatalf("main.lua was not migrated:\n%s", raw)
	}
}
