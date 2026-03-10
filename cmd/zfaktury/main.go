package main

import (
	"os"

	"github.com/zajca/zfaktury/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
