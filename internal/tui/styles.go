package tui

import "github.com/charmbracelet/lipgloss"

var (
	colAccent = lipgloss.AdaptiveColor{Light: "#5b21b6", Dark: "#a78bfa"}
	colSubtle = lipgloss.AdaptiveColor{Light: "#6b7280", Dark: "#8b8b9a"}
	colText   = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#e6e6ef"}
	colOn     = lipgloss.AdaptiveColor{Light: "#047857", Dark: "#4ade80"}
	colOff    = lipgloss.AdaptiveColor{Light: "#b91c1c", Dark: "#f87171"}
	colWarn   = lipgloss.AdaptiveColor{Light: "#b45309", Dark: "#fbbf24"}
	colBorder = lipgloss.AdaptiveColor{Light: "#c7c7d1", Dark: "#3b3b4d"}

	colBot     = lipgloss.AdaptiveColor{Light: "#1d4ed8", Dark: "#60a5fa"}
	colDiscord = lipgloss.AdaptiveColor{Light: "#4338ca", Dark: "#818cf8"}
	colPS4     = lipgloss.AdaptiveColor{Light: "#0369a1", Dark: "#38bdf8"}
	colValue   = lipgloss.AdaptiveColor{Light: "#0f766e", Dark: "#5eead4"}
)

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffffff")).
			Background(colAccent).
			Padding(0, 1)

	styleVersion = lipgloss.NewStyle().Foreground(colSubtle).Padding(0, 1)

	styleTab = lipgloss.NewStyle().
			Foreground(colSubtle).
			Padding(0, 2)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(colAccent).
			Padding(0, 2).
			Underline(true)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colBorder).
			Padding(0, 1)

	stylePanelTitle = lipgloss.NewStyle().Bold(true).Foreground(colAccent)

	styleLabel = lipgloss.NewStyle().Foreground(colSubtle)
	styleValue = lipgloss.NewStyle().Foreground(colText)

	styleRowActive = lipgloss.NewStyle().Bold(true).Foreground(colAccent)

	styleOn   = lipgloss.NewStyle().Bold(true).Foreground(colOn)
	styleOff  = lipgloss.NewStyle().Bold(true).Foreground(colOff)
	styleWarn = lipgloss.NewStyle().Foreground(colWarn)

	styleLogTime   = lipgloss.NewStyle().Foreground(colBorder)
	styleLogSource = lipgloss.NewStyle().Bold(true)
	styleLogValue  = lipgloss.NewStyle().Foreground(colValue)

	styleHelp   = lipgloss.NewStyle().Foreground(colSubtle)
	styleStatus = lipgloss.NewStyle().Foreground(colSubtle).Padding(0, 1)
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
