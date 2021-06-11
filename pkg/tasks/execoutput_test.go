package tasks

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExecOutputValidate(t *testing.T) {

	Convey("Given an 'EXECOUTPUT' task", t, func() {
		exec := &execOutputTask{}

		Convey("Given valid arguments", func() {
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

func TestExecOutputRun(t *testing.T) {

	Convey("Given an 'EXECOUTPUT' task", t, func(c C) {
		root := testhelpers.TempDir(c, "task_execoutput")
		scriptFile := filepath.Join(root, execOutputScriptFile)

		exec := &execOutputTask{}
		transCtx := &model.TransferContext{
			Transfer: &model.Transfer{},
			Rule:     &model.Rule{},
		}

		Convey("Given that the task is valid", func() {
			args := map[string]string{
				"path":  scriptFile,
				"args":  scriptFile,
				"delay": "1000",
			}

			Convey("Given that the command succeeds", func() {
				err := ioutil.WriteFile(scriptFile, []byte(scriptExecOK), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(context.Background(), args, nil, transCtx)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the command sends a warning", func() {
				err := ioutil.WriteFile(scriptFile, []byte(scriptExecWarn), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(context.Background(), args, nil, transCtx)

					Convey("Then it should return a 'warning' error", func() {
						So(err, ShouldHaveSameTypeAs, &errWarning{})
					})
				})
			})

			Convey("Given that the command fails", func() {
				So(ioutil.WriteFile(scriptFile, []byte(scriptExecOutputFail),
					0700), ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(context.Background(), args, nil, transCtx)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err, ShouldNotHaveSameTypeAs, &errWarning{})

						Convey("Then the transfer file should have changed", func() {
							So(transCtx.Transfer.LocalPath, ShouldEqual, "new_name.file")
						})
					})
				})
			})

			Convey("Given that the command delay expires", func() {
				err := ioutil.WriteFile(scriptFile, []byte(scriptExecInfinite), 0700)
				So(err, ShouldBeNil)

				args["delay"] = "100"

				Convey("When running the task", func() {
					_, err := exec.Run(context.Background(), args, nil, transCtx)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, "max execution delay expired")
					})
				})
			})
		})
	})
}
