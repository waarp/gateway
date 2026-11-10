package api

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullTypeJSON(t *testing.T) {
	t.Parallel()

	n := NewNullable[*string](nil)
	fmt.Print(n)

	t.Run("marshal", func(t *testing.T) {
		t.Parallel()

		for val, expOutput := range map[any]string{
			123:        `123`,
			1.23:       `1.23`,
			"foobar":   `"foobar"`,
			"":         `""`,
			true:       `true`,
			struct{}{}: `{}`,
			nil:        `null`,
		} {
			t.Run(fmt.Sprint(val), func(t *testing.T) {
				t.Parallel()

				nullVal := NewNullable(val)
				output, err := json.Marshal(nullVal)
				require.NoError(t, err)
				assert.Equal(t, expOutput, string(output))
			})
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		t.Parallel()

		for input, expVal := range map[string]any{
			`123`:      float64(123),
			`1.23`:     1.23,
			`"foobar"`: "foobar",
			`""`:       "",
			`true`:     true,
			`null`:     nil,
		} {
			t.Run(input, func(t *testing.T) {
				t.Parallel()

				var nullVal Nullable[any]
				err := json.Unmarshal([]byte(input), &nullVal)
				require.NoError(t, err)
				assert.Equal(t, expVal != nil, nullVal.Valid)
				assert.Equal(t, expVal, nullVal.Value)
			})
		}
	})
}

func TestNullTypeYAML(t *testing.T) {
	t.Parallel()

	n := NewNullable[*string](nil)
	fmt.Print(n)

	t.Run("marshal", func(t *testing.T) {
		t.Parallel()

		for val, expOutput := range map[any]string{
			123:        `123`,
			1.23:       `1.23`,
			"foobar":   `foobar`,
			"":         `""`,
			true:       `true`,
			struct{}{}: `{}`,
			nil:        `null`,
		} {
			t.Run(fmt.Sprint(val), func(t *testing.T) {
				t.Parallel()

				nullVal := NewNullable(val)
				output, err := yaml.Marshal(nullVal)
				require.NoError(t, err)
				assert.Equal(t, expOutput+"\n", string(output))
			})
		}
	})

	t.Run("unmarshal", func(t *testing.T) {
		t.Parallel()

		for input, expVal := range map[string]any{
			`123`:      uint64(123),
			`1.23`:     1.23,
			`"foobar"`: "foobar",
			`foobar`:   "foobar",
			`""`:       "",
			`true`:     true,
			`null`:     nil,
		} {
			t.Run(input, func(t *testing.T) {
				t.Parallel()

				var nullVal Nullable[any]
				err := yaml.Unmarshal([]byte(input), &nullVal)
				require.NoError(t, err)
				assert.Equal(t, expVal != nil, nullVal.Valid)
				assert.Equal(t, expVal, nullVal.Value)
			})
		}
	})
}
