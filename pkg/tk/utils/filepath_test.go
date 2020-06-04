package utils

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetPath(t *testing.T) {
	Convey("Given a non empty path", t, func() {
		path := "ressource"

		Convey("Given a list of non empty root paths", func() {
			roots := []string{"in", "out/test"}

			Convey("When calling the 'GetPath' function", func() {
				res := GetPath(path, roots...)

				Convey("Then it should returns 'in/ressource'", func() {
					So(res, ShouldEqual, "in/ressource")
				})
			})
		})

		Convey("Given a list of root paths with empty first root", func() {
			roots := []string{"", "out/test"}

			Convey("When calling the 'GetPath' function", func() {
				res := GetPath(path, roots...)

				Convey("Then it should returns 'in/ressource'", func() {
					So(res, ShouldEqual, "out/test/ressource")
				})
			})
		})

		Convey("Given a list of empty root paths", func() {
			roots := []string{"", ""}

			Convey("When calling the 'GetPath' function", func() {
				res := GetPath(path, roots...)

				Convey("Then it should returns 'in/ressource'", func() {
					So(res, ShouldEqual, "ressource")
				})
			})
		})

		Convey("When calling the 'GetPath' function", func() {
			res := GetPath(path)

			Convey("Then it should returns ''", func() {
				So(res, ShouldEqual, "ressource")
			})
		})
	})

	Convey("Given an empty path", t, func() {
		path := ""

		Convey("Given a list of non empty root paths", func() {
			roots := []string{"in", "out/test"}

			Convey("When calling the 'GetPath' function", func() {
				res := GetPath(path, roots...)

				Convey("Then it should returns 'in/ressource'", func() {
					So(res, ShouldEqual, "in")
				})
			})
		})

		Convey("Given a list of root paths with empty first root", func() {
			roots := []string{"", "out/test"}

			Convey("When calling the 'GetPath' function", func() {
				res := GetPath(path, roots...)

				Convey("Then it should returns 'in/ressource'", func() {
					So(res, ShouldEqual, "out/test")
				})
			})
		})

		Convey("Given a list of empty root paths", func() {
			roots := []string{"", ""}

			Convey("When calling the 'GetPath' function", func() {
				res := GetPath(path, roots...)

				Convey("Then it should returns 'in/ressource'", func() {
					So(res, ShouldEqual, "")
				})
			})
		})

		Convey("When calling the 'GetPath' function", func() {
			res := GetPath(path)

			Convey("Then it should returns ''", func() {
				So(res, ShouldEqual, "")
			})
		})
	})

	Convey("Given a path starting with '/'", t, func() {
		path := "/ressource"

		Convey("Given a list of non empty root paths", func() {
			roots := []string{"in", "out/test"}

			Convey("When calling the 'GetPath' function", func() {
				res := GetPath(path, roots...)

				Convey("Then it should returns '/ressource'", func() {
					So(res, ShouldEqual, "/ressource")
				})
			})
		})
	})
}
