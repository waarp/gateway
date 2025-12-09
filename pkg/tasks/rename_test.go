package tasks

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestRenameTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &renameTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &renameTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err, ShouldWrap, ErrRenameMissingPath)
			})
		})
	})
}

func TestRenameTaskRun(t *testing.T) {
	root := t.TempDir()

	Convey("Given a Runner for a sending Transfer", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_rename")
		task := &renameTask{}

		srcPath := fs.JoinPath(root, "rename.src")
		dstPath := fs.JoinPath(root, "rename.dst")

		So(fs.WriteFullFile(srcPath, []byte("Hello World")), ShouldBeNil)
		So(fs.WriteFullFile(dstPath, []byte("Goodbye World")), ShouldBeNil)

		transCtx := &model.TransferContext{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				LocalPath:  srcPath,
				RemotePath: "/remote/rename.src",
			},
		}

		Convey("Given a valid new path", func() {
			args := map[string]string{"path": dstPath}

			Convey("When calling the `run` method", func() {
				err := task.Run(t.Context(), args, nil, logger, transCtx, nil)
				So(err, ShouldBeNil)

				Convey("Then transfer filepath should be modified", func() {
					So(transCtx.Transfer.LocalPath, ShouldEqual, dstPath)
				})

				Convey("Then transfer source path should be modified", func() {
					So(transCtx.Transfer.RemotePath, ShouldEqual, "/remote/rename.dst")
				})
			})
		})

		Convey("Given an invalid new path", func() {
			args := map[string]string{"path": "wrong:/dummy.file"}

			Convey("When calling the `run` method", func() {
				err := task.Run(t.Context(), args, nil, logger, transCtx, nil)

				Convey("Then it should return an error", func() {
					So(err, ShouldWrap, fs.ErrUnknownFS)
				})
			})
		})
	})
}
