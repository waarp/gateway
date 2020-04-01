package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

const scriptExecOK = `#!/bin/sh
echo $1`

const scriptExecWarn = `#!/bin/sh
echo $1
exit 1`

const scriptExecFail = `#!/bin/sh
echo $1
exit 2`

const scriptExecInfinite = `#!/bin/sh
while [ true ]; do
  echo $1
done`

func TestExecValidate(t *testing.T) {
	Convey("Given an 'EXEC' task", t, func() {
		exec := &ExecTask{}

		Convey("Given valid arguments", func() {
			args := map[string]string{
				"path":  "cp",
				"args":  "exec.go exec_copy.go",
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
				"args":  "exec.go exec_copy.go",
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

func TestExecRun(t *testing.T) {
	script := "exec_test_script.sh"
	_ = os.Remove(script)

	Convey("Given an 'EXEC' task", t, func() {
		exec := &ExecTask{}
		args := map[string]string{
			"path":  "./" + script,
			"args":  "'exec run test message'",
			"delay": "1000",
		}
		Reset(func() { _ = os.Remove(script) })

		Convey("Given that the task is valid", func() {

			Convey("Given that the command succeeds", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecOK), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					msg, err := exec.Run(args, nil)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)

						Convey("Then the message should be empty", func() {
							So(msg, ShouldBeEmpty)
						})
					})
				})
			})

			Convey("Given that the command sends a warning", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecWarn), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					msg, err := exec.Run(args, nil)

					Convey("Then it should return a 'warning' error", func() {
						So(err, ShouldBeError, errWarning)

						Convey("Then the message should say the cause of warning", func() {
							So(msg, ShouldEqual, "exit status 1")
						})
					})
				})
			})

			Convey("Given that the command fails", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecFail), 0700)
				So(err, ShouldBeNil)

				Convey("When running the task", func() {
					msg, err := exec.Run(args, nil)

					Convey("Then it should return an error", func() {
						So(err, ShouldBeError)

						Convey("Then the message should say the cause of warning", func() {
							So(msg, ShouldEqual, "exit status 2")
						})
					})
				})
			})

			Convey("Given that the command delay expires", func() {
				err := ioutil.WriteFile(script, []byte(scriptExecInfinite), 0700)
				So(err, ShouldBeNil)

				args["delay"] = "100"

				Convey("When running the task", func() {
					msg, err := exec.Run(args, nil)

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
