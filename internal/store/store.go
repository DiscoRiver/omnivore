// Package store provides data storage for omnivore output.
package store

import (
	"fmt"
	ovlog "github.com/discoriver/omnivore/internal/log"
	"github.com/discoriver/omnivore/internal/path"
	"github.com/discoriver/omnivore/pkg/group"
	"log"
	"os"
	"time"
)

var (
	base    = ".omnivore"
	history = "history"

	defaultDirPermissions = os.FileMode(0775)

	// Session is initalised with NewStorageSession, used for file operations.
	Session *StorageSession

	// Trouble writing host output?
	hostFileWriteFailure bool
)

// StorageSession holds directory information about the current application run state.
type StorageSession struct {
	Timestamp  string
	UserHome   string
	BaseDir    string
	HistoryDir string
	SessionDir string
}

func NewStorageSession() {
	Session = &StorageSession{}

	var err error
	if Session.UserHome, err = path.GetUserHome(); err != nil {
		log.Fatalf("Unable to get user home: %s\n", err.Error())
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

// Read reads the given file from a storage session.
func (s *StorageSession) Read(name string) ([]byte, error) {
	filePath := s.SessionDir + string(os.PathSeparator) + name
	return os.ReadFile(filePath)
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

// InitHostDir creates host-named directory within ~/.omnivore/history.
func (s *StorageSession) InitHostDir(name string) {
	// Parents should exist or program should've terminated by here.
	if err := os.Mkdir(s.SessionDir+string(os.PathSeparator)+name, defaultDirPermissions); err != nil {
		log.Fatalf("Couldn't create host directory: %s\n", err.Error())
	}
}

// WriteOutputFileForHost writes the content of an identifying pair to a file, for future processing out of memory. The identifying pair's key should always be the hostname.
func (s *StorageSession) WriteOutputFileForHost(idp *group.IdentifyingPair) {
	// Key should always be the hostname in Omnivore
	filePath := s.SessionDir + string(os.PathSeparator) + idp.Key

	// Don't log fatal here, as the application can still be allowed to function in-memory.
	if err := os.WriteFile(filePath, idp.Value, defaultDirPermissions); err != nil {
		ovlog.OmniLog.Warn("Couldn't write host output file: %s", err.Error())

		// TODO: Have application continue to run in-memory if writing these files fails.
		hostFileWriteFailure = true
	}
	ovlog.OmniLog.Info("Written host output file for %s", idp.Key)
}
