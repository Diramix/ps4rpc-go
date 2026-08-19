//go:build !windows && !darwin

package autostart

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func dir() string {
	if base := os.Getenv("XDG_CONFIG_HOME"); base != "" {
		return filepath.Join(base, "autostart")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "autostart"
	}
	return filepath.Join(home, ".config", "autostart")
}

func Location() string { return filepath.Join(dir(), appName+".desktop") }

func Enabled() (bool, error) {
	_, err := os.Stat(Location())
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func Set(on bool) error {
	path := Location()
	if !on {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("autostart: %w", err)
		}
		return nil
	}

	exe := exePath()
	if exe == "" {
		return fmt.Errorf("autostart: cannot locate the executable")
	}
	if err := os.MkdirAll(dir(), 0o755); err != nil {
		return fmt.Errorf("autostart: %w", err)
	}
	entry := "[Desktop Entry]\n" +
		"Type=Application\n" +
		"Name=PS4RPC\n" +
		"Comment=Discord Rich Presence for PS4\n" +
		fmt.Sprintf("Exec=%q start\n", exe) +
		"Terminal=false\n" +
		"X-GNOME-Autostart-enabled=true\n"
	if err := os.WriteFile(path, []byte(entry), 0o644); err != nil {
		return fmt.Errorf("autostart: %w", err)
	}
	return nil
}

func registered() (string, error) {
	raw, err := os.ReadFile(Location())
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if !strings.HasPrefix(line, "Exec=") {
			continue
		}
		cmd := strings.TrimPrefix(line, "Exec=")
		if strings.HasPrefix(cmd, `"`) {
			if end := strings.Index(cmd[1:], `"`); end >= 0 {
				return cmd[1 : end+1], nil
			}
		}
		if fields := strings.Fields(cmd); len(fields) > 0 {
			return fields[0], nil
		}
		break
	}
	return "", fmt.Errorf("autostart: no Exec line in %s", Location())
}
