package tasks

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
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
				So(err.Error(), ShouldEqual, "cannot create a copy_rename task without a `path` argument")
			})
		})
	})
}

func TestCopyRenameTaskRun(t *testing.T) {
	Convey("Given a Runner for the 'COPYRENAME' task", t, func(c C) {
		root := testhelpers.TempDir(c, "task_copyrename")
		task := &copyRenameTask{}
		srcFile := filepath.Join(root, "test.src")

		info := &model.TransferContext{Transfer: &model.Transfer{
			LocalPath: utils.ToStandardPath(srcFile),
		}}

		So(ioutil.WriteFile(srcFile, []byte("Hello World"), 0o700), ShouldBeNil)
		args := map[string]string{}

		Convey("Given a valid new path", func() {
			args["path"] = filepath.Join(root, "test.src.copy")

			Convey("When the task is run", func() {

				_, err := task.Run(args, nil, info, nil)

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
				_, err := task.Run(args, nil, info, nil)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})

				Convey("Then error should say `no such file`", func() {
					So(err, ShouldBeError, &errFileNotFound{"open source file", srcFile})
				})
			})
		})

		Convey("Given the target is a non-existing subdir", func() {
			args["path"] = filepath.Join(root, "subdir", "test.src.copy")

			Convey("Given the target can be created", func() {

				Convey("When the task is run", func() {
					_, err := task.Run(args, nil, info, nil)

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
				So(ioutil.WriteFile(filepath.Join(root, "subdir"),
					[]byte("hello"), 0o600), ShouldBeNil)

				Convey("When the task is run", func() {
					_, err := task.Run(args, nil, info, nil)

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
