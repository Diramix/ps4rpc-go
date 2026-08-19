package tui

import (
	"fmt"
	"strconv"

	"ps4rpc/internal/config"
)

type fieldKind int

const (
	fieldToggle fieldKind = iota
	fieldText
	fieldSecret
	fieldSection
)

type field struct {
	kind    fieldKind
	label   string
	help    string
	get     func(*config.Config) string
	set     func(*config.Config, string) error
	getBool func(*config.Config) bool
	toggle  func(*config.Config)
}

func (f field) selectable() bool { return f.kind != fieldSection }

func settingsFields() []field {
	return []field{
		{kind: fieldSection, label: "Connection"},
		{
			kind:  fieldText,
			label: "PS4 IP",
			help:  "address of the console running GoldHEN",
			get:   func(c *config.Config) string { return c.Core.IP },
			set: func(c *config.Config, v string) error {
				if v == "" {
					return fmt.Errorf("IP must not be empty")
				}
				c.Core.IP = v
				return nil
			},
		},

		{kind: fieldSection, label: "Startup"},
		{
			kind:    fieldToggle,
			label:   "Autostart",
			help:    "Start the enabled services at login",
			getBool: func(c *config.Config) bool { return c.Core.Autostart },
			toggle:  func(c *config.Config) { c.Core.Autostart = !c.Core.Autostart },
		},

		{kind: fieldSection, label: "Rich Presence"},
		{
			kind:    fieldToggle,
			label:   "RPC enabled",
			help:    "start the presence loop automatically",
			getBool: func(c *config.Config) bool { return c.Core.Enabled },
			toggle:  func(c *config.Config) { c.Core.Enabled = !c.Core.Enabled },
		},
		{
			kind:  fieldText,
			label: "Client ID",
			help:  "default Discord application ID",
			get:   func(c *config.Config) string { return strconv.FormatInt(c.Core.ClientID, 10) },
			set: func(c *config.Config, v string) error {
				n, err := strconv.ParseInt(v, 10, 64)
				if err != nil {
					return fmt.Errorf("client_id must be a number")
				}
				c.Core.ClientID = n
				return nil
			},
		},
		{
			kind:  fieldText,
			label: "Poll interval",
			help:  "seconds between PS4 requests",
			get:   func(c *config.Config) string { return strconv.Itoa(c.Core.WaitTime) },
			set: func(c *config.Config, v string) error {
				n, err := strconv.Atoi(v)
				if err != nil || n <= 0 {
					return fmt.Errorf("interval must be a number > 0")
				}
				c.Core.WaitTime = n
				return nil
			},
		},

		{kind: fieldSection, label: "Discord bot"},
		{
			kind:    fieldToggle,
			label:   "Bot enabled",
			help:    "start the bot automatically",
			getBool: func(c *config.Config) bool { return c.Bot.Enabled },
			toggle:  func(c *config.Config) { c.Bot.Enabled = !c.Bot.Enabled },
		},
		{
			kind:  fieldSecret,
			label: "Token",
			help:  "bot token, v to show or hide",
			get:   func(c *config.Config) string { return c.Bot.Token },
			set:   func(c *config.Config, v string) error { c.Bot.Token = v; return nil },
		},
		{
			kind:  fieldText,
			label: "Owner ID",
			get:   func(c *config.Config) string { return c.Bot.OwnerID },
			set:   func(c *config.Config, v string) error { c.Bot.OwnerID = v; return nil },
		},
		{
			kind:  fieldText,
			label: "Guild ID",
			get:   func(c *config.Config) string { return c.Bot.GuildID },
			set:   func(c *config.Config, v string) error { c.Bot.GuildID = v; return nil },
		},
		{
			kind:  fieldText,
			label: "Account ID",
			get:   func(c *config.Config) string { return c.Bot.AccountID },
			set:   func(c *config.Config, v string) error { c.Bot.AccountID = v; return nil },
		},
	}
}

func mask(s string) string {
	if s == "" {
		return ""
	}
	out := make([]rune, len([]rune(s)))
	for i := range out {
		out[i] = '•'
	}
	return string(out)
}
