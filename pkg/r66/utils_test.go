package r66

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestCheckHash(t *testing.T) {
	path := filepath.Join(os.TempDir(), "test_check_hash_file")
	content := []byte("test CheckHash file content")
	sha256, _ := hex.DecodeString("cddfc994ff46f856395a6a387f722429bff47751cf0fd6924e80445e5c035672")

	Convey("Given a file", t, func() {
		So(ioutil.WriteFile(path, content, 0o600), ShouldBeNil)
		defer os.Remove(path)

		Convey("When calling the `checkHash` function with the correct hash", func() {
			err := checkHash(path, sha256)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})

		Convey("When calling the `checkHash` function with an incorrect hash", func() {
			err := checkHash(path, []byte("not a hash"))

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, errIncorrectHash)
			})
		})

		Convey("When calling the `checkHash` function with an invalid path", func() {
			err := checkHash("not a path", sha256)

			Convey("Then it should return an error", func() {
				So(os.IsNotExist(err), ShouldBeTrue)
			})
		})
	})
}
