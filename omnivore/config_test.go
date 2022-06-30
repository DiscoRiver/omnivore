package omnivore

import (
	"github.com/discoriver/omnivore/internal/config"
	"github.com/discoriver/omnivore/internal/test"
	"testing"
	"time"
)

var (
	bogConf = &OmniCommandFlags{
		Command:  "Hello, World",
		Hosts:    []string{"localhost"},
		Username: "runner",
		Password: "password",
		Insecure: true,
	}
)

func TestGetOSSHConfigDefaults_UnitWorkflow(t *testing.T) {
	test.InitTestLogger()

	config.SetConfigDefaults()
	config.InitConfig()

	osshConf := getOSSHConfig(bogConf)

	/*
		We should have password, key, and SSH_AUTH_SOCK auth in osshConf.Config.SSHConfig.Auth. I think this is the only way to see
		if private key is set because I don't think it's possible to look at the underlying type due to the way the ssh
		package is built.

		See: https://cs.opensource.google/go/x/crypto/+/1ad67e1f:ssh/client_auth.go
	*/
	if len(osshConf.Config.SSHConfig.Auth) != 3 {
		t.Logf("Expected auth length of 3, got %d", len(osshConf.Config.SSHConfig.Auth))
		t.Fail()
	}

	if osshConf.Config.SSHConfig.Timeout != time.Duration(config.SSHTimeoutDefault)*time.Second {
		t.Logf("Timeout was not expect default of %d seconds, got %s seconds", config.SSHTimeoutDefault, osshConf.Config.SSHConfig.Timeout)
		t.Fail()
	}

	if osshConf.Config.WorkerPool != config.ConcurrentWorkerPoolDefault {
		t.Logf("Expected worker pool of %d, got %d", config.ConcurrentWorkerPoolDefault, osshConf.Config.WorkerPool)
		t.Fail()
	}
}
