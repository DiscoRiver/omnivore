// Omnivore runs a command on many hosts over SSH and groups hosts with identical output.
package main

import (
	"os"

	"github.com/discoriver/massh/v2"
	"github.com/discoriver/omnivore/internal/cli"
)

func main() {
	// When ssh runs omnivore to ask for a password, answer and exit.
	massh.HandleAskpass()
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
