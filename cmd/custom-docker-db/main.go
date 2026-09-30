package main

import (
	"os"

	"github.com/pvfm/custom-docker-db/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
