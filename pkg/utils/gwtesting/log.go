// Package gwtesting contains utilities for testing the gateway.
package gwtesting

import (
	"testing"

	"code.waarp.fr/lib/log"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
)

const testLogLevel = "DEBUG"

//nolint:gochecknoinits //init is needed here
func init() {
	if err := logging.AddLogBackend(testLogLevel, log.Stdout, "", ""); err != nil {
		panic(err)
	}
}

func Logger(tb testing.TB) *log.Logger {
	tb.Helper()

	return LoggerWithName(tb, tb.Name())
}

func LoggerWithName(tb testing.TB, name string) *log.Logger {
	tb.Helper()

	back, err := log.NewBackend(log.LevelDebug, log.Stdout, "", "")
	require.NoError(tb, err)

	return back.NewLogger(name)
}
