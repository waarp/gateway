package tasks

import (
	"context"
	"path"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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
				So(err, ShouldWrap, ErrMoveMissingPath)
			})
		})
	})
}

func TestMoveTaskRun(t *testing.T) {
	root := t.TempDir()

	Convey("Given a Runner for a MOVE task", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_move")
		task := &moveTask{}

		srcPath := fs.JoinPath(root, "src_dir", "move.file")
		filename := path.Base(srcPath)

		So(fs.MkdirAll(path.Dir(srcPath)), ShouldBeNil)
		So(fs.WriteFullFile(srcPath, []byte("Hello World")), ShouldBeNil)

		transCtx := &model.TransferContext{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				LocalPath:  srcPath,
				RemotePath: "/remote/move.file",
			},
		}

		args := map[string]string{}

		Convey("Given that the path is valid", func() {
			dirPath := filepath.Join(root, "dst_dir")
			args["path"] = dirPath

			Convey("Given that the file exists", func() {
				Convey("When calling the `Run` method", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)
					So(err, ShouldBeNil)

					Convey("Then the destination file should exist", func() {
						_, err := fs.Stat(path.Join(dirPath, filename))
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer local filepath should be modified", func() {
						So(transCtx.Transfer.LocalPath, ShouldEqual,
							path.Join(args["path"], filename))
					})

					Convey("Then the transfer remote path should NOT be modified", func() {
						So(transCtx.Transfer.RemotePath, ShouldEqual, "/remote/move.file")
					})
				})
			})

			Convey("Given that the file does NOT exist", func() {
				So(fs.RemoveAll(srcPath), ShouldBeNil)

				Convey("When calling the 'Run' method", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldWrap, fs.ErrNotExist)
					})
				})
			})
		})
	})
}
