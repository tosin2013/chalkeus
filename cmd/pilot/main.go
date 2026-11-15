// Command pilot is the specfoundry pilot (specs/016-go-pilot).
package main

import "github.com/tosin2013/specfoundry/internal/cli"

// version is stamped at build time: -ldflags "-X main.version=<git describe>".
var version = "dev"

func main() { cli.Main(version) }
