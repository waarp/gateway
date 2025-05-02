package tasks

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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

		Convey("Given that a optional parameter is missing", func() {
			args := map[string]string{
				"path":  "cp",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
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
	transCtx := getExecTransCtx(t)

	Convey("Given an 'EXECOUTPUT' task", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_execoutput")
		root := testhelpers.TempDir(c, "task_execoutput")
		scriptFile := filepath.Join(root, execOutputScriptFile)

		exec := &execOutputTask{}

		Convey("Given that the task is valid", func() {
			args := map[string]string{
				"path":  scriptFile,
				"args":  scriptFile,
				"delay": "1000",
			}

			Convey("Given that the command succeeds", func() {
				err := os.WriteFile(scriptFile, []byte(scriptExecOK), 0o700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					err := exec.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the command sends a warning", func() {
				err := os.WriteFile(scriptFile, []byte(scriptExecWarn), 0o700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					err := exec.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should return a 'warning' error", func() {
						So(err, ShouldHaveSameTypeAs, &WarningError{})
					})
				})
			})

			Convey("Given that the command fails", func() {
				So(os.WriteFile(scriptFile, []byte(scriptExecOutputFail),
					0o700), ShouldBeNil)

				Convey("When running the task", func() {
					err := exec.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err, ShouldNotHaveSameTypeAs, &WarningError{})

						Convey("Then the transfer file should have changed", func() {
							So(transCtx.Transfer.LocalPath, ShouldEqual, execOutputNewFilename)
						})
					})
				})
			})

			Convey("Given that the command delay expires", func() {
				err := os.WriteFile(scriptFile, []byte(scriptExecInfinite), 0o700)
				So(err, ShouldBeNil)

				args["delay"] = "100"

				Convey("When running the task", func() {
					err := exec.Run(context.Background(), args, nil, logger, transCtx)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError, ErrCommandTimeout)
					})
				})
			})
		})
	})
}
