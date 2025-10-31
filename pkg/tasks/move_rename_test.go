package tasks

import (
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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
				So(err, ShouldWrap, ErrMoveRenameMissingPath)
			})
		})
	})
}

func TestMoveRenameTaskRun(t *testing.T) {
	root := t.TempDir()
	conf.GlobalConfig.Paths.DirPerms = 0o700
	conf.GlobalConfig.Paths.FilePerms = 0o600

	Convey("Given a Runner for a sending transfer", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_moverename")
		task := &moveRenameTask{}
		srcPath := fs.JoinPath(root, "src_dir", "move_rename.src")

		So(fs.MkdirAll(path.Dir(srcPath)), ShouldBeNil)
		So(fs.WriteFullFile(srcPath, []byte("Hello World")), ShouldBeNil)

		transCtx := &model.TransferContext{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				LocalPath:  srcPath,
				RemotePath: "/remote/move_rename.src",
			},
		}

		args := map[string]string{}

		Convey("Given that the path is valid", func() {
			dstPath := fs.JoinPath(root, "dst_dir", "move_rename.dst")
			args["path"] = dstPath

			Convey("Given that the file exists", func() {
				Convey("When calling the `Run` method", func() {
					err := task.Run(t.Context(), args, nil, logger, transCtx, nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := fs.Stat(dstPath)
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer local filepath should be modified", func() {
						So(transCtx.Transfer.LocalPath, ShouldEqual, args["path"])
					})

					Convey("Then the transfer source path should NOT be modified", func() {
						So(transCtx.Transfer.RemotePath, ShouldEqual, "/remote/move_rename.dst")
					})
				})
			})

			Convey("Given that the file does NOT exist", func() {
				So(fs.RemoveAll(srcPath), ShouldBeNil)

				Convey("When calling the 'Run' method", func() {
					err := task.Run(t.Context(), args, nil, logger, transCtx, nil)

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldWrap, fs.ErrNotExist)
					})
				})
			})
		})
	})
}
