package bot

import (
	"fmt"
	"strings"
	"time"

	"ps4rpc/internal/source/ps4"
)

const (
	colorPS4     = 0x2E6BE6
	colorOffline = 0x555B66
	colorGold    = 0xF1C40F
	colorTrophy  = 0x4E9AF1
)

func humanBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func progressBar(pct int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct / 10
	return strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
}

func gradeEmoji(grade int) string {
	switch grade {
	case ps4.GradePlatinum:
		return "🏆"
	case ps4.GradeGold:
		return "🥇"
	case ps4.GradeSilver:
		return "🥈"
	case ps4.GradeBronze:
		return "🥉"
	}
	return "•"
}

func discordRel(t time.Time) string {
	if t.IsZero() || t.Year() <= 1 {
		return "-"
	}
	return fmt.Sprintf("<t:%d:R>", t.Unix())
}

func discordDate(t time.Time) string {
	if t.IsZero() || t.Year() <= 1 {
		return "-"
	}
	return fmt.Sprintf("<t:%d:d>", t.Unix())
}

func discordTime(t time.Time) string {
	if t.IsZero() || t.Year() <= 1 {
		return "-"
	}
	return fmt.Sprintf("<t:%d:t>", t.Unix())
}

func elapsed(t time.Time) string {
	if t.IsZero() || t.Year() <= 1 {
		return "-"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	return humanDuration(d)
}

func humanDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	m := (d % time.Hour) / time.Minute
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
