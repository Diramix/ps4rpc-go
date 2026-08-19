package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var layoutKeys = map[rune]rune{
	'й': 'q', 'ц': 'w', 'у': 'e', 'к': 'r', 'е': 't', 'н': 'y', 'г': 'u',
	'ш': 'i', 'щ': 'o', 'з': 'p', 'х': '[', 'ъ': ']',
	'ф': 'a', 'ы': 's', 'в': 'd', 'а': 'f', 'п': 'g', 'р': 'h', 'о': 'j',
	'л': 'k', 'д': 'l', 'ж': ';', 'э': '\'',
	'я': 'z', 'ч': 'x', 'с': 'c', 'м': 'v', 'и': 'b', 'т': 'n', 'ь': 'm',
	'б': ',', 'ю': '.',

	'ї': ']', 'є': '\'', 'і': 's', 'ґ': '\\',
}

func keyName(msg tea.KeyMsg) string {
	if msg.Type != tea.KeyRunes || msg.Alt || len(msg.Runes) != 1 {
		return msg.String()
	}
	in := msg.Runes[0]
	lower := []rune(strings.ToLower(string(in)))[0]
	r, ok := layoutKeys[lower]
	if !ok {
		return msg.String()
	}
	if in != lower {
		return strings.ToUpper(string(r))
	}
	return string(r)
}
