package main

import (
	"os"

	"github.com/newrelic/nrdot-host/nrdot-ctl/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}