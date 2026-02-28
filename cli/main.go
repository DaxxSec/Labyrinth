package main

import (
	"os"

	"github.com/DaxxSec/labyrinth/cli/cmd"
)

func main() {
	defer cmd.CleanupEnvFiles()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
