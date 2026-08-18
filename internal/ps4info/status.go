package ps4info

import (
	"regexp"
	"strings"
	"time"
)

var titleIDRe = regexp.MustCompile(`[a-zA-Z0-9]{4}[0-9]{5}`)

type State string

const (
	StateOffline State = "offline"
	StateMenu    State = "menu"
	StateGame    State = "game"
)

type Status struct {
	Online       bool
	State        State
	TitleID      string
	SessionStart time.Time
}

func (c *Client) IsOnline() bool {
	conn, err := c.dial()
	if err != nil {
		return false
	}
	conn.Quit()
	return true
}

func (c *Client) Status() (Status, error) {
	entries, err := c.list("/mnt/sandbox")
	if err != nil {
		return Status{Online: false, State: StateOffline}, nil
	}

	s := Status{Online: true, State: StateMenu}
	for _, e := range entries {
		base := e.Name[strings.LastIndex(e.Name, "/")+1:]
		if strings.HasPrefix(base, "NPXS") || base == "." || base == ".." {
			continue
		}
		if m := titleIDRe.FindString(base); m != "" {
			s.State = StateGame
			s.TitleID = m
			s.SessionStart = e.Time
		}
	}
	return s, nil
}
