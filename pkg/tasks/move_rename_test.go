package tasks

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestMoveRenameTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &moveRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a invalid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &moveRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldContainSubstring, "cannot create a move_rename task without a `path` argument")
			})
		})
	})
}

func TestMoveRenameTaskRun(t *testing.T) {
	Convey("Given a Runner for a sending transfer", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_moverename")
		testFS := fstest.InitMemFS(c)
		task := &moveRenameTask{}
		srcPath := makeURL("/src_dir/move_rename.src")

		So(fs.MkdirAll(testFS, srcPath.Dir()), ShouldBeNil)
		So(fs.WriteFullFile(testFS, &srcPath, []byte("Hello World")), ShouldBeNil)

		transCtx := &model.TransferContext{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				LocalPath:  srcPath,
				RemotePath: "/remote/move_rename.src",
			},
			FS: testFS,
		}

		args := map[string]string{}

		Convey("Given that the path is valid", func() {
			dstPath := makeURL("/dst_dir/move_rename.dst")
			args["path"] = dstPath.String()

			Convey("Given that the file exists", func() {
				Convey("When calling the `Run` method", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := fs.Stat(testFS, &dstPath)
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer local filepath should be modified", func() {
						So(transCtx.Transfer.LocalPath.String(), ShouldEqual, args["path"])
					})

					Convey("Then the transfer source path should NOT be modified", func() {
						So(transCtx.Transfer.RemotePath, ShouldEqual, "/remote/move_rename.dst")
					})
				})
			})

			Convey("Given that the file does NOT exist", func() {
				So(fs.Remove(testFS, &srcPath), ShouldBeNil)

				Convey("When calling the 'Run' method", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then error should say `no such file`", func() {
						So(fs.IsNotExist(err), ShouldBeTrue)
					})
				})
			})
		})
	})
}
