package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestExecMoveValidate(t *testing.T) {

	Convey("Given an 'EXECMOVE' task", t, func() {
		exec := &ExecMoveTask{}

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
	script := "execmove_test_script.sh"
	_ = os.Remove(script)

	Convey("Given an 'EXECMOVE' task", t, func() {
		exec := &ExecMoveTask{}
		proc := &Processor{
			Transfer: &model.Transfer{},
			Rule:     &model.Rule{},
		}

		Reset(func() { _ = os.Remove(script) })

		Convey("Given valid arguments", func() {
			args := map[string]string{
				"path":  "./" + script,
				"args":  "execmove.go",
				"delay": "1000",
			}

			Convey("Given that the command succeeds", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecOK), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(args, proc)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)

						Convey("Then the transfer destination should have "+
							"changed", func() {
							So(proc.Transfer.DestFile, ShouldEqual, "execmove.go")
						})
					})
				})
			})

			Convey("Given that the command sends a warning", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecWarn), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(args, proc)

					Convey("Then it should return a 'warning' error", func() {
						So(err, ShouldBeError, errWarning)
					})
				})
			})

			Convey("Given that the command fails", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecFail), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					_, err := exec.Run(args, proc)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)
						So(err, ShouldNotEqual, errWarning)
					})
				})
			})

			Convey("Given that the command delay expires", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecInfinite), 0700)
				So(err, ShouldBeNil)

				args["delay"] = "100"

				Convey("When running the task", func() {
					msg, err := exec.Run(args, proc)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)

						Convey("Then the message should say that the delay has expired", func() {
							So(msg, ShouldEqual, "max exec delay expired")
						})
					})
				})
			})
		})
	})
}
