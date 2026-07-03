package main

import (
	"os"

	"github.com/areyesgfx/swale/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
