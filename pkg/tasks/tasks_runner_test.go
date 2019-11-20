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
			Args: []byte("{\"test\":\"#RULE#\", \"date\":\"#DATE#\", \"path\":\"#OUTPATH#\"}"),
		}

		Convey("Given a TasksRunner", func() {
			r := &TasksRunner{
				Rule: &model.Rule{
					Name: "Test",
					IsSend: true,
					Path: "path/to/test",
				},
				Transfer: &model.Transfer{
				},
			}

			Convey("When calling the `setup` function", func() {
				res, err := r.setup(task)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				val, ok := res["test"]

				Convey("Then res should contain an entry `test`", func() {
					So(ok, ShouldBeTrue)
				})

				Convey("Then res[test] should contain the resolved variable", func() {
					So(val.(string), ShouldEqual, r.Rule.Name)
				})

				val, ok = res["date"]

				Convey("Then res should contain an entry `date`", func() {
					So(ok, ShouldBeTrue)
				})

				Convey("Then res[date] should contain the resolved variable", func() {
					today := time.Now().Format("20060102")
					So(val.(string), ShouldEqual, today)
				})

				val, ok = res["path"]

				Convey("Then res should contain an entry `path`", func() {
					So(ok, ShouldBeTrue)
				})

				Convey("Then res[path] should contain the resolved variable", func() {
					So(val.(string), ShouldEqual, r.Rule.Path)
				})
			})
		})
	})
}
