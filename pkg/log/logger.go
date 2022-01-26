// Package log manages all the gateway's different loggers.
package log

import (
	"fmt"
	"time"

	"code.bcarlin.xyz/go/logging"
)

//nolint:gochecknoglobals // FIXME global var is used by design
var backend logging.Backend

// Logger is an internal abstraction of the underlying logging library.
type Logger struct {
	*logging.Logger
}

// InitBackend initializes the logging backend according to the given configuration.
// If the backend cannot be accessed, an error is returned.
func InitBackend(level, logTo, facility string) error {
	var err error

	switch logTo {
	case "stdout":
		backend = logging.NewStdoutBackend()
	case "stderr":
		backend = logging.NewStderrBackend()
	case "/dev/null", "nul", "NUL":
		backend, _ = logging.NewNoopBackend() //nolint:errcheck // error is always nil
	case "syslog":
		backend, err = newSyslogBackend(facility)
	default:
		backend, err = logging.NewFileBackend(logTo)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize log backend: %w", err)
	}

	if err1 := setLevel(level, backend); err1 != nil {
		record := &logging.Record{
			Logger:    "log",
			Timestamp: time.Now(),
			Level:     logging.ERROR,
			Message:   err1.Error(),
		}

		if err2 := backend.Write(record); err2 != nil {
			return fmt.Errorf("cannot initialize logging backend: %w", err2)
		}

		return err1
	}

	return nil
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
