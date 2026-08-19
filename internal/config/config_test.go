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
	if cfg.Var.IP != "192.168.31.114" || cfg.Var.WaitTime != 15 {
		t.Fatalf("var parsed wrong: %+v", cfg.Var)
	}
	if cfg.Var.ClientID != 858345055966461973 {
		t.Fatalf("client_id parsed wrong: %d", cfg.Var.ClientID)
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
	if cfg2.Var != cfg.Var || cfg2.Mapped[0] != cfg.Mapped[0] {
		t.Fatalf("round-trip mismatch:\n%+v\n%+v", cfg, cfg2)
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
	if cfg.Var.IP != "10.0.0.5" || cfg.Var.WaitTime != 42 {
		t.Fatalf("user values not preserved: %+v", cfg.Var)
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
