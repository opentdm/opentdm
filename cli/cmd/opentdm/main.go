// Command opentdm is the read-only consumption CLI: log in with a service
// token, then pull or inject a project+environment's resolved variables.
package main

import (
	"os"

	"github.com/opentdm/opentdm/cli/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cli.Main(version, os.Args[1:]))
}
