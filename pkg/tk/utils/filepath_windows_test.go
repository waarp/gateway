//+build windows

package utils

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetPath(t *testing.T) {
	testCases := []struct {
		path     string
		roots    []Elem
		expected string
	}{
		{`ressource`, []Elem{Branch(`C:\in`), Branch(`C:\out\test`)}, `C:\in\ressource`},
		{`ressource`, []Elem{Branch(`in`), Branch(`out\test`)}, `out\test\in\ressource`},
		{`ressource`, []Elem{Branch(`work`), Leaf(`out\test`)}, `work\ressource`},
		{`ressource`, []Elem{Branch(``), Branch(`out\test`)}, `out\test\ressource`},
		{`ressource`, []Elem{Branch(``), Branch(``)}, `ressource`},
		{`ressource`, []Elem{}, `ressource`},
		{`C:\ressource`, []Elem{Branch(`in`), Branch(`out\test`)}, `C:\ressource`},
		{`C:\ressource`, []Elem{Branch(``), Branch(`out\test`)}, `C:\ressource`},
		{`C:\ressource`, []Elem{Branch(``), Branch(``)}, `C:\ressource`},
		{``, []Elem{Branch(`C:\in`), Branch(`out\test`)}, `C:\in`},
		{``, []Elem{Branch(``), Branch(`out\test`)}, `out\test`},
		{``, []Elem{Branch(``), Branch(``)}, ``},
		{``, []Elem{}, ``},
	}

	for _, tc := range testCases {
		Convey(fmt.Sprintf("Given a ressource path: %s", tc.path), t, func() {
			Convey(fmt.Sprintf("When calling the 'GetPath' function with roots: [%v]", tc.roots), func() {
				res := GetPath(tc.path, tc.roots...)
				Convey(fmt.Sprintf("Then it should returns '%s'", tc.expected), func() {
					So(res, ShouldEqual, tc.expected)
				})
			})
		})
	}
}
