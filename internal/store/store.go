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
	if _, err := os.Stat(s.BaseDir); err == os.ErrNotExist {
		log.OmniLog.Info("BaseDir directory %s does not exist.", s.BaseDir)

		err = os.MkdirAll(s.BaseDir, 0655)
		if err != nil {
			log.OmniLog.Fatal("Couldn't create base directory: %s", err.Error())
		}

		log.OmniLog.Info("BaseDir directory %s was created.", s.BaseDir)
	}

	log.OmniLog.Info("BaseDir directory %s already exists.", s.BaseDir)
}

// InitHistoryDir ensures the directory ~/.omnivore/history exists, creating it if necessary.
func (s *StorageSession) InitHistoryDir() {
	if _, err := os.Stat(s.HistoryDir); err == os.ErrNotExist {
		log.OmniLog.Info("HistoryDir directory %s does not exist.", s.HistoryDir)

		err = os.Mkdir(s.HistoryDir, 0655)
		if err != nil {
			log.OmniLog.Fatal("Couldn't create history directory: %s", err.Error())
		}

		log.OmniLog.Info("HistoryDir directory %s was created.", s.HistoryDir)
	}

	log.OmniLog.Info("HistoryDir directory %s already exists.", s.HistoryDir)
}

// InitSessionDirectory creates a directory to hold session output in ~/.omnivore/history using the unix timestamp
// set by func setSessionTimestamp
func (s *StorageSession) InitSessionDirectory() {
	// Parents should exist or program should've terminated by here.
	if err := os.Mkdir(s.HistoryDir+ string(os.PathSeparator) + s.Timestamp, 0655); err != nil {
		log.OmniLog.Fatal("Couldn't create session directory: %s", err.Error())
	}

	log.OmniLog.Info("HistoryDir directory %s was created.", s.HistoryDir)
}

// InitHostDirs creates host-named directory within ~/.omnivore/history.
func (s *StorageSession) InitHostDir(name string) {
	// Parents should exist or program should've terminated by here.
	if err := os.Mkdir(s.SessionDir + string(os.PathSeparator) + name, 0655); err != nil {
		log.OmniLog.Fatal("Couldn't create host directory: %s", err.Error())
	}

	log.OmniLog.Info("HistoryDir directory %s was created.", s.HistoryDir)
}

