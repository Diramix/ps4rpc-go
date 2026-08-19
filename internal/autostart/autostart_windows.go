package autostart

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func Location() string { return `HKCU\` + runKey + `\` + appName }

func openRun(access uint32) (registry.Key, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, access)
	if err != nil {
		return 0, fmt.Errorf("autostart: %w", err)
	}
	return k, nil
}

func Enabled() (bool, error) {
	k, err := openRun(registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()

	_, _, err = k.GetStringValue(appName)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("autostart: %w", err)
}

func Set(on bool) error {
	k, err := openRun(registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !on {
		if err := k.DeleteValue(appName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("autostart: %w", err)
		}
		return nil
	}

	exe := exePath()
	if exe == "" {
		return fmt.Errorf("autostart: cannot locate the executable")
	}
	if err := k.SetStringValue(appName, fmt.Sprintf("%q start", exe)); err != nil {
		return fmt.Errorf("autostart: %w", err)
	}
	return nil
}

func registered() (string, error) {
	k, err := openRun(registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	cmd, _, err := k.GetStringValue(appName)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(cmd, `"`) {
		if end := strings.Index(cmd[1:], `"`); end >= 0 {
			return cmd[1 : end+1], nil
		}
	}
	if fields := strings.Fields(cmd); len(fields) > 0 {
		return fields[0], nil
	}
	return "", fmt.Errorf("autostart: %s is empty", Location())
}
