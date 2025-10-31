package tasks

import (
	"path"
	"runtime"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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
				So(err, ShouldWrap, ErrCopyMissingPath)
			})
		})
	})
}

func TestCopyTaskRun(t *testing.T) {
	Convey("Given a Runner for the 'COPY' task", t, func(c C) {
		root := t.TempDir()
		logger := testhelpers.TestLogger(c, "task_copy")
		task := &copyTask{}

		srcFile := fs.JoinPath(root, "src_dir", "test.file")
		filename := path.Base(srcFile)

		transCtx := &model.TransferContext{
			Transfer: &model.Transfer{LocalPath: srcFile},
		}

		So(fs.MkdirAll(path.Dir(srcFile)), ShouldBeNil)
		So(fs.WriteFullFile(srcFile, []byte("Hello World")), ShouldBeNil)

		args := map[string]string{}

		Convey("Given that the file does NOT exist", func() {
			if runtime.GOOS == "windows" {
				args["path"] = "C:/dst_dir"
			} else {
				args["path"] = "/dst_dir"
			}

			So(fs.RemoveAll(srcFile), ShouldBeNil)

			Convey("When calling the run method", func() {
				err := task.Run(t.Context(), args, nil, logger, transCtx, nil)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldWrap, fs.ErrNotExist)
					})
				})
			})
		})

		Convey("Given that the file is copied on itself", func() {
			dstPath := path.Dir(srcFile)
			args["path"] = dstPath

			Convey("When calling the run method", func() {
				err := task.Run(t.Context(), args, nil, logger, transCtx, nil)
				So(err, ShouldBeNil)

				Convey("Then the file should NOT be empty", func() {
					info, err := fs.Stat(srcFile)
					So(err, ShouldBeNil)
					So(info.Size(), ShouldNotEqual, 0)
				})
			})
		})

		Convey("Given the destination is a non-existing subdir", func() {
			dstDir := fs.JoinPath(root, "dst_dir")
			args["path"] = dstDir

			Convey("Given the target can be created", func() {
				Convey("When the task is run", func() {
					err := task.Run(t.Context(), args, nil, logger, transCtx, nil)
					So(err, ShouldBeNil)

					Convey("Then the target file exists", func() {
						_, err := fs.Stat(path.Join(dstDir, filename))
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given the target CANNOT be created", func() {
				So(fs.WriteFullFile(dstDir, []byte("hello")), ShouldBeNil)

				Convey("When the task is run", func() {
					err := task.Run(t.Context(), args, nil, logger, transCtx, nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)

						Convey("Then the target file does not exist", func() {
							dstFile := path.Join(dstDir, filename)
							_, err := fs.Stat(dstFile)
							So(err, ShouldWrap, fs.ErrNotDir)
						})
					})
				})
			})
		})
	})
}
