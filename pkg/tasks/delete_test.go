package tasks

import (
	"context"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
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
	root := t.TempDir()

	Convey("Given a processor for a sending transfer", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_delete")
		task := &deleteTask{}
		srcFile := filepath.Join(root, "test.src")

		transCtx := &model.TransferContext{
			Transfer: &model.Transfer{LocalPath: srcFile},
		}

		So(fs.WriteFullFile(srcFile, []byte("Hello World")), ShouldBeNil)

		args := map[string]string{}

		Convey("Given that the file exists", func() {
			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})

				Convey("Then the local file should no longer be present in the system", func() {
					_, err := fs.Stat(srcFile)
					So(err, ShouldWrap, fs.ErrNotExist)
				})
			})
		})

		Convey("Given that the file does NOT exist", func() {
			So(fs.RemoveAll(srcFile), ShouldBeNil)

			Convey("When calling the run method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)

					Convey("Then error should say `no such file`", func() {
						So(err, ShouldWrap, fs.ErrNotExist)
					})
				})
			})
		})
	})
}
