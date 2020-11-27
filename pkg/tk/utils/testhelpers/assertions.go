package testhelpers

import (
	"fmt"
	"strings"

	"github.com/smartystreets/assertions"
)

// ShouldBeOneOf receives at least 2 parameters. The first is a proposed member
// of the collection that is comprised of all the remaining parameters.
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
