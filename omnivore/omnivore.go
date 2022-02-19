package omnivore

import (
	"github.com/discoriver/omnivore/internal/ui"
)

type OmniCommandFlags struct {
	Hosts              []string
	BastionHost        string
	Username           string
	Password           string
	PrivateKeyLocation string
	PrivateKeyPassword string
	Command            string

	// Timeout for the SSH connection
	SSHTimeout int
	// Timeout for the command execution
	CommandTimeout int

	// Insecure mode
	Insecure bool
}

func Run(cmd *OmniCommandFlags) {
	ui.MakeDP()

	// Used to avoid race condition in UI initialisation.
	safeToStartUI := make(chan struct{}, 1)
	uiStarted := make(chan struct{}, 1)

	go OmniRun(cmd, safeToStartUI, uiStarted)

	select {
	case <-safeToStartUI:
		ui.DP.StartUI(uiStarted) // Is blocking, code below this line will not start.
	}
}
