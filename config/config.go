package config

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	DefaultPeriod       = 30
	DefaultResponseCode = 200
	DefaultMethod       = http.MethodHead
)

const DefaultDatabasePath = "updawg.db"

var logFile *os.File

func init() {
	logFilePath := os.Getenv("UPDAWG_LOG")
	if logFilePath == "" {
		logFile = os.Stdout
	} else {
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Panicln("could not open log file", logFilePath+":", err)
		}

		logFile = file
	}
}

func GetDatabaseUri() string {
	path := os.Getenv("UPDAWG_DB")
	if path == "" {
		path = filepath.Join(os.Getenv("PWD"), DefaultDatabasePath)
	}

	return "file://" + path + "?_pragma=journal_mode(WAL)"
}

func GetLogFile() *os.File {
	return logFile
}
