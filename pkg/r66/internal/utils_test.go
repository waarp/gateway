package internal

import (
	"context"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	. "github.com/smartystreets/goconvey/convey"
)

func init() {
	_ = log.InitBackend("DEBUG", "stdout", "")
}

func TestCheckHash(t *testing.T) {
	logger := log.NewLogger("test_check_hash")

	path := filepath.Join(os.TempDir(), "test_check_hash_file")
	content := []byte("test CheckHash file content")
	expHash, _ := hex.DecodeString("cddfc994ff46f856395a6a387f722429bff47751cf0fd6924e80445e5c035672")

	Convey("Given a file", t, func() {
		So(ioutil.WriteFile(path, content, 0o600), ShouldBeNil)
		defer os.Remove(path)

		Convey("When calling the `checkHash` function with the correct hash", func() {
			hash, err := MakeHash(context.Background(), logger, path)
			So(err, ShouldBeNil)

			Convey("Then it should return the expected hash", func() {
				So(hash, ShouldResemble, expHash)
			})
		})

		Convey("When calling the `checkHash` function with an invalid path", func() {
			path := "not a path"
			_, err := MakeHash(context.Background(), logger, path)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, types.NewTransferError(types.TeInternal, "failed to open file"))
			})
		})
	})
}
