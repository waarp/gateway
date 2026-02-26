package logtest

import (
	"io"
	"log/slog"
	"testing"

	"code.waarp.fr/lib/log/v2"
	"github.com/stretchr/testify/require"
)

type LogOpt func(testing.TB, *log.Handler)

func GetTestLogger(tb testing.TB, opts ...LogOpt) *log.Logger {
	tb.Helper()

	leveler := &slog.LevelVar{}
	handler := log.NewLogHandler(tb.Output(), leveler, tb.Name(), nil)
	handler.SetLevel(log.LevelInfo)

	for _, opt := range opts {
		opt(tb, handler)
	}

	slogger := slog.New(handler)
	logger := log.NewLogger(slogger)

	return logger
}

func WithLevel(level string) LogOpt {
	return func(tb testing.TB, handler *log.Handler) {
		tb.Helper()
		lvl, err := log.LevelByName(level)
		require.NoError(tb, err)
		handler.SetLevel(lvl)
	}
}

func WithWriter(w io.Writer) LogOpt {
	//nolint:staticcheck //we need to reassign here
	return func(tb testing.TB, handler *log.Handler) {
		tb.Helper()
		handler.SetOutput(w)
	}
}

func WithName(name string) LogOpt {
	return func(tb testing.TB, handler *log.Handler) {
		tb.Helper()
		handler.SetGroup(name)
	}
}
