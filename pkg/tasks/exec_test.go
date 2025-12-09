package tasks

import (
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func getExecTransCtx(tb testing.TB) *model.TransferContext {
	tb.Helper()

	root := filepath.ToSlash(tb.TempDir())

	paths := &conf.PathsConfig{
		GatewayHome:   root,
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "tmp",
	}

	rule := &model.Rule{
		ID:     1,
		Name:   "push",
		IsSend: true,
	}

	client := &model.Client{
		ID:           20,
		Name:         "test_client",
		Protocol:     "test_protocol",
		LocalAddress: types.Addr("localhost", 9876),
	}

	partner := &model.RemoteAgent{
		ID:       10,
		Name:     "test_partner",
		Protocol: "test_protocol",
		Address:  types.Addr("localhost", 1234),
	}

	account := &model.RemoteAccount{
		ID:            100,
		RemoteAgentID: partner.ID,
		Login:         "test_login",
	}

	transfer := &model.Transfer{
		ID:               1000,
		RemoteTransferID: "abcd",
		RuleID:           rule.ID,
		ClientID:         utils.NewNullInt64(client.ID),
		RemoteAccountID:  utils.NewNullInt64(account.ID),
		SrcFilename:      "test.src",
		DestFilename:     "test.dst",
		LocalPath:        path.Join(paths.GatewayHome, paths.DefaultOutDir, "test.src"),
		RemotePath:       path.Join("remote", "dir", "test.dst"),
		Filesize:         1000,
		Start:            time.Now(),
		Status:           types.StatusRunning,
		Step:             types.StepPreTasks,
	}

	return &model.TransferContext{
		Transfer:           transfer,
		Rule:               rule,
		PreTasks:           model.Tasks{},
		PostTasks:          model.Tasks{},
		ErrTasks:           model.Tasks{},
		Client:             client,
		RemoteAgent:        partner,
		RemoteAgentCreds:   model.Credentials{},
		RemoteAccount:      account,
		RemoteAccountCreds: model.Credentials{},
		Paths:              paths,
	}
}

func TestExecValidate(t *testing.T) {
	Convey("Given an 'EXEC' task", t, func() {
		exec := &execTask{}

		Convey("Given valid arguments", func() {
			args := map[string]string{
				"path":  "cp",
				"args":  "exec.go exec_copy.go",
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

		Convey("Given that a optional parameter is missing", func() {
			args := map[string]string{
				"path":  "cp",
				"delay": "1000",
			}

			Convey("When validating the task", func() {
				err := exec.Validate(args)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that a parameter is empty", func() {
			args := map[string]string{
				"path":  "",
				"args":  "exec.go exec_copy.go",
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

func TestExecRun(t *testing.T) {
	transCtx := getExecTransCtx(t)

	Convey("Given an 'EXEC' task", t, func(c C) {
		logger := testhelpers.TestLogger(c, "task_exec")
		root := testhelpers.TempDir(c, "task_exec")
		scriptPath := filepath.Join(root, execScriptFile)

		exec := &execTask{}
		args := map[string]string{
			"path":  scriptPath,
			"args":  `"exec run test message"`,
			"delay": "0",
		}

		Convey("Given that the command succeeds", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecOK), 0o700), ShouldBeNil)

			Convey("When running the task", func() {
				err := exec.Run(t.Context(), args, nil, logger, transCtx, nil)

				Convey("Then it should NOT return an error", func() {
					So(err, ShouldBeNil)
				})
			})
		})

		Convey("Given that the command sends a warning", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecWarn), 0o700), ShouldBeNil)

			Convey("When running the task", func() {
				err := exec.Run(t.Context(), args, nil, logger, transCtx, nil)

				Convey("Then it should return a 'warning' error", func() {
					So(err, ShouldHaveSameTypeAs, &WarningError{})
					So(err, ShouldBeError, "exit status 1")
				})
			})
		})

		Convey("Given that the command fails", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecFail), 0o700), ShouldBeNil)

			Convey("When running the task", func() {
				err := exec.Run(t.Context(), args, nil, logger, transCtx, nil)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, "exit status 2")
				})
			})
		})

		Convey("Given that the command delay expires", func() {
			So(os.WriteFile(scriptPath, []byte(scriptExecInfinite), 0o700), ShouldBeNil)

			args["delay"] = "100"

			Convey("When running the task", func() {
				err := exec.Run(t.Context(), args, nil, logger, transCtx, nil)

				Convey("Then it should return an error", func() {
					So(err, ShouldBeError, ErrExecTimeout)
				})
			})
		})
	})
}
