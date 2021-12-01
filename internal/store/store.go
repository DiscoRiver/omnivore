// Package store provides data storage for omnivore output.
package store

import (
	"fmt"
	"github.com/discoriver/omnivore/internal/log"
	"github.com/discoriver/omnivore/internal/path"
	"os"
	"time"
)

var (
	base = ".omnivore"
	history = "history"
)

// StorageSession holds directory information about the current application run state.
type StorageSession struct {
	Timestamp string
	UserHome string
	BaseDir    string
	HistoryDir string
	SessionDir string

	hostDirs []string
}

func NewStorageSession() *StorageSession {
	s := &StorageSession{}

	var err error
	if s.UserHome, err = path.GetUserHome(); err != nil {
		log.OmniLog.Fatal("Unable to get user home: %s", err.Error())
	}

	s.BaseDir = s.UserHome + string(os.PathSeparator) + base
	s.HistoryDir = s.BaseDir + string(os.PathSeparator) + history

	s.Timestamp = fmt.Sprintf("%d", time.Now().UnixNano())
	s.SessionDir = s.HistoryDir + string(os.PathSeparator) + s.Timestamp

	// Just initialise the directories here
	s.InitBaseDir()
	s.InitHistoryDir()
	s.InitSessionDirectory()

	return s
}

// InitBaseDir ensures the directory ~/.omnivore/ exists, creating it if necessary.
func (s *StorageSession) InitBaseDir() {
	if _, err := os.Stat(s.BaseDir); err != nil {
		if os.IsNotExist(err) {
			log.OmniLog.Info("Base directory %s does not exist.", s.BaseDir)

			err = os.MkdirAll(s.BaseDir, 0755)
			if err != nil {
				log.OmniLog.Fatal("Couldn't create base directory: %s", err.Error())
			}

			log.OmniLog.Info("Base directory %s was created.", s.BaseDir)
			return
		}
	}

	log.OmniLog.Info("Base directory %s already exists.", s.BaseDir)
}

// InitHistoryDir ensures the directory ~/.omnivore/history exists, creating it if necessary.
func (s *StorageSession) InitHistoryDir() {
	if _, err := os.Stat(s.HistoryDir); err != nil {
		if os.IsNotExist(err) {
			log.OmniLog.Info("History directory %s does not exist.", s.HistoryDir)

			err = os.Mkdir(s.HistoryDir, 0755)
			if err != nil {
				log.OmniLog.Fatal("Couldn't create history directory: %s", err.Error())
			}

			log.OmniLog.Info("History directory %s was created.", s.HistoryDir)
			return
		} else {
			log.OmniLog.Fatal("Couldn't create history directory: %s", err.Error())
		}
	}

	log.OmniLog.Info("History directory %s already exists.", s.HistoryDir)
}

// InitSessionDirectory creates a directory to hold session output in ~/.omnivore/history using the unix timestamp
// set by func setSessionTimestamp
func (s *StorageSession) InitSessionDirectory() {
	// Parents should exist or program should've terminated by here.
	if err := os.Mkdir(s.SessionDir, 0755); err != nil {
		log.OmniLog.Fatal("Couldn't create session directory: %s", err.Error())
	}

	log.OmniLog.Info("Session directory %s was created.", s.SessionDir)
}

// InitHostDirs creates host-named directory within ~/.omnivore/history.
func (s *StorageSession) InitHostDir(name string) {
	// Parents should exist or program should've terminated by here.
	if err := os.Mkdir(s.SessionDir + string(os.PathSeparator) + name, 0755); err != nil {
		log.OmniLog.Fatal("Couldn't create host directory: %s", err.Error())
	}

	log.OmniLog.Info("History directory %s was created.", s.HistoryDir)
}

