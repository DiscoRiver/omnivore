package omnivore

import (
	"fmt"
	"github.com/discoriver/omnivore/pkg/group"
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
	grp := group.NewValueGrouping()

	go func(){
		for {
			select {
			case <-grp.Update:
				for k, i := range grp.EncodedValueGroup {
					for h := range i {
						fmt.Println(i[h], string(grp.EncodedValueToOriginal[k]))
					}
				}
			}
		}
	}()

	OmniRun(cmd, grp)
}
