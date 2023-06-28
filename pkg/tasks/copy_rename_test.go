package tasks

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/fs/fstest"
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
		fstest.InitMemFS(c)

		logger := testhelpers.TestLogger(c, "task_copyrename")
		task := &copyRenameTask{}
		srcPath := makeURL("mem:/test.src")

		transCtx := &model.TransferContext{
			Transfer: &model.Transfer{LocalPath: srcPath},
		}

		So(fs.WriteFullFile(&srcPath, []byte("Hello World")), ShouldBeNil)

		args := map[string]string{}

		Convey("Given a valid new path", func() {
			dstPath := makeURL("mem:/test.copy")
			args["path"] = dstPath.String()

			Convey("When the task is run", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then the destination file should exist", func() {
					_, err := fs.Stat(&dstPath)
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that the file does NOT exist", func() {
			So(fs.Remove(&srcPath), ShouldBeNil)

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldNotBeNil)

				Convey("Then error should say `no such file`", func() {
					So(err, ShouldWrap, fs.ErrNotExist)
				})
			})
		})

		Convey("Given that the file is copied on itself", func() {
			args["path"] = srcPath.String()

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then the file should NOT be empty", func() {
					info, err := fs.Stat(&srcPath)
					So(err, ShouldBeNil)
					So(info.Size(), ShouldNotEqual, 0)
				})
			})
		})

		Convey("Given the target is a non-existing subdir", func() {
			dstPath := makeURL("mem:/sub_dir/test.src.copy")
			args["path"] = dstPath.String()

			Convey("Given the target can be created", func() {
				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)
					So(err, ShouldBeNil)

					Convey("Then the target file exists", func() {
						_, err := fs.Stat(&dstPath)
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given the target CANNOT be created", func() {
				dummyPath := dstPath.Dir()
				So(fs.WriteFullFile(dummyPath, []byte("hello")), ShouldBeNil)

				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)
					So(err, ShouldBeError)

					Convey("Then the target file does not exist", func() {
						_, err := fs.Stat(&dstPath)
						So(fs.IsNotExist(err), ShouldBeTrue)
					})
				})
			})
		})
	})
}
