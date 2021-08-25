// Package testhelpers provides utilities to help writing mor concise and
// readable tests.
package testhelpers

import (
	"io/ioutil"
	"os"

	"github.com/smartystreets/goconvey/convey"
)

// TempDir creates a new temporary directory and returns its path.
//
// The directory exits for the duration of the test only.
//
// It integrates fully with Convey :
// - Reset is used to remove the directory at the end of the test
// - any error will mark the test as failed
// - It will panic if it is not called from within a convey context.
func TempDir(c convey.C, name string) string {
	path, err := ioutil.TempDir("", "gateway-test."+name+".*")
	c.So(err, convey.ShouldBeNil)
	c.Reset(func() {
		c.So(os.RemoveAll(path), convey.ShouldBeNil)
	})

	return path
}

// TempFile creates a new temporary file and returns its path. The file will have
// a name in accordance with the given pattern. See the ioutil.TempFile doc for
// more information on the pattern format.
//
// The file exits for the duration of the test only.
//
// It integrates fully with Convey :
// - Reset is used to remove the file at the end of the test
// - any error will mark the test as failed
// - It will panic if it is not called from within a convey context.
func TempFile(c convey.C, pattern string) string {
	file, err := ioutil.TempFile("", pattern)
	c.So(err, convey.ShouldBeNil)
	c.Reset(func() {
		c.So(os.RemoveAll(file.Name()), convey.ShouldBeNil)
	})
	c.So(file.Close(), convey.ShouldBeNil)
	return file.Name()
}
