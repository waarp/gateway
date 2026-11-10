package articles

import (
	"encoding/json"
	"slices"
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArticleLengthsJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		lengths  []uint16
		expected string
	}{
		{"empty", nil, `[]`},
		{"single", []uint16{12}, `[12]`},
		{"distinct", []uint16{12, 0, 47, 4}, `[12,0,47,4]`},
		{"repeated", []uint16{12, 12, 12, 0, 47, 47}, `[{"l":12,"c":3},0,{"l":47,"c":2}]`},
		{"uniform", slices.Repeat([]uint16{80}, 10000), `[{"l":80,"c":10000}]`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			lengths := NewLengths(test.lengths...)

			marshalled, err := json.Marshal(lengths)
			require.NoError(t, err)
			assert.JSONEq(t, test.expected, string(marshalled))

			var unmarshalled Lengths
			require.NoError(t, json.Unmarshal(marshalled, &unmarshalled))
			assert.Equal(t, test.lengths, slices.Collect(unmarshalled.All()))
			assert.Equal(t, len(test.lengths), unmarshalled.Len())
		})
	}
}

func TestArticleLengthsLegacyJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		expected []uint16
	}{
		{"null", `null`, nil},
		{"empty array", `[]`, nil},
		{"array", `[12, 12, 12, 0, 47]`, []uint16{12, 12, 12, 0, 47}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var lengths Lengths
			require.NoError(t, json.Unmarshal([]byte(test.json), &lengths))
			assert.Equal(t, test.expected, slices.Collect(lengths.All()))
		})
	}
}

func TestArticleLengthsMethods(t *testing.T) {
	t.Parallel()

	var nilLengths *Lengths
	assert.Equal(t, 0, nilLengths.Len())
	assert.Equal(t, uint16(0), nilLengths.Max())
	assert.Empty(t, slices.Collect(nilLengths.All()))
	assert.True(t, nilLengths.Uniform())

	lengths := NewLengths(5, 5, 300, 7)
	assert.Equal(t, 4, lengths.Len())
	assert.Equal(t, uint16(300), lengths.Max())
	assert.Equal(t, `[{"l":5,"c":2},300,7]`, lengths.String())
	assert.False(t, lengths.Uniform())
	assert.True(t, NewLengths(5, 5, 5).Uniform())
	assert.True(t, NewLengths().Uniform())

	// Lengths read back keep the runs they were written with.
	var read Lengths
	require.NoError(t, json.Unmarshal([]byte(`[5,5]`), &read))
	assert.True(t, read.Uniform())

	lengths.Add(7)
	assert.Equal(t, `[{"l":5,"c":2},300,{"l":7,"c":2}]`, lengths.String())

	// The iterator can be stopped early.
	for length := range lengths.All() {
		assert.Equal(t, uint16(5), length)

		break
	}
}

// The values of the format are the names that the PeSIT library gives its
// formats, which the PeSIT module compares them with.
func TestFormatValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, pesit.FormatFixed.String(), FormatFixed)
	assert.Equal(t, pesit.FormatVariable.String(), FormatVariable)
}
