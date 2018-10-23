package log

import (
	"code.bcarlin.xyz/go/logging"
)

// Logger is an internal abstraction of the underlying logging library
type Logger struct {
	*logging.Logger
}

// NewLogger initiates a new logger
func NewLogger() *Logger {
	return &Logger{
		Logger: logging.GetLogger("waarp-gateway-ng"),
	}
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

