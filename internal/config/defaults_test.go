package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultsParse(t *testing.T) {
	cfg, err := loadDefaults()
	if err != nil {
		t.Fatalf("embedded defaults are broken: %v", err)
	}
	if cfg.Core.ClientID == 0 || cfg.Core.WaitTime <= 0 {
		t.Fatalf("defaults/rpc.lua did not reach the config: %+v", cfg.Core)
	}
	if !cfg.Core.Enabled || !cfg.Core.Autostart {
		t.Fatalf("presence and autostart should ship enabled: %+v", cfg.Core)
	}
	if cfg.Bot.Enabled || cfg.Bot.Token != "" {
		t.Fatalf("the bot should ship off: %+v", cfg.Bot)
	}
	if len(cfg.Devapps) != 1 || cfg.Devapps[0] != (DevApp{}) {
		t.Fatalf("defaults/dev.lua should carry one empty row: %+v", cfg.Devapps)
	}
	if cfg.Mapped == nil || len(cfg.Mapped) != 0 {
		t.Fatalf("defaults/mapped.lua should be empty, got %+v", cfg.Mapped)
	}
}

func TestDefaultsMatchWhatTheProgramRenders(t *testing.T) {
	cfg := Default()
	rendered := map[string]string{
		"main.lua":   cfg.renderMain(),
		"rpc.lua":    cfg.renderRpc(),
		"bot.lua":    cfg.renderBot(),
		"dev.lua":    cfg.renderDev(),
		"mapped.lua": cfg.renderMapped(),
	}
	for name, want := range rendered {
		got, err := defaults.ReadFile("defaults/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Errorf("defaults/%s drifted from the renderer:\n--- shipped ---\n%s\n--- rendered ---\n%s",
				name, got, want)
		}
	}
}

func TestFirstRunWritesTheDefaults(t *testing.T) {
	SetDir(t.TempDir())

	cfg, existed, err := Load()
	if err != nil || existed {
		t.Fatalf("load on an empty dir: existed=%v err=%v", existed, err)
	}
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"main.lua", "rpc.lua", "bot.lua", "dev.lua", "mapped.lua"} {
		if _, err := os.Stat(filepath.Join(Dir, name)); err != nil {
			t.Errorf("%s not written: %v", name, err)
		}
	}

	back, existed, err := Load()
	if err != nil || !existed {
		t.Fatalf("reload: existed=%v err=%v", existed, err)
	}
	if !back.Equal(cfg) {
		t.Fatalf("round trip changed the config:\n%+v\n%+v", back, cfg)
	}
}
