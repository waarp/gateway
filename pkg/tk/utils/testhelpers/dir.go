// Package testhelpers provides utilities to help writing mor concise and
// readable tests.
package testhelpers

import (
	"io/ioutil"
	"os"

	"github.com/smartystreets/goconvey/convey"
)

// TempDir creates a new temporary directoryand returns its path.
//
// The directory exits for the duration of the test only.
//
// It integrates fully with Convey :
// - Reset is used to remove the diractory at the end of the test
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
