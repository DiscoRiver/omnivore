package log

import (
	"fmt"
	"log"
	"os"
)

var (
	defaultLogLoc = "/var/log/omnivore.log"

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
	var s string
	if len(args) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, args...)
	}

	o.info.Println(s)

	o.Messages = append(o.Messages, s)
}

func (o *OmniLogger) Warn(format string, args ...interface{}) {
	var s string
	if len(args) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, args...)
	}

	o.warn.Println(s)

	o.Messages = append(o.Messages, s)
}

func (o *OmniLogger) Error(format string, args ...interface{}) {
	var s string
	if len(args) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, args...)
	}

	o.er.Println(s)

	o.Messages = append(o.Messages, s)
}

func (o *OmniLogger) Fatal(format string, args ...interface{}) {
	var s string
	if len(args) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, args...)
	}

	o.fatal.Println(s)

	o.Messages = append(o.Messages, s)

	os.Exit(1)
}

func (o *OmniLogger) Init() {
	if o.FileOutput == nil {
		var err error
		o.FileOutput, err = os.OpenFile(defaultLogLoc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}
	}

	o.info = log.New(o.FileOutput, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	o.warn = log.New(o.FileOutput, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	o.er = log.New(o.FileOutput, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	o.fatal = log.New(o.FileOutput, "FATAL: ", log.Ldate|log.Ltime|log.Lshortfile)
}
