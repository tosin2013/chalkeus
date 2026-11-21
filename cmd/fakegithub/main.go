// Command fakegithub is the integration tests' fake GitHub (025): a REST subset, git over HTTP
// (/git/) and the owner API (/_test/), backed by real
// bare git repositories. It is a test tool, built and run only by test/integration, and never
// released (no release workflow builds it).
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tosin2013/specfoundry/internal/fakegh"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8090", "listen address (the integration tests use the kind network's gateway)")
	dir := flag.String("dir", "", "folder for the bare repositories (default: a new temporary folder)")
	flag.Parse()
	if *dir == "" {
		d, err := os.MkdirTemp("", "fakegithub-")
		if err != nil {
			log.Fatal(err)
		}
		*dir = d
	}
	s := fakegh.New(*dir)
	fmt.Printf("fakegithub: listening on %s, repositories in %s\n", *addr, *dir)
	log.Fatal(http.ListenAndServe(*addr, fakegh.NewMux(s)))
}
