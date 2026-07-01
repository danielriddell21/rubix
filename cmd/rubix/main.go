package main

import (
	"os"

	"github.com/danielriddell21/rubix/internal/cli"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(version, os.Args[1:]))
}
