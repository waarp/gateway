package testhelpers

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/smartystreets/goconvey/convey"
)

// ShouldBeOneOf receives at least 2 parameters. The first is a proposed member
// of the collection that is composed of all the remaining parameters.
// This assertion ensures that the proposed member is in the collection.
func ShouldBeOneOf(actual any, expected ...any) string {
	res := convey.ShouldBeIn(actual, expected...)
	if strings.HasPrefix(res, "Expected ") {
		var vals []string
		for _, exp := range expected {
			vals = append(vals, fmt.Sprintf("%#v", exp))
		}

		exp := strings.Join(vals, ",\n              ")

		return fmt.Sprintf("Expected     '%#v'\nto be one of '%s'\n(but it didn't)!", actual, exp)
	}

	return res
}

func ShouldEqualJSON(actual any, expected ...any) string {
	if n := len(expected); n != 1 {
		return fmt.Sprintf("This assertion requires exactly 1 comparison value "+
			"(you provided %d).", n)
	}

	actualJSON, err1 := json.Marshal(actual)
	if err1 != nil {
		return fmt.Sprintf("Failed to marshal actual value: %v", err1)
	}

	expectedJSON, err2 := json.Marshal(expected[0])
	if err2 != nil {
		return fmt.Sprintf("Failed to marshal expected value: %v", err2)
	}

	return convey.ShouldEqualJSON(string(actualJSON), string(expectedJSON))
}

func ShouldSucceedAfter[T any](duration time.Duration, assert convey.Assertion,
	getActual func() T, expected any,
) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			// c.So(getActual(), assert, expected)
			return false
		default:
			if assert(getActual(), expected) == "" {
				return true
			}
		}
	}
}
