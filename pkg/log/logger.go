// Package log manages all the gateway's different loggers.
package log

import (
	"time"

	"code.bcarlin.xyz/go/logging"
)

var backend logging.Backend

// Logger is an internal abstraction of the underlying logging library
type Logger struct {
	*logging.Logger
}

// InitBackend initializes the logging backend according to the given configuration.
// If the backend cannot be accessed, an error is returned.
func InitBackend(level, logTo, facility string) (err error) {
	switch logTo {
	case "stdout":
		backend = logging.NewStdoutBackend()
	case "/dev/null", "nul", "NUL":
		backend, _ = logging.NewNoopBackend()
	case "syslog":
		backend, err = newSyslogBackend(facility)
	default:
		backend, err = logging.NewFileBackend(logTo)
	}
	if err != nil {
		return
	}

	if err := setLevel(level, backend); err != nil {
		record := &logging.Record{
			Logger:    "log",
			Timestamp: time.Now(),
			Level:     logging.ERROR,
			Message:   err.Error(),
		}
		if err := backend.Write(record); err != nil {
			return err
		}
	}
	return
}

// NewLogger initiates a new logger
func NewLogger(name string) *Logger {
	l := &Logger{Logger: logging.NewLogger(name)}
	l.Logger.SetBackend(backend)
	return l
}

// setLevel sets the level of the logger
func setLevel(level string, b logging.Backend) error {
	logLevel, err := logging.LevelByName(level)
	if err != nil {
		return err
	}
	b.SetLevel(logLevel)
	return nil
}
