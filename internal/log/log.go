package log

import (
	"log"
	"os"
)

var (
	defaultLogLoc = "/var/log/omnivore.log"
	testLogLoc    = os.Stdout
)

var (
	Warn  *log.Logger
	Info  *log.Logger
	Error *log.Logger
)

func InitLogger() {
	file, err := os.OpenFile(defaultLogLoc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	Info = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func InitTestLogger() {
	Info = log.New(testLogLoc, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(testLogLoc, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(testLogLoc, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}
