package tasks

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDeleteTaskValidate(t *testing.T) {
	Convey("Given a model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the validate method", func() {
			task := &DeleteTask{}
			err := task.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestDeleteTaskRun(t *testing.T) {
	Convey("Given a processor for a sending transfer", t, func() {
		processor := &Processor{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				TrueFilepath: "out.file",
				SourceFile:   "out.file",
				DestFile:     "in.file",
			},
		}

		Convey("Given a model.Task and a file to transfer", func() {
			args := map[string]string{}

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(processor.Transfer.TrueFilepath, []byte("Hello World"), 0700)

				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(processor.Transfer.TrueFilepath)
				})

				Convey("When calling the run method", func() {
					task := &DeleteTask{}
					_, err := task.Run(args, processor)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the source file should not be present in the system", func() {
						_, err := os.Stat(processor.Transfer.TrueFilepath)
						So(os.IsNotExist(err), ShouldBeTrue)
					})
				})
			})

			Convey("Given no file to transfer", func() {

				Convey("When calling the run method", func() {
					task := &DeleteTask{}
					_, err := task.Run(args, processor)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err.Error(), ShouldEqual,
							fmt.Sprintf("remove %s: no such file or directory",
								processor.Transfer.TrueFilepath))
					})
				})
			})
		})
	})

	Convey("Given a processor for a receiving transfer", t, func() {
		processor := &Processor{
			Rule: &model.Rule{
				IsSend: false,
			},
			Transfer: &model.Transfer{
				TrueFilepath: "in.file",
				SourceFile:   "out.file",
				DestFile:     "in.file",
			},
		}

		Convey("Given a model.Task and a file to transfer", func() {
			args := map[string]string{}

			Convey("Given a file to transfer", func() {
				err := ioutil.WriteFile(processor.Transfer.TrueFilepath, []byte("Hello World"), 0700)

				So(err, ShouldBeNil)

				Reset(func() {
					_ = os.Remove(processor.Transfer.TrueFilepath)
				})

				Convey("When calling the run method", func() {
					task := &DeleteTask{}
					_, err := task.Run(args, processor)

					Convey("Then it should NOT return an error", func() {
						So(err, ShouldBeNil)
					})

					Convey("Then the source file should not be present in the system", func() {
						_, err := os.Stat(processor.Transfer.TrueFilepath)
						So(os.IsNotExist(err), ShouldBeTrue)
					})
				})
			})

			Convey("Given no file to transfer", func() {

				Convey("When calling the run method", func() {
					task := &DeleteTask{}
					_, err := task.Run(args, processor)

					Convey("Then it should return an error", func() {
						So(err, ShouldNotBeNil)
					})

					Convey("Then error should say `no such file`", func() {
						So(err.Error(), ShouldEqual,
							fmt.Sprintf("remove %s: no such file or directory", processor.Transfer.TrueFilepath))
					})
				})
			})
		})
	})
}
