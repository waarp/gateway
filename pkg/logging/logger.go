// Package logging provides utilities for logging.
package logging

import (
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

var (
	//nolint:gochecknoglobals //global var is more convenient here
	back = &log.Backend{Output: io.Discard, Level: log.LevelInfo}

	ErrUnknownLogLevel = errors.New("unknown log level")
)

func NewLogBackend(level, logTo, facility, tag string) (*log.Backend, error) {
	lvl, err := log.LevelByName(level)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnknownLogLevel, level)
	}

	backend, err := log.NewBackend(lvl, logTo, facility, tag)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize backend: %w", err)
	}

	return backend, nil
}

func SetLogBackend(level, logTo, facility, tag string) error {
	backend, err := NewLogBackend(level, logTo, facility, tag)
	if err != nil {
		return err
	}

	back = backend

	return nil
}

func NewLogger(name string) *log.Logger {
	return back.NewLogger(name)
}

func Discard() *log.Logger {
	//nolint:errcheck //never returns an error
	backend, _ := log.NewBackend(log.LevelInfo, log.Discard, "", "")

	return backend.NewLogger("")
}
