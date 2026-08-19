package tui

import (
	"log"
	"os"

	"ps4rpc/internal/app/config"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/term"
)

func IsTTY() bool {
	return term.IsTerminal(os.Stdout.Fd())
}

func Run(cfg *config.Config, version string, startTab int) error {
	m := New(cfg, version, startTab)
	defer func() {
		m.Shutdown()
		log.SetOutput(os.Stderr)
	}()

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
