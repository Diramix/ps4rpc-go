package autostart

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const label = "com.ps4rpc.start"

func Location() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return label + ".plist"
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist")
}

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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("autostart: %w", err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + label + `</string>
	<key>ProgramArguments</key>
	<array>
		<string>` + plistEscape(exe) + `</string>
		<string>start</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`
	if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
		return fmt.Errorf("autostart: %w", err)
	}
	return nil
}

func registered() (string, error) {
	raw, err := os.ReadFile(Location())
	if err != nil {
		return "", err
	}
	body := string(raw)
	start := strings.Index(body, "<array>")
	if start < 0 {
		return "", fmt.Errorf("autostart: no ProgramArguments in %s", Location())
	}
	rest := body[start:]
	open := strings.Index(rest, "<string>")
	if open < 0 {
		return "", fmt.Errorf("autostart: no ProgramArguments in %s", Location())
	}
	rest = rest[open+len("<string>"):]
	end := strings.Index(rest, "</string>")
	if end < 0 {
		return "", fmt.Errorf("autostart: no ProgramArguments in %s", Location())
	}
	return plistUnescape(rest[:end]), nil
}

var plistEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func plistEscape(s string) string { return plistEscaper.Replace(s) }

var plistUnescaper = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&")

func plistUnescape(s string) string { return plistUnescaper.Replace(s) }
