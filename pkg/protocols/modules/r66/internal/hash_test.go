package internal

import (
	"context"
	"encoding/hex"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCheckHash(t *testing.T) {
	content := []byte("test CheckHash file content")
	expHash, _ := hex.DecodeString("cddfc994ff46f856395a6a387f722429bff47751cf0fd6924e80445e5c035672")

	Convey("Given a file", t, func(c C) {
		testFS := fstest.InitMemFS(c)
		path := mkPath("memory:/test_check_hash_file")
		logger := testhelpers.TestLogger(c, "test_check_hash")

		So(fs.WriteFullFile(testFS, &path, content), ShouldBeNil)

		Convey("When calling the `checkHash` function with the correct hash", func() {
			hash, err := MakeHash(context.Background(), "SHA-256", testFS, logger, &path)
			So(err, ShouldBeNil)

			Convey("Then it should return the expected hash", func() {
				So(hash, ShouldResemble, expHash)
			})
		})

		Convey("When calling the `checkHash` function with an invalid path", func() {
			filepath := mkPath("memory:/not_a_path")
			_, err := MakeHash(context.Background(), "SHA-256", testFS, logger, &filepath)

			Convey("Then it should return an error", func() {
				So(err, ShouldBeError, pipeline.NewError(types.TeInternal,
					"failed to open file"))
			})
		})
	})
}
