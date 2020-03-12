package tasks

import (
	"testing"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSetup(t *testing.T) {
	Convey("Given a Task with some replacement variables", t, func() {
		task := &model.Task{
			Type: "DUMMY",
			Args: []byte(`{"test":"#RULE#", "date":"#DATE#", "path":"#OUTPATH#", "msg":"#ERRORMSG#"}`),
		}

		Convey("Given a Processor", func() {
			r := &Processor{
				Rule: &model.Rule{
					Name:   "Test",
					IsSend: true,
					Path:   "path/to/test",
				},
				Transfer: &model.Transfer{
					Error: model.TransferError{
						Details: `", "bad":1`,
					},
				},
			}

			Convey("When calling the `setup` function", func() {
				res, err := r.setup(task)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then res should contain an entry `test`", func() {
						val, ok := res["test"]
						So(ok, ShouldBeTrue)

						Convey("Then res[test] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Rule.Name)
						})
					})

					Convey("Then res should contain an entry `date`", func() {
						val, ok := res["date"]
						So(ok, ShouldBeTrue)

						Convey("Then res[date] should contain the resolved variable", func() {
							today := time.Now().Format("20060102")
							So(val, ShouldEqual, today)
						})
					})

					Convey("Then res should contain an entry `path`", func() {
						val, ok := res["path"]
						So(ok, ShouldBeTrue)

						Convey("Then res[path] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Rule.Path)
						})
					})

					Convey("Then res should contain an entry `msg`", func() {
						val, ok := res["msg"]
						So(ok, ShouldBeTrue)

						Convey("Then res[msg] should contain the resolved variable", func() {
							So(val, ShouldEqual, r.Transfer.Error.Details)
						})
					})
				})
			})
		})
	})
}
