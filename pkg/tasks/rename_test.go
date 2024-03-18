package tasks

import (
	"context"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestRenameTaskValidate(t *testing.T) {
	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{
			"path": "path/to/dest",
		}

		Convey("When calling the `Validate` method", func() {
			runner := &renameTask{}
			err := runner.Validate(args)

			Convey("Then it should NOT return an error", func() {
				So(err, ShouldBeNil)
			})
		})
	})

	Convey("Given a valid model.Task", t, func() {
		args := map[string]string{}

		Convey("When calling the `Validate` method", func() {
			runner := &renameTask{}
			err := runner.Validate(args)

			Convey("Then it should return an error", func() {
				So(err, ShouldNotBeNil)
			})

			Convey("Then error should say `need path argument`", func() {
				So(err.Error(), ShouldContainSubstring, "cannot create a rename task without a `path` argument")
			})
		})
	})
}

func TestRenameTaskRun(t *testing.T) {
	Convey("Given a Runner for a sending Transfer", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_rename")
		testFS := fstest.InitMemFS(c)
		task := &renameTask{}

		srcPath := makeURL("/rename.src")
		dstPath := makeURL("/rename.dst")

		So(fs.WriteFullFile(testFS, &srcPath, []byte("Hello World")), ShouldBeNil)
		So(fs.WriteFullFile(testFS, &dstPath, []byte("Goodbye World")), ShouldBeNil)

		transCtx := &model.TransferContext{
			Rule: &model.Rule{
				IsSend: true,
			},
			Transfer: &model.Transfer{
				LocalPath:  srcPath,
				RemotePath: "/remote/rename.src",
			},
			FS: testFS,
		}

		Convey("Given a valid new path", func() {
			args := map[string]string{"path": dstPath.String()}

			Convey("When calling the `run` method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)
				So(err, ShouldBeNil)

				Convey("Then transfer filepath should be modified", func() {
					So(transCtx.Transfer.LocalPath.String(), ShouldEqual, dstPath.String())
				})

				Convey("Then transfer source path should be modified", func() {
					So(transCtx.Transfer.RemotePath, ShouldEqual, "/remote/rename.dst")
				})
			})
		})

		Convey("Given an invalid new path", func() {
			args := map[string]string{"path": "memory:/dummy.file"}

			Convey("When calling the `run` method", func() {
				err := task.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should return an error", func() {
					So(fs.IsNotExist(err), ShouldBeTrue)
				})
			})
		})
	})
}
