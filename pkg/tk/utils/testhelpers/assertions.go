package testhelpers

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/smartystreets/assertions"
)

// ShouldBeOneOf receives at least 2 parameters. The first is a proposed member
// of the collection that is composed of all the remaining parameters.
// This assertion ensures that the proposed member is in the collection.
func ShouldBeOneOf(actual interface{}, expected ...interface{}) string {
	res := assertions.ShouldBeIn(actual, expected...)
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

func ShouldBeErrorType(actual interface{}, expected ...interface{}) string {
	if n := len(expected); n != 1 {
		return fmt.Sprintf("This assertion requires exactly 1 comparison value "+
			"(you provided %d).", n)
	}

	err, ok1 := actual.(error)
	if !ok1 || err == nil {
		return fmt.Sprintf("Expected a non-nil error value (but was '%v' instead)!",
			reflect.TypeOf(actual))
	}

	val := reflect.ValueOf(expected[0])
	if val.Kind() != reflect.Ptr || !isError(val.Elem().Interface()) {
		return fmt.Sprintf("The final argument to this assertion must be a "+
			"pointer to a non-nil error value (you provided: '%v').", expected[0])
	}

	if !errors.As(err, expected[0]) {
		return fmt.Sprintf("Error '%v' does not match the expected error type '%T'",
			err, expected[0])
	}

	return ""
}

func isError(val interface{}) bool {
	err, ok := val.(error)

	return ok && err != nil
}
