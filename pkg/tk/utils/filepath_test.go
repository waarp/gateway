package utils

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetPath(t *testing.T) {
	testCases := []struct {
		path     string
		roots    Elems
		expected string
	}{
		{"ressource", Elems{{"/in", false}, {"/out/test", false}}, "/in/ressource"},
		{"ressource", Elems{{"in", false}, {"out/test", false}}, "/out/test/in/ressource"},
		{"ressource", Elems{{"work", false}, {"out/test", true}}, "/work/ressource"},
		{"ressource", Elems{{"", false}, {"out/test", false}}, "/out/test/ressource"},
		{"ressource", Elems{{"", false}, {"", false}}, "/ressource"},
		{"ressource", Elems{}, "/ressource"},
		{"/ressource", Elems{{"in", false}, {"out/test", false}}, "/ressource"},
		{"/ressource", Elems{{"", false}, {"out/test", false}}, "/ressource"},
		{"/ressource", Elems{{"", false}, {"", false}}, "/ressource"},
		{"", Elems{{"/in", false}, {"out/test", false}}, "/in"},
		{"", Elems{{"", false}, {"out/test", false}}, "/out/test"},
		{"", Elems{{"", false}, {"", false}}, "/"},
		{"", Elems{}, "/"},
	}

	for _, tc := range testCases {
		Convey(fmt.Sprintf("Given a ressource path: %s", tc.path), t, func() {
			Convey(fmt.Sprintf("When calling the 'GetPath' function with roots: [%v]", tc.roots), func() {
				res := GetPath(tc.path, tc.roots)
				Convey(fmt.Sprintf("Then it should returns '%s'", tc.expected), func() {
					So(res, ShouldEqual, tc.expected)
				})
			})
		})
	}
}
