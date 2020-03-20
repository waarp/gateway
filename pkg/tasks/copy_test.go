package tasks

import (
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCopyTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &CopyTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a invalid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &CopyTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldEqual, "cannot create a copy task without a `path` argument")
			})
		})
	})
}

func TestCopyTaskRun(t *testing.T) {
	Convey("Given a Processor for a sending transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				SourcePath: "test.src",
				DestPath:   "test.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]interface{}{
				"path": "test",
			}
			So(os.Mkdir("test", 0744), ShouldBeNil)
			Reset(func() {
				_ = os.RemoveAll("test")
			})

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(runner.Transfer.SourcePath, []byte("Hello World"), 0700)
				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(runner.Transfer.SourcePath)
				})

				Convey("When calling the `run` method", func() {
					task := &CopyTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test/test.src")
						So(err, ShouldBeNil)

						Reset(func() {
							_ = os.Remove("test/test.src")
						})
					})
				})
			})

			Convey("Given NO file to transfer", func() {

				Convey("When calling the `run` method", func() {
					task := &CopyTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err.Error(), ShouldEqual, "open test.src: no such file or directory")
					})

					Convey("Then the destination file should NOT exist", func() {
						_, err := os.Stat("test/test.src")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
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
				SourcePath: "test.src",
				DestPath:   "test.dst",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]interface{}{
				"path": "test",
			}
			So(os.Mkdir("test", 0o700), ShouldBeNil)
			Reset(func() {
				_ = os.RemoveAll("test")
			})

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(runner.Transfer.DestPath, []byte("Hello World"), 0o700)
				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(runner.Transfer.DestPath)
				})

				Convey("When calling the `run` method", func() {
					task := &CopyTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should NOT return error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the destination file should exist", func() {
						_, err := os.Stat("test/test.dst")
						So(err, ShouldBeNil)

						Reset(func() {
							_ = os.Remove("test/test.dst")
						})
					})
				})
			})

			Convey("Given NO file to transfer", func() {

				Convey("When calling the `run` method", func() {
					task := &CopyTask{}
					_, err := task.Run(args, runner)

					Convey("Then it should return error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err.Error(), ShouldEqual, "open test.dst: no such file or directory")
					})

					Convey("Then the destination file should NOT exist", func() {
						_, err := os.Stat("test/test.dst")
						So(err, ShouldNotBeNil)
						So(os.IsNotExist(err), ShouldBeTrue)
					})
				})
			})
		})
	})
}
