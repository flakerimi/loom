package main

import (
	"os"

	"github.com/constructspace/loom/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
