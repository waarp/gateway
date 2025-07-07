package testhelpers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func MustMarshalJSON(tb testing.TB, v any) string {
	tb.Helper()

	b, err := json.Marshal(v)
	require.NoError(tb, err)

	return string(b)
}
