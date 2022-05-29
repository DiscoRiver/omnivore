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

	/*
		Used to avoid race condition in UI initialisation.

		Essentially, we need to populate the ossh StreamCycle before starting the UI due to pointer logic. Additionally,
		once the UI is started, it can be cumbersome to process single line errors when closing the app, so if anything
		goes wrong with the job initialisation, we can just print a failure without needing to handle any UI closures.

		safeToStartUI should indicate that StreamCycle was successfully populated.

		uiStarted should indicate when the UI is functional, so we can start refreshing it.
	*/
	safeToStartUI := make(chan struct{}, 1)
	uiStarted := make(chan struct{}, 1)

	go OmniRun(cmd, safeToStartUI, uiStarted)

	select {
	case <-safeToStartUI:
		ui.Collective.StartUI(uiStarted) // Is blocking, code below this line will not start.
	}
}
