package tasks

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
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
				So(err.Error(), ShouldEqual, "cannot create a move task without a `path` argument")
			})
		})
	})
}

func TestMoveTaskRun(t *testing.T) {
	Convey("Given a Runner for a MOVE task", t, func(c C) {
		root := testhelpers.TempDir(c, "task_move")
		task := &moveTask{}
		srcPath := filepath.Join(root, "move.src")
		So(ioutil.WriteFile(srcPath, []byte("Hello World"), 0o700), ShouldBeNil)

		info := &model.TransferContext{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				TrueFilepath: srcPath,
				SourceFile:   "move.src",
			},
		}

		args := map[string]string{}

		Convey("Given that the path is valid", func() {
			args["path"] = path.Join(root, "dest")

			Convey("Given that the file exists", func() {

				Convey("When calling the `Run` method", func() {
					_, err := task.Run(args, nil, info, nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat(args["path"])
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer true file path should be modified", func() {
						So(info.Transfer.TrueFilepath, ShouldEqual, utils.NormalizePath(
							path.Join(args["path"], "move.src")))
					})

					Convey("Then the transfer source path should NOT be modified", func() {
						So(info.Transfer.SourceFile, ShouldEqual, "move.src")
					})
				})
			})

			Convey("Given that the file does NOT exist", func() {
				So(os.Remove(srcPath), ShouldBeNil)

				Convey("When calling the 'Run' method", func() {
					_, err := task.Run(args, nil, info, nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldBeError, &errFileNotFound{"open source file", srcPath})
					})
				})
			})
		})
	})
}
