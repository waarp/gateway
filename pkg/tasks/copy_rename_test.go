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
				So(err.Error(), ShouldContainSubstring, "cannot create a copy_rename task without a `path` argument")
			})
		})
	})
}

func TestCopyRenameTaskRun(t *testing.T) {
	Convey("Given a Runner for the 'COPYRENAME' task", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_copyrename")
		root := testhelpers.TempDir(c, "task_copyrename")
		task := &copyRenameTask{}
		srcFile := filepath.Join(root, "test.src")

		transCtx := &model.TransferContext{Transfer: &model.Transfer{
			LocalPath: utils.ToOSPath(srcFile),
		}}

		So(os.WriteFile(srcFile, []byte("Hello World"), 0o700), ShouldBeNil)
		args := map[string]string{}

		Convey("Given a valid new path", func() {
			args["path"] = filepath.Join(root, "test.src.copy")

			Convey("When the task is run", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should NOT return error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the destination file should exist", func() {
					_, err := os.Stat(args["path"])
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that the file does NOT exist", func() {
			So(os.Remove(srcFile), ShouldBeNil)

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})

				Convey("Then error should say `no such file`", func() {
					So(err, ShouldBeError, &fileNotFoundError{"open source file", srcFile})
				})
			})
		})

		Convey("Given that the file is copied on itself", func() {
			args["path"] = srcFile

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the file should NOT be empty", func() {
						info, err := os.Stat(srcFile)
						So(err, ShouldBeNil)
						So(info.Size(), ShouldNotEqual, 0)
					})
				})
			})
		})

		Convey("Given the target is a non-existing subdir", func() {
			args["path"] = filepath.Join(root, "subdir", "test.src.copy")

			Convey("Given the target can be created", func() {
				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should return no error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the target file exists", func() {
						_, err := os.Stat(args["path"])
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given the target CANNOT be created", func() {
				So(os.WriteFile(filepath.Join(root, "subdir"),
					[]byte("hello"), 0o600), ShouldBeNil)

				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
					})

					Convey("Then the target file does not exist", func() {
						_, err := os.Stat(args["path"])
						So(err, ShouldBeError)
					})
				})
			})
		})
	})
}
