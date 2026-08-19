package cli

import (
	"fmt"
	"io"

	"ps4rpc/internal/config"
)

const helpTemplate = `ps4rpc %s - Discord Rich Presence (RPC) for PS4 with GoldHEN

Usage: ps4rpc [option]

Run without options to open the interactive interface (TUI).

Options:
  -h, --help       show help
  -v, --version    show version
  -c, --config     open the configuration directory
      --headless   run the presence loop without the TUI

The TUI is skipped automatically when stdout is not a terminal.
The Discord bot has its own binary: ps4bot [-ip ADDRESS] [-version]

Config directory: %s
`

func PrintHelp(w io.Writer, version string) {
	fmt.Fprintf(w, helpTemplate, version, config.DefaultDir())
}
