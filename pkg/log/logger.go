// Package log manages all the gateway's different loggers.
package log

import (
	"fmt"
	"time"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

//nolint:gochecknoglobals // FIXME global var is used by design
var backend logging.Backend

// Logger is an internal abstraction of the underlying logging library.
type Logger struct {
	*logging.Logger
}

// InitBackend initializes the logging backend according to the given configuration.
// If the backend cannot be accessed, an error is returned.
func InitBackend(config conf.LogConfig) (err error) {
	switch config.LogTo {
	case "stdout":
		backend = logging.NewStdoutBackend()
	case "/dev/null", "nul", "NUL":
		backend, _ = logging.NewNoopBackend() //nolint:errcheck // error is always nil
	case "syslog":
		backend, err = newSyslogBackend(config.SyslogFacility)
	default:
		backend, err = logging.NewFileBackend(config.LogTo)
	}

	if err != nil {
		return
	}

	if err := setLevel(config.Level, backend); err != nil {
		record := &logging.Record{
			Logger:    "log",
			Timestamp: time.Now(),
			Level:     logging.ERROR,
			Message:   err.Error(),
		}

		if err := backend.Write(record); err != nil {
			return fmt.Errorf("cannot initialize logging backend: %w", err)
		}
	}

	return
}

// NewLogger initiates a new logger.
func NewLogger(name string) *Logger {
	l := &Logger{Logger: logging.NewLogger(name)}
	l.Logger.SetBackend(backend)

	return l
}

// setLevel sets the level of the logger.
func setLevel(level string, b logging.Backend) error {
	logLevel, err := logging.LevelByName(level)
	if err != nil {
		return fmt.Errorf("cannot set log level to %q: %w", level, err)
	}

	b.SetLevel(logLevel)

	return nil
}
