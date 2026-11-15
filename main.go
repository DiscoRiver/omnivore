// Omnivore runs a command on many hosts over SSH and groups hosts with identical output.
package main

import (
	"os"

	"github.com/discoriver/omnivore/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
