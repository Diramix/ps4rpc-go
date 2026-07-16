package main

import (
	"flag"
	"fmt"
	"os"

	"ps4rpc/internal/config"
)

func main() {
	out := flag.String("o", config.Dir, "output config directory")
	flag.Parse()

	config.SetDir(*out)
	if err := config.Default().Save(); err != nil {
		fmt.Fprintf(os.Stderr, "genconfig: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("generated %s and %s\n", config.Path, config.MappedPath)
}
