package log

import (
	"fmt"
	"github.com/discoriver/omnivore/internal/path"
	"log"
	"os"
)

var (
	defaultLogLoc = "~/.omnivore/log.txt"

	OmniLog *OmniLogger
)

type OmniLogger struct {
	// Messages stores logs for a single Omnivore session.
	Messages []string

	FileOutput *os.File

	warn  *log.Logger
	info  *log.Logger
	er    *log.Logger
	fatal *log.Logger
}

func (o *OmniLogger) Info(format string, args ...interface{}) {
	infoPrefix := "INFO: "
	var s string
	if len(args) == 0 {
		s = infoPrefix + format
	} else {
		s = infoPrefix + fmt.Sprintf(format, args...)
	}

	o.info.Println(s)

	o.Messages = append(o.Messages, s)
}

func (o *OmniLogger) Warn(format string, args ...interface{}) {
	warnPrefix := "WARNING: "
	var s string
	if len(args) == 0 {
		s = warnPrefix + format
	} else {
		s = warnPrefix + fmt.Sprintf(format, args...)
	}

	o.warn.Println(s)

	o.Messages = append(o.Messages, s)
}

func (o *OmniLogger) Error(format string, args ...interface{}) {
	errorPrefix := "ERROR: "
	var s string
	if len(args) == 0 {
		s = errorPrefix + format
	} else {
		s = errorPrefix + fmt.Sprintf(format, args...)
	}

	o.er.Println(s)

	o.Messages = append(o.Messages, s)
}

func (o *OmniLogger) Fatal(format string, args ...interface{}) {
	fatalPrefix := "FATAL: "
	var s string
	if len(args) == 0 {
		s = fatalPrefix + format
	} else {
		s = fatalPrefix + fmt.Sprintf(format, args...)
	}

	o.fatal.Println(s)

	o.Messages = append(o.Messages, s)

	os.Exit(1)
}

func (o *OmniLogger) Init() {
	defaultLogExpanded, err := path.ExpandUserHome(defaultLogLoc)
	if err != nil {
		log.Fatalf("Couldn't expand user home: %s", err)
	}

	if o.FileOutput == nil {
		var err error
		o.FileOutput, err = os.OpenFile(defaultLogExpanded, os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Fatal(err)
		}
	}

	o.info = log.New(o.FileOutput, "", log.Ldate|log.Lmicroseconds)
	o.warn = log.New(o.FileOutput, "", log.Ldate|log.Lmicroseconds)
	o.er = log.New(o.FileOutput, "", log.Ldate|log.Lmicroseconds)
	o.fatal = log.New(o.FileOutput, "", log.Ldate|log.Lmicroseconds)

	o.info.Println("Omnivore Started.")
}
