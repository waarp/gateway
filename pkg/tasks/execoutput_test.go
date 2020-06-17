package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExecOutputValidate(t *testing.T) {

	Convey("Given an 'EXECOUTPUT' task", t, func() {
		exec := &ExecOutputTask{}

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

	Convey("Given an 'EXECOUTPUT' task", t, func() {
		exec := &ExecOutputTask{}
		proc := &Processor{
			Transfer: &model.Transfer{},
			Rule:     &model.Rule{},
		}

		Reset(func() { _ = os.Remove(execOutputScriptFile) })

		Convey("Given that the task is valid", func() {
			args := map[string]string{
				"path":  execOutputScriptFile,
				"args":  "execmove.go",
				"delay": "1000",
			}

			Convey("Given that the command succeeds", func() {
				err := ioutil.WriteFile(execOutputScriptFile, []byte(scriptExecOK), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(args, proc)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})
				})
			})

			Convey("Given that the command sends a warning", func() {
				err := ioutil.WriteFile(execOutputScriptFile, []byte(scriptExecWarn), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(args, proc)

					Convey("Then it should return a 'warning' error", func() {
						So(err, ShouldBeError, errWarning)
					})
				})
			})

			Convey("Given that the command fails", func() {
				err := ioutil.WriteFile(execOutputScriptFile, []byte(scriptExecOutputFail), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					msg, err := exec.Run(args, proc)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err, ShouldNotEqual, errWarning)

						Convey("Then the message should contain the script "+
							"output", func() {
							So(msg, ShouldEqual, "This is a message"+lineSeparator+
								"NEWFILENAME:new_name.file"+lineSeparator)
						})

						Convey("Then the transfer file should have changed", func() {
							So(proc.Transfer.DestFile, ShouldEqual, "new_name.file")
						})
					})
				})
			})

			Convey("Given that the command delay expires", func() {
				err := ioutil.WriteFile(execOutputScriptFile, []byte(scriptExecInfinite), 0700)
				So(err, ShouldBeNil)

				args["delay"] = "100"

				Convey("When running the task", func() {
					msg, err := exec.Run(args, proc)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)

						Convey("Then the message should say that the "+
							"delay has expired", func() {
							So(msg, ShouldEqual, "max exec delay expired")
						})
					})
				})
			})
		})
	})
}
