package logtest

import (
	"io"
	"testing"

	"code.bcarlin.net/go/logging"
	"code.waarp.fr/lib/log"
	"github.com/stretchr/testify/require"
)

type LogOpt func(testing.TB, logging.Backend, *log.Logger)

func GetTestLogger(tb testing.TB, opts ...LogOpt) *log.Logger {
	tb.Helper()

	backend := logging.NewIoBackend(tb.Output())
	backend.SetLevel(logging.Info)
	logger := &log.Logger{Logger: logging.NewLogger(tb.Name())}
	logger.SetBackend(backend)

	for _, opt := range opts {
		opt(tb, backend, logger)
	}

	return logger
}

func WithLevel(level string) LogOpt {
	return func(tb testing.TB, backend logging.Backend, _ *log.Logger) {
		tb.Helper()
		lvl, err := logging.LevelByName(level)
		require.NoError(tb, err)
		backend.SetLevel(lvl)
	}
}

func WithWriter(w io.Writer) LogOpt {
	//nolint:staticcheck //we need to reassign here
	return func(tb testing.TB, backend logging.Backend, logger *log.Logger) {
		tb.Helper()
		backend = logging.NewIoBackend(w)
		logger.SetBackend(backend)
	}
}

func WithName(name string) LogOpt {
	return func(tb testing.TB, _ logging.Backend, logger *log.Logger) {
		tb.Helper()
		logger.Logger = logging.NewLogger(name)
	}
}
