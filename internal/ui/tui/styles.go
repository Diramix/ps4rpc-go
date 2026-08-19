package tui

import (
	"github.com/charmbracelet/lipgloss"

	"ps4rpc/internal/ui/theme"
)

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(theme.Accent).
			Padding(0, 1)

	styleVersion = lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1)

	styleTab = lipgloss.NewStyle().
			Foreground(theme.Subtle).
			Padding(0, 2)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Accent).
			Padding(0, 2).
			Underline(true)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Border).
			Padding(0, 1)

	stylePanelTitle = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)

	styleLabel = lipgloss.NewStyle().Foreground(theme.Subtle)
	styleValue = lipgloss.NewStyle().Foreground(theme.Text)

	styleRowActive = lipgloss.NewStyle().Bold(true).Foreground(theme.Accent)

	styleOn   = lipgloss.NewStyle().Bold(true).Foreground(theme.On)
	styleOff  = lipgloss.NewStyle().Bold(true).Foreground(theme.Off)
	styleWarn = lipgloss.NewStyle().Foreground(theme.Warn)

	styleLogTime   = lipgloss.NewStyle().Foreground(theme.Border)
	styleLogSource = lipgloss.NewStyle().Bold(true)
	styleLogValue  = lipgloss.NewStyle().Foreground(theme.Value)

	styleHelp   = lipgloss.NewStyle().Foreground(theme.Subtle)
	styleStatus = lipgloss.NewStyle().Foreground(theme.Subtle).Padding(0, 1)
)

func onOff(v bool) string {
	return upDown(v, "on", "off")
}

func upDown(v bool, up, down string) string {
	if v {
		return styleOn.Render(up)
	}
	return styleOff.Render(down)
}
