// Package gwtesting contains utilities for testing the gateway.
package gwtesting

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
)

const testLogLevel = "DEBUG"

//nolint:gochecknoinits //init is needed here
func init() {
	if err := logging.SetLogBackend(testLogLevel, log.Stdout, "", ""); err != nil {
		panic(err)
	}
}

func Logger(tb testing.TB, opts ...logtest.LogOpt) *log.Logger {
	tb.Helper()

	return logtest.GetTestLogger(tb, opts...)
}
