package log

import (
	"code.bcarlin.xyz/go/logging"
)

// Logger is an internal abstraction of the underlying logging library
type Logger struct {
	*logging.Logger
}

// NewLogger initiates a new logger
func NewLogger(name string) *Logger {
	return &Logger{Logger: logging.GetLogger(name)}
}

// SetLevel sets the level of the logger
func (l *Logger) SetLevel(level string) error {
	logLevel, err := logging.LevelByName(level)
	if err != nil {
		return err
	}
	l.Logger.SetLevel(logLevel)
	return nil
}

// SetOutput sets the outpur of the underlying backend.
// It expects out to be a file path. It can also be 'stdout' to log to the
// standard output or 'syslog' to log to a syslog daemon .
func (l *Logger) SetOutput(out string, syslogfacility string) error {
	var (
		b   logging.Backend
		err error
	)

	switch out {
	case "stdout":
		b = logging.NewStdoutBackend()
	case "syslog":
		b, err = logging.NewSyslogBackend(syslogfacility, "waarp-manager-ng")
	default:
		b, err = logging.NewFileBackend(out)
	}

	l.Logger.SetBackend(b)

	return err
}
