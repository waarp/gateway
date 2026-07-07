package gwtesting

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func JSONEqual(tb testing.TB, expected, actual any) {
	tb.Helper()
	jsExp, err := json.Marshal(expected)
	require.NoError(tb, err)
	jsAct, err := json.Marshal(actual)
	require.NoError(tb, err)

	assert.JSONEq(tb, string(jsExp), string(jsAct))
}
