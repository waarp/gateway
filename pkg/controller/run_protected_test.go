package controller

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
)

func TestRunProtectedTurnsAPanicIntoAnError(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	logger := logtest.GetTestLogger(t, logtest.WithLevel("INFO"), logtest.WithWriter(&logs))

	// A transfer that panics is reported as failed, and the goroutine survives.
	var err error

	require.NotPanics(t, func() {
		err = runProtected(logger, 42, func() error { panic("nil pointer in a protocol module") })
	})
	require.ErrorIs(t, err, errTransferPanicked)
	assert.Contains(t, logs.String(), "Transfer n°42 panicked")
	assert.Contains(t, logs.String(), "nil pointer in a protocol module")

	// A transfer that fails normally keeps its error; one that succeeds keeps nil.
	boom := errors.New("boom") //nolint:err113 // test
	assert.ErrorIs(t, runProtected(logger, 43, func() error { return boom }), boom)
	assert.NoError(t, runProtected(logger, 44, func() error { return nil }))
}
