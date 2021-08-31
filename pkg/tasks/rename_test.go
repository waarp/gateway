package tasks

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
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
				So(err.Error(), ShouldEqual, "cannot create a rename task without a `path` argument")
			})
		})
	})
}

func TestRenameTaskRun(t *testing.T) {
	Convey("Given a Runner for a sending Transfer", t, func(c C) {
		root := testhelpers.TempDir(c, "task_rename")
		task := &renameTask{}

		srcPath := filepath.Join(root, "rename.src")
		So(ioutil.WriteFile(srcPath, []byte("Hello World"), 0o700), ShouldBeNil)
		dstPath := filepath.Join(root, "rename.dst")
		So(ioutil.WriteFile(dstPath, []byte("Goodbye World"), 0o700), ShouldBeNil)

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
				_, err := task.Run(nil, args, nil, transCtx)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then transfer filepath should be modified", func() {
					So(transCtx.Transfer.LocalPath, ShouldEqual, utils.ToStandardPath(dstPath))
				})

				Convey("Then transfer source path should be modified", func() {
					So(transCtx.Transfer.RemotePath, ShouldEqual, "/remote/rename.dst")
				})
			})
		})

		Convey("Given an invalid new path", func() {
			args := map[string]string{"path": filepath.Join(root, "dummy")}

			Convey("When calling the `run` method", func() {
				_, err := task.Run(nil, args, nil, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, &errFileNotFound{"change transfer target file to",
						args["path"]})
				})
			})
		})
	})
}
