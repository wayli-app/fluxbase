package main

import (
	"os"

	"github.com/fluxbase-eu/fluxbase/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
