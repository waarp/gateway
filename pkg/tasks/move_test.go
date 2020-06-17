package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMoveTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &MoveTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a invalid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &MoveTask{}
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
	Convey("Given a Processor for a sending transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				TrueFilepath: "move_out.src",
				SourceFile:   "move_out.src",
				DestFile:     "move_out.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]string{
				"path": "move_out",
			}
			So(os.Mkdir("move_out", 0700), ShouldBeNil)
			Reset(func() {
				So(os.RemoveAll("move_out"), ShouldBeNil)
			})

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile("move_out.src", []byte("Hello World"), 0o700)
				So(err, ShouldBeNil)
				Reset(func() { _ = os.Remove("move_out.src") })

				Convey("When calling the `run` method", func() {
					task := &MoveTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("move_out/move_out.src")
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer true file path should be modified", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "move_out/move_out.src")
					})

					Convey("Then the transfer source path should NOT be modified", func() {
						So(runner.Transfer.SourceFile, ShouldEqual, "move_out.src")
					})
				})
			})

			Convey("Given NO file to transfer", func() {

				Convey("When calling the `run` method", func() {
					task := &MoveTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldBeError, fileNotFound("move_out.src"))
					})

					Convey("Then the destination file should NOT exist", func() {
						_, err := os.Stat("move_out/move_out.src")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})

					Convey("Then the transfer true file path should NOT be modified", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "move_out.src")
					})

					Convey("Then the transfer source path should NOT be modified", func() {
						So(runner.Transfer.SourceFile, ShouldEqual, "move_out.src")
					})
				})
			})
		})
	})

	Convey("Given a Processor for a receiving transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: false,
			},
			Transfer: &model.Transfer{
				TrueFilepath: "move_in.dst",
				SourceFile:   "move_in.src",
				DestFile:     "move_in.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]string{
				"path": "move_in",
			}
			So(os.Mkdir("move_in", 0700), ShouldBeNil)
			Reset(func() { _ = os.RemoveAll("move_in") })

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile("move_in.dst", []byte("Hello World"), 0o700)
				So(err, ShouldBeNil)
				Reset(func() { _ = os.Remove("move_in.dst") })

				Convey("When calling the `run` method", func() {
					task := &MoveTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("move_in/move_in.dst")
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer true file path should be modified", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "move_in/move_in.dst")
					})

					Convey("Then the transfer dest path shouldn't be modified", func() {
						So(runner.Transfer.DestFile, ShouldEqual, "move_in.dst")
					})
				})
			})

			Convey("Given NO file to transfer", func() {

				Convey("When calling the `run` method", func() {
					task := &MoveTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldBeError, fileNotFound("move_in.dst"))
					})

					Convey("Then the destination file should NOT exist", func() {
						_, err := os.Stat("move_in/move_in.dst")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})

					Convey("Then the transfer true file path should NOT be modified", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "move_in.dst")
					})

					Convey("Then the transfer dest path shouldn't be modified", func() {
						So(runner.Transfer.DestFile, ShouldEqual, "move_in.dst")
					})
				})
			})
		})
	})
}
