package utils

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetPath(t *testing.T) {

	testCases := []struct {
		path     string
		roots    []string
		expected string
	}{
		{"ressource", []string{"in", "out/test"}, "in/ressource"},
		{"ressource", []string{"", "out/test"}, "out/test/ressource"},
		{"ressource", []string{"", ""}, "ressource"},
		{"ressource", []string{}, "ressource"},
		{"/ressource", []string{"in", "out/test"}, "/ressource"},
		{"/ressource", []string{"", "out/test"}, "/ressource"},
		{"/ressource", []string{"", ""}, "/ressource"},
		{"", []string{"in", "out/test"}, "in"},
		{"", []string{"", "out/test"}, "out/test"},
		{"", []string{"", ""}, ""},
		{"", []string{}, ""},
	}

	for _, tc := range testCases {
		Convey(fmt.Sprintf("Given a ressource path: %s", tc.path), t, func() {
			Convey(fmt.Sprintf("When calling the 'GetPath' function with roots: [%s]", strings.Join(tc.roots, ", ")), func() {
				res := GetPath(tc.path, tc.roots...)
				Convey(fmt.Sprintf("Then it should returns '%s'", tc.expected), func() {
					So(res, ShouldEqual, tc.expected)
				})
			})
		})
	}
}
