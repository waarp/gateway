package tasks

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
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

		transCtx := &model.TransferContext{Transfer: &model.Transfer{
			LocalPath: utils.ToOSPath(srcFile),
		}}

		So(ioutil.WriteFile(srcFile, []byte("Hello World"), 0o700), ShouldBeNil)
		args := map[string]string{}

		Convey("Given that the file exists", func() {
			Convey("When calling the run method", func() {
				_, err := task.Run(context.Background(), args, nil, transCtx)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the local file should no longer be present in the system", func() {
					_, err := os.Stat(utils.ToOSPath(transCtx.Transfer.LocalPath))
					So(os.IsNotExist(err), ShouldBeTrue)
				})
			})
		})

		Convey("Given that the file does NOT exist", func() {
			So(os.Remove(srcFile), ShouldBeNil)

			Convey("When calling the run method", func() {
				_, err := task.Run(context.Background(), args, nil, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})

				Convey("Then error should say `no such file`", func() {
					So(err, ShouldBeError, &fileNotFoundError{"delete file", srcFile})
				})
			})
		})
	})
}
