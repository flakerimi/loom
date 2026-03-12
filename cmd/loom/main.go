package main

import (
	"os"

	"github.com/constructspace/loom/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd := cli.NewRootCmd()
	cmd.Version = version + " (" + commit + ")"
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
