package backtest

import (
	"os"
	"strconv"
	"testing"
)

// ciEnvVar is set by GitLab CI (and by most other CI runners) in every job.
const ciEnvVar = "CI"

// inCI reports whether the tests run on a CI runner. A value that is not a
// boolean still means CI, but some tools set the variable to false to state
// the opposite.
func inCI() bool {
	value := os.Getenv(ciEnvVar)
	if value == "" {
		return false
	}

	isCI, err := strconv.ParseBool(value)

	return err != nil || isCI
}

// EnvOrSkip returns the value of the given environment variable. When it is
// unset or empty, the test is skipped: the cloud backend tests need real
// credentials, which only the environment can provide. In CI those credentials
// are expected, so a missing one fails the test instead of silently turning
// the pipeline green.
func EnvOrSkip(tb testing.TB, name string) string {
	tb.Helper()

	value := os.Getenv(name)
	if value == "" {
		if inCI() {
			tb.Fatalf("the %q environment variable is not set", name)
		}

		tb.Skipf("skipping test: the %q environment variable is not set", name)
	}

	return value
}
