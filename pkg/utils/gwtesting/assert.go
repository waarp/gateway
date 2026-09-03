package gwtesting

import (
	"encoding/json"
	"fmt"
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

type equalFunc[A any] interface {
	Equal(A) bool
}

func Equal[T any](tb testing.TB, expected, actual T) {
	tb.Helper()

	equalizer, isEqualizer := any(expected).(equalFunc[T])
	if !isEqualizer {
		assert.Equal(tb, expected, actual)

		return
	}

	if !equalizer.Equal(actual) {
		assert.Fail(tb, fmt.Sprintf(`Expected "%v" to be equal to "%v"`, expected, actual))
	}
}
