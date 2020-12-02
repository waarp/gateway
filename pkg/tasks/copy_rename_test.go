package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCopyRenameTaskValidate(t *testing.T) {
	Convey("Given a valid argument", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &CopyRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given an invalid argument", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &CopyRenameTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldEqual, "cannot create a copy_rename task without a `path` argument")
			})
		})
	})
}

func TestCopyRenameTaskRun(t *testing.T) {
	Convey("Given a Processor for a sending transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				TrueFilepath: "test.src",
				SourceFile:   "test.src",
				DestFile:     "test.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]string{
				"path": "test/dummy.test",
			}
			So(os.Mkdir("test", 0o744), ShouldBeNil)
			Reset(func() {
				_ = os.RemoveAll("test")
			})

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(runner.Transfer.TrueFilepath, []byte("Hello World"), 0o700)
				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(runner.Transfer.TrueFilepath)
				})

				Convey("When calling the `run` method", func() {
					task := &CopyRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test/dummy.test")
						So(err, ShouldBeNil)

						Reset(func() {
							_ = os.Remove("test/dummy.test")
						})
					})

					Convey("Then the transfer true file path should NOT be modifier", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "test.src")
					})

					Convey("Then the transfer source path should NOT be modifier", func() {
						So(runner.Transfer.SourceFile, ShouldEqual, "test.src")
					})
				})

				Convey("Given the target is a non-existing subdir", func() {
					args["path"] = "test/subdir/subsubdir/dummy.test"

					Convey("Given the target can be created", func() {
						Convey("When the task is run", func() {
							task := &CopyRenameTask{}
							_, err := task.Run(args, runner)

							Convey("Then it should return no error", func() {
								So(err, ShouldBeNil)
							})

							Convey("Then the target file exists", func() {
								_, err := os.Stat("test/subdir/subsubdir/dummy.test")
								So(err, ShouldBeNil)
							})
						})
					})

					Convey("Given the target CANNOT be created", func() {
						err := os.Mkdir("test/subdir", 0o400)
						So(err, ShouldBeNil)

						Convey("When the task is run", func() {
							task := &CopyRenameTask{}
							_, err := task.Run(args, runner)

							Convey("Then it should return an error", func() {
								So(err, ShouldBeError)
							})

							Convey("Then the target file does not exist", func() {
								_, err := os.Stat("test/subdir/subsubdir/dummy.test")
								So(err, ShouldBeError)
							})
						})
					})
				})
			})

			Convey("Given NO file to transfer", func() {
				Convey("When calling the `run` method", func() {
					task := &CopyRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldBeError, fileNotFound("test.src"))
					})

					Convey("Then the destination file should NOT exist", func() {
						_, err := os.Stat("test/dummy.test")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})

					Convey("Then the transfer true file path should NOT be modifier", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "test.src")
					})

					Convey("Then the transfer source path should NOT be modifier", func() {
						So(runner.Transfer.SourceFile, ShouldEqual, "test.src")
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
				TrueFilepath: "test.dst",
				SourceFile:   "test.src",
				DestFile:     "test.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]string{
				"path": "test/dummy.test",
			}
			So(os.Mkdir("test", 0o700), ShouldBeNil)
			Reset(func() {
				_ = os.RemoveAll("test")
			})

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(runner.Transfer.TrueFilepath, []byte("Hello World"), 0o700)
				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(runner.Transfer.TrueFilepath)
				})

				Convey("When calling the `run` method", func() {
					task := &CopyRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test/dummy.test")
						So(err, ShouldBeNil)

						Reset(func() {
							_ = os.Remove("test/dummy.test")
						})
					})

					Convey("Then the transfer true file path should NOT be modifier", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "test.dst")
					})

					Convey("Then the transfer destination path should NOT be modifier", func() {
						So(runner.Transfer.DestFile, ShouldEqual, "test.dst")
					})
				})
			})

			Convey("Given NO file to transfer", func() {
				Convey("When calling the `run` method", func() {
					task := &CopyRenameTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldBeError, fileNotFound("test.dst"))
					})

					Convey("Then the destination file should NOT exist", func() {
						_, err := os.Stat("test/dummy.test")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})

					Convey("Then the transfer true file path should NOT be modifier", func() {
						So(runner.Transfer.TrueFilepath, ShouldEqual, "test.dst")
					})

					Convey("Then the transfer destination path should NOT be modifier", func() {
						So(runner.Transfer.DestFile, ShouldEqual, "test.dst")
					})
				})
			})
		})
	})
}
