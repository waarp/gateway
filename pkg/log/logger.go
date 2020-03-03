// Package log manages all the gateway's different loggers.
package log

import (
	"time"

	"code.bcarlin.xyz/go/logging"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
)

var backend logging.Backend

// Logger is an internal abstraction of the underlying logging library
type Logger struct {
	*logging.Logger
}

// InitBackend initializes the logging backend according to the given configuration.
// If the backend cannot be accessed, an error is returned.
func InitBackend(conf conf.LogConfig) (err error) {
	switch conf.LogTo {
	case "stdout":
		backend = logging.NewStdoutBackend()
	case "syslog":
		backend, err = logging.NewSyslogBackend(conf.SyslogFacility, "waarp-manager-ng")
	default:
		backend, err = logging.NewFileBackend(conf.LogTo)
	}
	if err != nil {
		return
	}

	if err := setLevel(conf.Level, backend); err != nil {
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
