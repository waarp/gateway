package tasks

import (
	"context"
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestCopyTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &copyTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a invalid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &copyTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldContainSubstring, "cannot create a copy task without a `path` argument")
			})
		})
	})
}

func TestCopyTaskRun(t *testing.T) {
	Convey("Given a Runner for the 'COPY' task", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_copy")
		testFS := fstest.InitMemFS(c)
		task := &copyTask{}

		srcFile := makeURL("/src_dir/test.file")
		filename := path.Base(srcFile.Path)

		transCtx := &model.TransferContext{
			Transfer: &model.Transfer{LocalPath: srcFile},
			FS:       testFS,
		}

		So(fs.MkdirAll(testFS, srcFile.Dir()), ShouldBeNil)
		So(fs.WriteFullFile(testFS, &srcFile, []byte("Hello World")), ShouldBeNil)

		args := map[string]string{}

		Convey("Given that the file does NOT exist", func() {
			args["path"] = "/dst_dir"

			So(fs.Remove(testFS, &srcFile), ShouldBeNil)

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)

					Convey("Then error should say `no such file`", func() {
						So(fs.IsNotExist(err), ShouldBeTrue)
					})
				})
			})
		})

		Convey("Given that the file is copied on itself", func() {
			dstPath := srcFile.Dir()
			args["path"] = dstPath.String()

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then the file should NOT be empty", func() {
					info, err := fs.Stat(testFS, &srcFile)
					So(err, ShouldBeNil)
					So(info.Size(), ShouldNotEqual, 0)
				})
			})
		})

		Convey("Given the destination is a non-existing subdir", func() {
			dstDir := makeURL("/dst_dir")
			args["path"] = dstDir.String()

			Convey("Given the target can be created", func() {
				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)
					So(err, ShouldBeNil)

					Convey("Then the target file exists", func() {
						_, err := fs.Stat(testFS, dstDir.JoinPath(filename))
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given the target CANNOT be created", func() {
				So(fs.WriteFullFile(testFS, &dstDir, []byte("hello")), ShouldBeNil)

				Convey("When the task is run", func() {
					err := task.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)

						Convey("Then the target file does not exist", func() {
							dstFile := dstDir.JoinPath(filename)
							_, err := fs.Stat(testFS, dstFile)
							So(fs.IsNotExist(err), ShouldBeTrue)
						})
					})
				})
			})
		})
	})
}
