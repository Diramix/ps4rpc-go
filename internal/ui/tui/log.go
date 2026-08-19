package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"ps4rpc/internal/ui/theme"
)

var logSources = map[string]lipgloss.AdaptiveColor{
	"presence": theme.Accent,
	"rpc":      theme.Accent,
	"bot":      theme.Bot,
	"discord":  theme.Discord,
	"ps4":      theme.PS4,
	"config":   theme.Subtle,
	"daemon":   theme.Warn,
	"tmdb":     theme.PS4,
}

var (
	logGood = map[string]bool{
		"online": true, "started": true, "running": true, "connected": true,
		"ready": true, "ok": true, "enabled": true, "up": true, "registered": true,
	}
	logBad = map[string]bool{
		"offline": true, "unreachable": true, "stopped": true, "failed": true,
		"error": true, "disabled": true, "down": true, "refused": true, "timeout": true,
	}
	logQuiet = map[string]bool{
		"sleeping": true, "idle": true, "waiting": true, "reusing": true, "skipping": true,
	}
)

func formatLog(line string) string {
	stamp := styleLogTime.Render(time.Now().Format("15:04:05"))

	source, message, ok := strings.Cut(line, ": ")
	if !ok || !isSource(source) {
		return stamp + " " + highlight(line)
	}
	style := styleLogSource.Foreground(theme.Subtle)
	if col, known := logSources[source]; known {
		style = styleLogSource.Foreground(col)
	}
	return stamp + " " + style.Render(source) + styleLogTime.Render(" · ") + highlight(message)
}

func isSource(s string) bool {
	if s == "" || len(s) > 12 || strings.ContainsAny(s, " \t") {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func highlight(message string) string {
	fields := strings.Split(message, " ")
	for i, word := range fields {
		key := strings.ToLower(strings.Trim(word, ".,;:!?()[]\"'"))
		switch {
		case logGood[key]:
			fields[i] = styleOn.Render(word)
		case logBad[key]:
			fields[i] = styleOff.Render(word)
		case logQuiet[key]:
			fields[i] = styleHelp.Render(word)
		case looksLikeValue(key):
			fields[i] = styleLogValue.Render(word)
		default:
			fields[i] = styleValue.Render(word)
		}
	}
	return strings.Join(fields, " ")
}

func looksLikeValue(s string) bool {
	if s == "" {
		return false
	}
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits > 0 && digits*2 >= len(s)
}
