package tasks

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestExecMoveValidate(t *testing.T) {
	Convey("Given an 'EXECMOVE' task", t, func() {
		exec := &execMoveTask{}

		Convey("Given valid argument", func() {
			args := map[string]string{
				"path":  "cp",
				"args":  "file1 file2",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that a parameter is NOT the valid type", func() {
			args := map[string]string{
				"path":  "cp",
				"args":  "file1 file2",
				"delay": "true",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("Given that a parameter is missing", func() {
			args := map[string]string{
				"path":  "cp",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("Given that a parameter is empty", func() {
			args := map[string]string{
				"path":  "",
				"args":  "file1 file2",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}

func TestExecMoveRun(t *testing.T) {
	Convey("Given an 'EXECMOVE' task", t, func(c C) {
		root := testhelpers.TempDir(c, "task_execmove")
		scriptPath := filepath.Join(root, execMoveScriptFile)

		exec := &execMoveTask{}
		transCtx := &model.TransferContext{
			Rule:     &model.Rule{IsSend: false},
			Transfer: &model.Transfer{},
		}

		srcFile := filepath.Join(root, "test.src")
		dstFile := filepath.Join(root, "test.dst")
		So(ioutil.WriteFile(filepath.Join(srcFile), []byte("Hello world"), 0o600),
			ShouldBeNil)

		args := map[string]string{
			"path":  scriptPath,
			"args":  srcFile + " " + dstFile,
			"delay": "0",
		}

		Convey("Given that the command succeeds", func() {
			err := ioutil.WriteFile(scriptPath, []byte(scriptExecMove), 0700)
			So(err, ShouldBeNil)

			Convey("When running the task", func() {
				_, err := exec.Run(context.Background(), args, nil, transCtx)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the transfer filepath should have changed", func() {
						So(utils.ToOSPath(transCtx.Transfer.LocalPath),
							ShouldEqual, dstFile)
					})
				})
			})
		})

		Convey("Given that the command sends a warning", func() {
			err := ioutil.WriteFile(scriptPath, []byte(scriptExecWarn), 0700)
			So(err, ShouldBeNil)

			Convey("When running the task", func() {
				_, err := exec.Run(context.Background(), args, nil, transCtx)

				Convey("Then it should return a 'warning' error", func() {
					So(err, ShouldHaveSameTypeAs, &errWarning{})
				})
			})
		})

		Convey("Given that the command fails", func() {
			err := ioutil.WriteFile(scriptPath, []byte(scriptExecFail), 0700)
			So(err, ShouldBeNil)

			Convey("When running the task", func() {
				_, err := exec.Run(context.Background(), args, nil, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
					So(err, ShouldNotHaveSameTypeAs, &errWarning{})
				})
			})
		})

		Convey("Given that the command delay expires", func() {
			err := ioutil.WriteFile(scriptPath, []byte(scriptExecInfinite), 0700)
			So(err, ShouldBeNil)

			args["delay"] = "100"

			Convey("When running the task", func() {
				_, err := exec.Run(context.Background(), args, nil, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, errCommandTimeout)
				})
			})
		})
	})
}
