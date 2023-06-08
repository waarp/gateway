package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestMoveTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &moveTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a invalid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &moveTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldContainSubstring, "cannot create a move task without a `path` argument")
			})
		})
	})
}

func TestMoveTaskRun(t *testing.T) {
	Convey("Given a Runner for a MOVE task", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_move")
		root := testhelpers.TempDir(c, "task_move")
		task := &moveTask{}
		srcPath := filepath.Join(root, "move.src")
		So(os.WriteFile(srcPath, []byte("Hello World"), 0o700), ShouldBeNil)

		transCtx := &model.TransferContext{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				LocalPath:  srcPath,
				RemotePath: "/remote/move.src",
			},
		}

		args := map[string]string{}

		Convey("Given that the path is valid", func() {
			args["path"] = filepath.Join(root, "dest")

			Convey("Given that the file exists", func() {
				Convey("When calling the `Run` method", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat(args["path"])
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer local filepath should be modified", func() {
						So(transCtx.Transfer.LocalPath, ShouldEqual, utils.ToOSPath(
							args["path"]+"/move.src"))
					})

					Convey("Then the transfer remote path should NOT be modified", func() {
						So(transCtx.Transfer.RemotePath, ShouldEqual, "/remote/move.src")
					})
				})
			})

			Convey("Given that the file does NOT exist", func() {
				So(os.Remove(srcPath), ShouldBeNil)

				Convey("When calling the 'Run' method", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldBeError, &fileNotFoundError{"open source file", srcPath})
					})
				})
			})
		})
	})
}
