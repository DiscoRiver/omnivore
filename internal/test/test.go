package test

import (
	"os"

	"github.com/discoriver/omnivore/internal/log"
)

func InitTestLogger() {
	log.OmniLog = &log.OmniLogger{FileOutput: os.Stdout}
	log.OmniLog.Init()
}
