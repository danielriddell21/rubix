// Command rubix is a Rubik's cube solver: a headless core that models the cube as
// data, solves it with the methods from two "Coding Adventure" videos, and can drive
// an optional Ebiten visualizer or a LEGO EV3 robot. See the README for details.
package main

import (
	"os"

	"github.com/danielriddell21/rubix/internal/cli"
)

// version is the build version, overridden at release time via
// -ldflags "-X main.version=...". It defaults to "dev" for local builds.
var version = "dev"

func main() {
	os.Exit(cli.Run(version, os.Args[1:]))
}
