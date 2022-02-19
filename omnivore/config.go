package omnivore

import (
	"github.com/discoriver/massh"
	"github.com/discoriver/omnivore/internal/config"
	"github.com/discoriver/omnivore/internal/log"
	"github.com/discoriver/omnivore/internal/ossh"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
	"time"
)

func getOSSHConfig(cmd *OmniCommandFlags) *ossh.OmniSSHConfig {
	conf := ossh.NewConfig()

	// FROM FLAG ONLY
	conf.AddJob(&massh.Job{Command: cmd.Command})
	conf.AddHosts(cmd.Hosts)

	// Password auth
	conf.AddPasswordAuth(cmd.Username, cmd.Password)

	// FROM FLAG / CONFIG FILE / DEFAULT
	// Private key auth
	privateKey := viper.GetString(config.PrivateKeyLocConfigKey)
	privateKeyPassword := viper.GetString(config.PrivateKeyPassword)
	// We don't want to fail here because we have a default we'll try, so just log it.
	if err := conf.AddPrivateKeyAuth(privateKey, privateKeyPassword); err != nil {
		log.OmniLog.Warn("Couldn't set private key auth with key \"%s\": %s", privateKey, err)
	} else {
		log.OmniLog.Info("Using private key \"%s\" in auth.", privateKey)
	}

	conf.Config.SSHConfig.Timeout = time.Duration(viper.GetInt(config.SSHTimeoutConfigKey)) * time.Second
	conf.AddWorkerPool(viper.GetInt(config.ConcurrentWorkerPoolConfigKey))

	// Add HostKeyCallback
	if cmd.Insecure {
		log.OmniLog.Warn("Running in INSECURE MODE, known_hosts will be ignored.")
		conf.Config.SSHConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		var err error
		conf.Config.SSHConfig.HostKeyCallback, err = ossh.GetKnownHosts()
		if err != nil {
			log.OmniLog.Fatal("Couldn't get known_hosts for host key callback.", err)
		}
	}

	// SSH_AUTH_SOCK auth
	if err := conf.AddAgent(); err != nil {
		log.OmniLog.Warn("Couldn't add agent (SSH_AUTH_SOCK): %s", err)
	}

	conf.StreamChan = make(chan massh.Result)

	return conf
}
