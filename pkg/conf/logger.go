package conf

import (
	"errors"
	"fmt"
	"strings"

	"code.waarp.fr/lib/log"
)

var (
	//nolint:gochecknoglobals //global var is more convenient here
	pool = &log.BackendPool{}

	ErrUnknownLogLevel = errors.New("unknown log level")
)

func levelFromString(level string) log.Level {
	switch {
	case strings.EqualFold(level, log.LevelTrace.String()):
		return log.LevelTrace
	case strings.EqualFold(level, log.LevelDebug.String()):
		return log.LevelDebug
	case strings.EqualFold(level, log.LevelInfo.String()):
		return log.LevelInfo
	case strings.EqualFold(level, log.LevelNotice.String()):
		return log.LevelNotice
	case strings.EqualFold(level, log.LevelWarning.String()):
		return log.LevelWarning
	case strings.EqualFold(level, log.LevelError.String()):
		return log.LevelError
	case strings.EqualFold(level, log.LevelCritical.String()):
		return log.LevelCritical
	case strings.EqualFold(level, log.LevelAlert.String()):
		return log.LevelAlert
	case strings.EqualFold(level, log.LevelFatal.String()):
		return log.LevelFatal
	default:
		return 0
	}
}

func NewLogBackend(level, logTo, facility, tag string) (*log.Backend, error) {
	lvl := levelFromString(level)
	if lvl == 0 {
		return nil, fmt.Errorf("%w: %s", ErrUnknownLogLevel, level)
	}

	backend, err := log.NewBackend(lvl, logTo, facility, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize backend: %w", err)
	}

	return backend, nil
}

func InitBackend(level, logTo, facility, tag string) error {
	backend, err := NewLogBackend(level, logTo, facility, tag)
	if err != nil {
		return err
	}

	pool.AddBackend(backend)

	return nil
}

func GetLogger(name string) *log.Logger {
	return pool.NewLogger(name)
}
