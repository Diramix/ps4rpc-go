package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colAccent = lipgloss.AdaptiveColor{Light: "#5b21b6", Dark: "#a78bfa"}
	colSubtle = lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#8b8b9a"}
	colText   = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#e6e6ef"}
	colOn     = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#4ade80"}
	colOff    = lipgloss.AdaptiveColor{Light: "#b91c1c", Dark: "#f87171"}
)

var (
	styleRole   = lipgloss.NewStyle().Bold(true).Foreground(colText)
	styleOn     = lipgloss.NewStyle().Bold(true).Foreground(colOn)
	styleOff    = lipgloss.NewStyle().Foreground(colSubtle)
	styleErr    = lipgloss.NewStyle().Bold(true).Foreground(colOff)
	styleKey    = lipgloss.NewStyle().Foreground(colSubtle)
	styleValue  = lipgloss.NewStyle().Foreground(colText)
	styleHead   = lipgloss.NewStyle().Bold(true).Foreground(colAccent)
	styleDim    = lipgloss.NewStyle().Foreground(colSubtle)
	styleOption = lipgloss.NewStyle().Foreground(colAccent)
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
