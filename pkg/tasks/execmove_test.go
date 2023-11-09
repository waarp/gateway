package tasks

import (
	"context"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils/testhelpers"
)

func TestExecMoveValidate(t *testing.T) {
	Convey("Given an 'EXECMOVE' task", t, func() {
		exec := &execMoveTask{}

		Convey("Given valid argument", func() {
			args := map[string]string{
				"path":  "cp",
				"args":  "file1 file2",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that a parameter is NOT the valid type", func() {
			args := map[string]string{
				"path":  "cp",
				"args":  "file1 file2",
				"delay": "true",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("Given that a parameter is missing", func() {
			args := map[string]string{
				"path":  "cp",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})

		Convey("Given that a parameter is empty", func() {
			args := map[string]string{
				"path":  "",
				"args":  "file1 file2",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should return an error", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}

func TestExecMoveRun(t *testing.T) {
	Convey("Given an 'EXECMOVE' task", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_execmove")
		testFS := fstest.InitMemFS(c)
		root := testhelpers.TempDir(c, "task_execmove")
		scriptPath := filepath.Join(root, execMoveScriptFile)

		exec := &execMoveTask{}
		transCtx := &model.TransferContext{
			Rule:     &model.Rule{IsSend: false},
			Transfer: &model.Transfer{},
			FS:       testFS,
		}

		srcFile := filepath.Join(root, "test.src")
		dstFile := filepath.Join(root, "test.dst")

		So(os.WriteFile(srcFile, []byte("Hello world"), 0o600), ShouldBeNil)

		args := map[string]string{
			"path":  scriptPath,
			"args":  srcFile + " " + dstFile,
			"delay": "0",
		}

		Convey("Given that the command succeeds", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecMove), 0o700), ShouldBeNil)

			Convey("When running the task", func() {
				err := exec.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)

					Convey("Then the transfer filepath should have changed", func() {
						dstURL := url.URL{
							Scheme:   "file",
							OmitHost: true,
							Path:     path.Join("/", filepath.ToSlash(dstFile)),
						}

						So(transCtx.Transfer.LocalPath.String(), ShouldEqual,
							dstURL.String())
					})
				})
			})
		})

		Convey("Given that the command sends a warning", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecWarn), 0o700), ShouldBeNil)

			Convey("When running the task", func() {
				err := exec.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should return a 'warning' error", func() {
					So(err, ShouldHaveSameTypeAs, &warningError{})
				})
			})
		})

		Convey("Given that the command fails", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecFail), 0o700), ShouldBeNil)

			Convey("When running the task", func() {
				err := exec.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError)
					So(err, ShouldNotHaveSameTypeAs, &warningError{})
				})
			})
		})

		Convey("Given that the command delay expires", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecInfinite), 0o700), ShouldBeNil)

			args["delay"] = "100"

			Convey("When running the task", func() {
				err := exec.Run(context.Background(), args, nil, logger, transCtx)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, errCommandTimeout)
				})
			})
		})
	})
}
