package common

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDurationPart(t *testing.T) {
	for _, test := range []struct {
		value, toUnit, expected string
		err                     error
	}{
		{value: "6020s", toUnit: "h", expected: "1"},
		{value: "6020s", toUnit: "m", expected: "40"},
		{value: "6020s", toUnit: "s", expected: "20"},
		{value: "62500ms", toUnit: "m", expected: "1"},
		{value: "62500ms", toUnit: "s", expected: "2"},
		{value: "62500ms", toUnit: "ms", expected: "500"},
	} {
		t.Run(fmt.Sprintf("%s to %s", test.value, test.toUnit), func(t *testing.T) {
			res, err := getDurationPart(test.value, test.toUnit)
			if err != nil {
				assert.Equal(t, test.err, err)
			} else {
				assert.Equal(t, test.expected, res)
			}
		})
	}
}
