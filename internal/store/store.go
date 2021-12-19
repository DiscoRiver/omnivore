// Package store provides data storage for omnivore output.
package store

import (
	"fmt"
	"github.com/discoriver/omnivore/internal/path"
	"log"
	"os"
	"time"
)

var (
	base    = ".omnivore"
	history = "history"

	defaultDirPermissions = os.FileMode(0775)

	Session *StorageSession
)

// StorageSession holds directory information about the current application run state.
type StorageSession struct {
	Timestamp  string
	UserHome   string
	BaseDir    string
	HistoryDir string
	SessionDir string

	hostDirs []string
}

func NewStorageSession() {
	Session = &StorageSession{}

	var err error
	if Session.UserHome, err = path.GetUserHome(); err != nil {
		log.Fatalf("Unable to get user home: %Session\n", err.Error())
	}

	Session.BaseDir = Session.UserHome + string(os.PathSeparator) + base
	Session.HistoryDir = Session.BaseDir + string(os.PathSeparator) + history

	Session.Timestamp = fmt.Sprintf("%d", time.Now().UnixNano())
	Session.SessionDir = Session.HistoryDir + string(os.PathSeparator) + Session.Timestamp

	// Just initialise the directories here
	Session.InitBaseDir()
	Session.InitHistoryDir()
	Session.InitSessionDirectory()
}

// InitBaseDir ensures the directory ~/.omnivore/ exists, creating it if necessary.
func (s *StorageSession) InitBaseDir() {
	if _, err := os.Stat(s.BaseDir); err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(s.BaseDir, defaultDirPermissions)
			if err != nil {
				log.Fatalf("Couldn't create base directory: %s\n", err.Error())
			}
			return
		}
	}
}

// InitHistoryDir ensures the directory ~/.omnivore/history exists, creating it if necessary.
func (s *StorageSession) InitHistoryDir() {
	if _, err := os.Stat(s.HistoryDir); err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(s.HistoryDir, defaultDirPermissions)
			if err != nil {
				log.Fatalf("Couldn't create history directory: %s\n", err.Error())
			}
			return
		} else {
			log.Fatalf("Couldn't create history directory: %s\n", err.Error())
		}
	}
}

// InitSessionDirectory creates a directory to hold session output in ~/.omnivore/history using the unix timestamp
// set by func setSessionTimestamp
func (s *StorageSession) InitSessionDirectory() {
	// Parents should exist or program should've terminated by here.
	if err := os.Mkdir(s.SessionDir, defaultDirPermissions); err != nil {
		log.Fatalf("Couldn't create session directory: %s\n", err.Error())
	}
}

// InitHostDirs creates host-named directory within ~/.omnivore/history.
func (s *StorageSession) InitHostDir(name string) {
	// Parents should exist or program should've terminated by here.
	if err := os.Mkdir(s.SessionDir+string(os.PathSeparator)+name, defaultDirPermissions); err != nil {
		log.Fatalf("Couldn't create host directory: %s\n", err.Error())
	}

}
