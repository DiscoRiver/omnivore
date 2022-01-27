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
}

func Run(cmd *OmniCommandFlags) {
	ui.MakeDP()

	go OmniRun(cmd)
	ui.DP.StartUI()
}
