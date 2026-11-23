// Command chalkeus is Chalkeus (specs/027-chalkeus-rename); its pilot commands are under `forge` (specs/016-go-pilot).
package main

import "github.com/tosin2013/chalkeus/internal/cli"

// version is stamped at build time: -ldflags "-X main.version=<git describe>".
var version = "dev"

func main() { cli.Main(version) }
