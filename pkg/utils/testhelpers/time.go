package testhelpers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func MustParseTime(tb testing.TB, s string) time.Time {
	tb.Helper()

	t, err := time.Parse(time.RFC3339, s)
	require.NoError(tb, err)

	return t
}

func MustParseDuration(tb testing.TB, s string) time.Duration {
	tb.Helper()

	d, err := time.ParseDuration(s)
	require.NoError(tb, err)

	return d
}
