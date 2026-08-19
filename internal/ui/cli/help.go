package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"ps4rpc/internal/app/autostart"
	"ps4rpc/internal/app/config"
	"ps4rpc/internal/service/ipc"
)

var commands = [][2]string{
	{"start", "Start the services enabled in the config, then exit"},
	{"stop [ROLE...]", "Stop the background services (default: all)"},
	{"status", "Print the state of the background services"},
	{"presence", "Run the presence service in the foreground"},
	{"bot", "Run the Discord bot in the foreground"},
}

var options = [][2]string{
	{"-h, --help", "Show help"},
	{"-v, --version", "Show version"},
	{"-c, --config", "Open the configuration directory"},
	{"    --headless", "Alias for the presence command"},
}

const autostartNote = `Autostart is on by default: the program registers "ps4rpc start" with the login
autostart of the system, which launches only the enabled services and never the
interface. Switch it off with the Autostart toggle on the Settings tab or with
core.autostart in main.lua.`

func PrintHelp(w io.Writer, version string) {
	var b strings.Builder

	b.WriteString(styleHead.Render("ps4rpc") + " " + styleDim.Render(version) + " " +
		styleDim.Render("- Discord Rich Presence (RPC) for PS4 with GoldHEN") + "\n\n")
	b.WriteString(styleKey.Render("Usage:") + " " + styleValue.Render("ps4rpc [command] [option]") + "\n\n")

	b.WriteString(styleHead.Render("Commands") + "\n")
	writeRows(&b, commands, styleRole)
	b.WriteString("\n" + styleHead.Render("Options") + "\n")
	writeRows(&b, options, styleOption)

	for _, line := range strings.Split(autostartNote, "\n") {
		b.WriteString("\n" + styleDim.Render(line))
	}
	b.WriteString("\n\n")
	for _, row := range [][2]string{
		{"Config directory", config.DefaultDir()},
		{"Runtime directory", ipc.RuntimeDir()},
		{"Cache directory", config.CacheDir()},
		{"Autostart entry", autostart.Location()},
	} {
		fmt.Fprintf(&b, "%s  %s\n", styleKey.Render(pad(row[0]+":", 18)), styleValue.Render(row[1]))
	}

	io.WriteString(w, b.String())
}

func writeRows(b *strings.Builder, rows [][2]string, name lipgloss.Style) {
	width := 0
	for _, row := range rows {
		if n := len([]rune(row[0])); n > width {
			width = n
		}
	}
	for _, row := range rows {
		fmt.Fprintf(b, "  %s  %s\n", name.Render(pad(row[0], width)), styleDim.Render(row[1]))
	}
}
