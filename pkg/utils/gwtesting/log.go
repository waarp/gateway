// Package gwtesting contains utilities for testing the gateway.
package gwtesting

import (
	"testing"

	"code.waarp.fr/lib/log"
	"github.com/stretchr/testify/require"
)

func Logger(tb testing.TB) *log.Logger {
	tb.Helper()

	back, err := log.NewBackend(log.LevelDebug, log.Stdout, "", "")
	require.NoError(tb, err)

	return back.NewLogger(tb.Name())
}
