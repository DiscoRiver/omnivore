package omnivore

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
