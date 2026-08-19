package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"ps4rpc/internal/ui/theme"
)

func (m *Model) layout() {
	w := m.contentWidth() - 4
	if w < 20 {
		w = 20
	}
	m.vp.Width = w
	m.setLogContent()
	m.vp.GotoBottom()
}

func lineCount(s string) int { return strings.Count(s, "\n") + 1 }

func (m *Model) contentWidth() int {
	w := m.width - 4
	if w < 30 {
		w = 30
	}
	return w
}

func (m *Model) View() string {
	if !m.ready {
		return "loading…"
	}
	header, footer := m.header(), m.footer()
	m.body = m.height - lineCount(header) - lineCount(footer)
	if m.body < 3 {
		m.body = 3
	}

	var body string
	switch m.tabIdx {
	case tabDashboard:
		body = m.viewDashboard()
	case tabSettings:
		body = m.viewSettings()
	case tabMappings:
		body = m.viewMappings()
	}

	if pad := m.body - lineCount(body); pad > 0 {
		body += strings.Repeat("\n", pad)
	}
	return header + "\n" + body + "\n" + footer
}

func (m *Model) header() string {
	title := styleTitle.Render("PS4RPC")
	ver := styleVersion.Render(m.version)

	tabs := make([]string, 0, len(tabNames))
	for i, name := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if tab(i) == m.tabIdx {
			tabs = append(tabs, styleTabActive.Render(label))
		} else {
			tabs = append(tabs, styleTab.Render(label))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Bottom, title, ver)
	line := lipgloss.NewStyle().Foreground(theme.Border).
		Render(strings.Repeat("─", max(1, m.contentWidth())))
	return lipgloss.JoinVertical(lipgloss.Left, bar, strings.Join(tabs, ""), line)
}

func (m *Model) footer() string {
	var parts []string
	switch m.tabIdx {
	case tabDashboard:
		parts = append(parts, "s rpc", "c clear log", "↑↓ scroll")
	case tabSettings:
		if m.editing {
			parts = append(parts, "enter apply", "esc cancel")
		} else {
			parts = append(parts, "↑↓ select", "enter/space change", "v reveal token")
		}
	case tabMappings:
		if m.editing {
			parts = append(parts, "enter apply", "esc cancel")
		} else {
			parts = append(parts, "↑↓ row", "h l column", "a add", "d delete", "enter edit")
		}
	}
	parts = append(parts, "tab switch", "q quit")
	help := styleHelp.Width(m.contentWidth()).Render(strings.Join(parts, " · "))

	var state []string
	if m.err != "" {
		state = append(state, styleOff.Render("✗ "+m.err))
	} else if m.status != "" {
		state = append(state, styleStatus.Render(m.status))
	}

	line := lipgloss.NewStyle().Foreground(theme.Border).
		Render(strings.Repeat("─", max(1, m.contentWidth())))
	out := line + "\n" + help
	if len(state) > 0 {
		out += "\n" + strings.Join(state, "  ")
	}
	return out
}

func (m *Model) viewDashboard() string {
	st := m.rpcStatus

	ip := m.cfg.Core.IP
	if ip == "" {
		ip = "not set"
	}
	cardW := (m.contentWidth() - 6) / 3
	if cardW < 18 {
		cardW = 18
	}
	lastW := m.contentWidth() - 6 - 2*cardW
	if lastW < cardW {
		lastW = cardW
	}

	gameName := st.GameName
	if gameName == "" {
		gameName = "-"
	}

	ps4Lines := []string{
		styleLabel.Render("address ") + styleValue.Render(ip),
		styleLabel.Render("status  ") + upDown(m.ps4Online, "online", "not found"),
	}
	rpcLines := []string{
		styleLabel.Render("service ") + upDown(st.Running, "running", "stopped"),
		styleLabel.Render("game    ") + styleValue.Render(trunc(gameName, cardW-12)),
	}
	botLines := []string{
		styleLabel.Render("service ") + upDown(m.botStatus.Running, "running", "stopped"),
		styleLabel.Render("token   ") + onOff(m.cfg.Bot.Token != ""),
	}

	cardH := max(len(ps4Lines), max(len(rpcLines), len(botLines)))
	ps4Card := m.card(cardW, cardH, "PlayStation 4", ps4Lines)
	rpcCard := m.card(cardW, cardH, "Rich Presence", rpcLines)
	botCard := m.card(lastW, cardH, "Discord bot", botLines)

	cards := lipgloss.JoinHorizontal(lipgloss.Top, ps4Card, rpcCard, botCard)

	vpHeight := m.body - lineCount(cards) - 3
	if vpHeight < 3 {
		vpHeight = 3
	}
	m.vp.Height = vpHeight

	logs := stylePanel.Width(m.contentWidth() - 2).Render(
		stylePanelTitle.Render("Log") + "\n" + m.vp.View())

	return cards + "\n" + logs
}

