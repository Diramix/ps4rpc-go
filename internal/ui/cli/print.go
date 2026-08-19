package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"ps4rpc/internal/ui/theme"
)

var (
	styleRole   = lipgloss.NewStyle().Bold(true).Foreground(theme.Text)
	styleOn     = lipgloss.NewStyle().Bold(true).Foreground(theme.On)
	styleOff    = lipgloss.NewStyle().Foreground(theme.Subtle)
	styleErr    = lipgloss.NewStyle().Bold(true).Foreground(theme.Off)
	styleKey    = lipgloss.NewStyle().Foreground(theme.Subtle)
	styleValue  = lipgloss.NewStyle().Foreground(theme.Text)
	styleHead   = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)
	styleDim    = lipgloss.NewStyle().Foreground(theme.Subtle)
	styleOption = lipgloss.NewStyle().Foreground(theme.Accent)
)

const roleWidth = 9

func StatusLine(role string, up bool, state string, details ...string) string {
	styled := styleOff.Render(state)
	if up {
		styled = styleOn.Render(state)
	}
	line := fmt.Sprintf("%s %s", styleRole.Render(pad(role, roleWidth)), styled)
	if len(details) > 0 {
		line += "  " + styleDim.Render("· ") + strings.Join(details, styleDim.Render(" · "))
	}
	return line
}

func DoneLine(role, state string) string {
	return fmt.Sprintf("%s %s", styleRole.Render(pad(role, roleWidth)), styleDim.Render(state))
}

func KV(key, value string) string {
	return styleKey.Render(key+" ") + styleValue.Render(value)
}

func Flag(key string, on bool) string {
	mark := styleOff.Render("no")
	if on {
		mark = styleOn.Render("yes")
	}
	return styleKey.Render(key+" ") + mark
}

func Fail(w io.Writer, err error) {
	fmt.Fprintln(w, styleErr.Render("✗ ")+styleErr.Render(err.Error()))
}

func pad(s string, w int) string {
	if n := len([]rune(s)); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}
