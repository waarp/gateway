package tasks

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils/testhelpers"
	. "github.com/smartystreets/goconvey/convey"
)

func TestDeleteTaskValidate(t *testing.T) {
	Convey("Given a model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the validate method", func() {
			task := &deleteTask{}
			err := task.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})
}

func TestDeleteTaskRun(t *testing.T) {
	Convey("Given a processor for a sending transfer", t, func(c C) {
		root := testhelpers.TempDir(c, "task_delete")
		task := &deleteTask{}
		srcFile := filepath.Join(root, "test.src")

		info := &model.TransferContext{Transfer: &model.Transfer{
			TrueFilepath: utils.NormalizePath(srcFile),
		}}

		So(ioutil.WriteFile(srcFile, []byte("Hello World"), 0o700), ShouldBeNil)
		args := map[string]string{}

		Convey("Given that the file exists", func() {

			Convey("When calling the run method", func() {
				_, err := task.Run(args, nil, info, nil)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the source file should no longer be present in the system", func() {
					_, err := os.Stat(info.Transfer.TrueFilepath)
					So(os.IsNotExist(err), ShouldBeTrue)
				})
			})
		})

		Convey("Given that the file does NOT exist", func() {
			So(os.Remove(srcFile), ShouldBeNil)

			Convey("When calling the run method", func() {
				_, err := task.Run(args, nil, info, nil)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})

				Convey("Then error should say `no such file`", func() {
					So(err, ShouldBeError, &errFileNotFound{"delete file", srcFile})
				})
			})
		})
	})
}
