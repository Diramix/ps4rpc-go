package ps4

import (
	"fmt"
	"regexp"
	"strings"
)

const sandboxDir = "/mnt/sandbox"

var titleIDRe = regexp.MustCompile(`[a-zA-Z0-9]{4}[0-9]{5}`)

var ps4Prefixes = map[string]bool{"CUSA": true}

var classicPrefixes = map[string]bool{
	"SLES": true, "SCES": true, "SCED": true, "SLUS": true, "SCUS": true,
	"SLPS": true, "SCAJ": true, "SLKA": true, "SLPM": true, "SCPS": true,
	"CF00": true, "SCKA": true, "ALCH": true, "CPCS": true, "SLAJ": true,
	"KOEI": true, "ARZE": true, "TCPS": true, "SCCS": true, "PAPX": true,
	"SRPM": true, "GUST": true, "WLFD": true, "ULKS": true, "VUGJ": true,
	"HAKU": true, "ROSE": true, "CZP2": true, "ARP2": true, "PKP2": true,
	"SLPN": true, "NMP2": true, "MTP2": true, "SCPM": true, "PBPX": true,
}

func (c *Client) Check() error {
	entries, err := c.nameList(sandboxDir)
	if err != nil {
		return fmt.Errorf("no FTP server or sandbox on %s: %w", c.ip, err)
	}
	for _, e := range entries {
		if strings.HasPrefix(baseName(e), "NPXS20001_") {
			return nil
		}
	}
	return fmt.Errorf("shell UI sandbox (NPXS20001) not found on %s", c.ip)
}

func (c *Client) TitleID() (titleID, gameType string, ok bool) {
	entries, err := c.nameList(sandboxDir)
	if err != nil {
		return "", "", false
	}

	for _, e := range entries {
		base := baseName(e)
		if strings.HasPrefix(base, "NPXS") {
			continue
		}
		if m := titleIDRe.FindString(base); m != "" {
			titleID = m
		}
	}

	if titleID == "" {
		return "main_menu", "", true
	}
	return titleID, classify(titleID), true
}

func classify(titleID string) string {
	if titleID == "" {
		return ""
	}
	switch prefix := titleID[:4]; {
	case ps4Prefixes[prefix]:
		return "PS4"
	case classicPrefixes[prefix]:
		return "PS1/2"
	default:
		return "Homebrew"
	}
}

func baseName(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}
