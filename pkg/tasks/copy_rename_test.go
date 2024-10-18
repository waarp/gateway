package tasks

import (
	"context"
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCopyRenameTaskValidate(t *testing.T) {
	Convey("Given a valid argument", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &copyRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given an invalid argument", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &copyRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err, ShouldWrap, ErrCopyRenameMissingPath)
			})
		})
	})
}

func TestCopyRenameTaskRun(t *testing.T) {
	Convey("Given a Runner for the 'COPYRENAME' task", t, func(c C) {
		root := t.TempDir()
		logger := testhelpers.TestLogger(c, "task_copyrename")
		task := &copyRenameTask{}
		srcPath := fs.JoinPath(root, "test.src")

		transCtx := &model.TransferContext{
			Transfer: &model.Transfer{LocalPath: srcPath},
		}

		So(fs.WriteFullFile(srcPath, []byte("Hello World")), ShouldBeNil)

		dstPath := fs.JoinPath(root, "test.copy")
		args := map[string]string{"path": dstPath}

		Convey("Given a valid new path", func() {
			Convey("When the task is run", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then the destination file should exist", func() {
					_, err := fs.Stat(dstPath)
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that the file does NOT exist", func() {
			So(fs.RemoveAll(srcPath), ShouldBeNil)

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldNotBeNil)

				Convey("Then error should say `no such file`", func() {
					So(err, ShouldWrap, fs.ErrNotExist)
				})
			})
		})

		Convey("Given that the file is copied on itself", func() {
			args["path"] = srcPath

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then the file should NOT be empty", func() {
					info, err := fs.Stat(srcPath)
					So(err, ShouldBeNil)
					So(info.Size(), ShouldNotEqual, 0)
				})
			})
		})

		Convey("Given the target is a non-existing subdir", func() {
			newDstPath := fs.JoinPath(root, "sub_dir", "test.src.copy")
			args["path"] = newDstPath

			Convey("Given the target can be created", func() {
				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)
					So(err, ShouldBeNil)

					Convey("Then the target file exists", func() {
						_, err := fs.Stat(newDstPath)
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given the target CANNOT be created", func() {
				dummyPath := path.Dir(newDstPath)
				So(fs.WriteFullFile(dummyPath, []byte("hello")), ShouldBeNil)

				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)
					So(err, ShouldBeError)

					Convey("Then the target file does not exist", func() {
						_, err := fs.Stat(newDstPath)
						So(err, ShouldWrap, fs.ErrNotExist)
					})
				})
			})
		})
	})
}
