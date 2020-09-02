package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
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
				TrueFilepath: "move_rename_out.src",
				SourceFile:   "move_rename_out.src",
				DestFile:     "move_rename_out.dst",
			},
		}

		Convey("Given a model.Task", func(c C) {
			renameRoot := testhelpers.TempDir(c, "move_test_out")
			args := map[string]string{
				"path": renameRoot + "/move_rename_out_new.src",
			}

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile("move_rename_out.src", []byte("Hello World"), 0o600)
				So(err, ShouldBeNil)
				Reset(func() { _ = os.Remove(runner.Transfer.TrueFilepath) })

				Convey("When calling the `run` method", func() {
					task := &MoveRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat(renameRoot + "/move_rename_out_new.src")
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer true file path should be modified", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, renameRoot+"/move_rename_out_new.src")
					})

					Convey("Then the transfer source path should be modified", func() {
						So(runner.Transfer.SourceFile, ShouldEqual, "move_rename_out_new.src")
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
						So(err, ShouldBeError, fileNotFound("move_rename_out.src"))
					})

					Convey("Then the destination file NOT should exist", func() {
						fi, err := ioutil.ReadDir(renameRoot)
						So(err, ShouldBeNil)
						So(fi, ShouldBeEmpty)
					})

					Convey("Then the transfer true file path should  NOT be modifier", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "move_rename_out.src")
					})

					Convey("Then the transfer source path should NOT be modifier", func() {
						So(runner.Transfer.SourceFile, ShouldEqual, "move_rename_out.src")
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
				TrueFilepath: "move_rename_in.dst",
				SourceFile:   "move_rename_in.src",
				DestFile:     "move_rename_in.dst",
			},
		}

		Convey("Given a model.Task", func(c C) {
			renameRoot := testhelpers.TempDir(c, "move_rename_in")
			args := map[string]string{
				"path": renameRoot + "/move_rename_in_new.dst",
			}

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile("move_rename_in.dst", []byte("Hello World"), 0o600)
				So(err, ShouldBeNil)
				Reset(func() { _ = os.Remove("move_rename_in.dst") })

				Convey("When calling the `run` method", func() {
					task := &MoveRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat(renameRoot + "/move_rename_in_new.dst")
						So(err, ShouldBeNil)
					})

					Convey("Then the transfer true file path should be modifier", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, renameRoot+"/move_rename_in_new.dst")
					})

					Convey("Then the transfer destination path should be modifier", func() {
						So(runner.Transfer.DestFile, ShouldEqual, "move_rename_in_new.dst")
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
						So(err, ShouldBeError, fileNotFound("move_rename_in.dst"))
					})

					Convey("Then the destination file NOT should exist", func() {
						_, err := os.Stat("move_rename_in/move_rename_in_new.dst")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})

					Convey("Then the transfer true file path should NOT be modified", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "move_rename_in.dst")
					})

					Convey("Then the transfer destination path should NOT be modified", func() {
						So(runner.Transfer.DestFile, ShouldEqual, "move_rename_in.dst")
					})
				})
			})
		})
	})
}
