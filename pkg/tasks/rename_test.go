package tasks

import (
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestRenameTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &RenameTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &RenameTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldEqual, "cannot create a rename task without a `path` argument")
			})
		})
	})
}

func TestRebaneTaskRun(t *testing.T) {
	Convey("Given a Processor for a sending Transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				SourcePath: "test/dummy",
				DestPath:   "test/dest",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]interface{}{
				"path": "test",
			}

			Convey("When caliing the `run` method", func() {
				task := &RenameTask{}
				_, err := task.Run(args, runner)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then transfer source path should be modified", func() {
					So(runner.Transfer.SourcePath, ShouldEqual, "test")
				})
			})
		})
	})

	Convey("Given a Processor for a sending Transfer", t, func() {
		runner := &Processor{
			Rule: &model.Rule{
				IsSend: false,
			},
			Transfer: &model.Transfer{
				SourcePath: "test/dummy",
				DestPath:   "test/dest",
			},
		}

		Convey("Given a model.Task", func() {
			args := map[string]interface{}{
				"path": "test",
			}

			Convey("When caliing the `run` method", func() {
				task := &RenameTask{}
				_, err := task.Run(args, runner)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then transfer DestPath should be modified", func() {
					So(runner.Transfer.DestPath, ShouldEqual, "test")
				})
			})
		})
	})
}
