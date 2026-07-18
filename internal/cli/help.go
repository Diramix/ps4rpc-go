package cli

import (
	"fmt"
	"io"

	"ps4rpc/internal/config"
)

const helpTemplate = `ps4rpc %s - Discord Rich Presence (RPC) for PS4 with GoldHEN

Usage: ps4rpc [option]

Options:
  -h, --help       show help
  -v, --version    show version
  -c, --config     open the configuration directory

Config directory: %s
`

func PrintHelp(w io.Writer, version string) {
	fmt.Fprintf(w, helpTemplate, version, config.DefaultDir())
}
