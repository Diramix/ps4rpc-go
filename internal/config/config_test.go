package config

import (
	"os"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	oldWD, _ := os.Getwd()
	defer os.Chdir(oldWD)
	os.Chdir(dir)

	if err := os.MkdirAll(Dir, 0o755); err != nil {
		t.Fatal(err)
	}

	src := `# PS4RPC configuration

var {
    ip = 192.168.31.114
    client_id = 858345055966461973
    wait_time = 15
    retro_covers = true
    hibernate = false
    hibernate_time = 600
    use_devapps = false
    show_timer = false
    use_appname = false
}
`
	if err := os.WriteFile(Path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	mappedSrc := `[{"titleid":"CUSA10249","name":"The Last of Us, Part II = Remastered","image":"http://example/icon0.png"}]`
	if err := os.WriteFile(MappedPath, []byte(mappedSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, existed, err := Load()
	if err != nil || !existed {
		t.Fatalf("load: existed=%v err=%v", existed, err)
	}
	if cfg.Var.IP != "192.168.31.114" || cfg.Var.WaitTime != 15 || !cfg.Var.RetroCovers {
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
