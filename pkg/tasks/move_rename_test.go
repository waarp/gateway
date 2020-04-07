package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMoveRenameTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &MoveRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a invalid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &MoveRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldEqual, "cannot create a move_rename task without a `path` argument")
			})
		})
	})
}

func TestMoveRenameTaskRun(t *testing.T) {
	Convey("Given a Processor for a sending transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				SourceFile: "test.src",
				DestFile:   "test.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]string{
				"path": "test/dest",
			}

			So(os.Mkdir("test", 744), ShouldBeNil)
			Reset(func() {
				_ = os.RemoveAll("test")
			})

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(runner.Transfer.SourceFile, []byte("Hello World"), 0o700)
				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(runner.Transfer.SourceFile)
				})

				Convey("When calling the `run` method", func() {
					task := &MoveRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test/dest")
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer source path should be modifier", func() {
						So(runner.Transfer.SourceFile, ShouldEqual, "test/dest")
					})

					Reset(func() {
						_ = os.Remove("test/dest")
					})
				})
			})

			Convey("Given NO file to transfer", func() {

				Convey("When calling the `run` method", func() {
					task := &MoveRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then it error should say `no such file`", func() {
						So(err.Error(), ShouldEqual, "rename test.src test/dest: no such file or directory")
					})

					Convey("Then the destination file NOT should exist", func() {
						_, err := os.Stat("test/dest")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})
				})
			})
		})
	})

	Convey("Given a Processor for a sending transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: false,
			},
			Transfer: &model.Transfer{
				SourceFile: "test.src",
				DestFile:   "test.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]string{
				"path": "test/dest",
			}

			So(os.Mkdir("test", 744), ShouldBeNil)
			Reset(func() {
				_ = os.RemoveAll("test")
			})

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(runner.Transfer.DestFile, []byte("Hello World"), 0o700)
				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(runner.Transfer.DestFile)
				})

				Convey("When calling the `run` method", func() {
					task := &MoveRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test/dest")
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer dest path should be modifier", func() {
						So(runner.Transfer.DestFile, ShouldEqual, "test/dest")
					})

					Reset(func() {
						_ = os.Remove("test/dest")
					})
				})
			})

			Convey("Given NO file to transfer", func() {

				Convey("When calling the `run` method", func() {
					task := &MoveRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then it error should say `no such file or directory`", func() {
						So(err.Error(), ShouldEqual, "rename test.dst test/dest: no such file or directory")
					})

					Convey("Then the destination file NOT should exist", func() {
						_, err := os.Stat("test/dest")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})
				})
			})
		})
	})
}