func (m *Model) card(w, h int, title string, lines []string) string {
	for len(lines) < h {
		lines = append(lines, "")
	}
	body := stylePanelTitle.Render(title) + "\n" + strings.Join(lines, "\n")
	return stylePanel.Width(w).Render(body)
}

func (m *Model) bodyHeight() int {
	h := m.body - 2
	if h < 3 {
		h = 3
	}
	return h
}

func window(lines []string, cursor, h int) []string {
	if len(lines) <= h {
		return lines
	}
	start := cursor - h/2
	if start < 0 {
		start = 0
	}
	if start+h > len(lines) {
		start = len(lines) - h
	}
	return lines[start : start+h]
}

func (m *Model) viewSettings() string {
	labelW := 0
	for _, f := range m.fields {
		if n := len([]rune(f.label)); f.selectable() && n > labelW {
			labelW = n
		}
	}

	var lines []string
	cursorLine := 0
	for i, f := range m.fields {
		if f.kind == fieldSection {
			if i > 0 {
				lines = append(lines, "")
			}
			lines = append(lines, stylePanelTitle.Render("▍"+f.label))
			continue
		}

		selected := i == m.cursor
		cursor := "  "
		label := styleLabel.Render(pad(f.label, labelW))
		if selected {
			cursor = styleRowActive.Render("▸ ")
			label = styleRowActive.Render(pad(f.label, labelW))
		}

		var value string
		switch f.kind {
		case fieldToggle:
			value = onOff(f.getBool(m.cfg))
		case fieldSecret:
			raw := f.get(m.cfg)
			if m.secret {
				value = styleValue.Render(orDash(raw))
			} else {
				value = styleValue.Render(orDash(mask(raw)))
			}
		default:
			value = styleValue.Render(orDash(f.get(m.cfg)))
		}
		if selected && m.editing {
			value = m.input.View()
		}

		line := cursor + label + "  " + value
		if selected && !m.editing && f.help != "" {
			line += "  " + styleHelp.Render("- "+f.help)
		}
		if selected {
			cursorLine = len(lines)
		}
		lines = append(lines, line)
	}
	lines = window(lines, cursorLine, m.bodyHeight())
	return stylePanel.Width(m.contentWidth() - 2).Render(strings.Join(lines, "\n"))
}

func (m *Model) viewMappings() string {
	var rows []string
	if len(m.cfg.Mapped) == 0 {
		rows = append(rows, styleHelp.Render("empty - press a to add"))
	}
	for i, mp := range m.cfg.Mapped {
		cells := []string{trunc(orDash(mp.TitleID), 16), trunc(orDash(mp.Name), 28), trunc(orDash(mp.Image), 24)}
		rows = append(rows, m.renderRow(i, cells))
	}
	rows = window(rows, m.rowMapped, m.bodyHeight()-1)

	body := stylePanelTitle.Render("Mapped games") + "\n" + strings.Join(rows, "\n")
	return stylePanel.Width(m.contentWidth() - 2).Render(body)
}

func (m *Model) renderRow(idx int, cells []string) string {
	active := m.rowMapped == idx
	widths := []int{18, 30, 26}

	var parts []string
	for c, cell := range cells {
		text := cell
		if active && c == m.col {
			if m.editing {
				text = m.input.View()
			} else {
				text = styleRowActive.Render(stripPad(cell))
			}
		}
		parts = append(parts, padVisible(text, widths[c]))
	}
	prefix := "  "
	if active {
		prefix = styleRowActive.Render("▸ ")
	}
	return prefix + strings.Join(parts, " ")
}

func pad(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(r))
}

func padVisible(s string, w int) string {
	n := lipgloss.Width(s)
	if n >= w {
		return s
	}
	return s + strings.Repeat(" ", w-n)
}

func stripPad(s string) string { return strings.TrimRight(s, " ") }

func trunc(s string, w int) string {
	if w < 4 {
		w = 4
	}
	r := []rune(s)
	if len(r) <= w {
		return s
	}
	return string(r[:w-1]) + "…"
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
