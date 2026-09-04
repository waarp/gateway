package tasks

import (
	"bytes"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/logtest"
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
		Identifier: model.ID(1),
		Name:       "push",
		IsSend:     true,
	}

	client := &model.Client{
		Identifier:   model.ID(20),
		Name:         "test_client",
		Protocol:     "test_protocol",
		LocalAddress: types.Addr("localhost", 9876),
	}

	partner := &model.RemoteAgent{
		Identifier: model.ID(10),
		Name:       "test_partner",
		Protocol:   "test_protocol",
		Address:    types.Addr("localhost", 1234),
	}

	account := &model.RemoteAccount{
		Identifier:    model.ID(100),
		RemoteAgentID: partner.ID,
		Login:         "test_login",
	}

	transfer := &model.Transfer{
		Identifier:       model.ID(1000),
		RemoteTransferID: "abcd",
		RuleID:           rule.ID,
		ClientID:         client.NullableID(),
		RemoteAccountID:  account.NullableID(),
		SrcFilename:      "test.src",
		DestFilename:     "test.dst",
		LocalPath:        path.Join(paths.GatewayHome, paths.DefaultOutDir, "test.src"),
		RemotePath:       path.Join("remote", "dir", "test.dst"),
		Filesize:         1000,
		Start:            time.Now(),
		Status:           types.StatusRunning,
		Step:             types.StepPreTasks,
		TransferInfo:     map[string]any{},
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
	t.Parallel()

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
	t.Parallel()

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

// A program that succeeds may still have something to say on stderr, and until
// now runExec only journalled that buffer when the program had failed. The
// compatibility shims installed at the pre-0.17 helper paths write their
// deprecation notice there, so on every working invocation -- which is to say
// for the whole population the notice is aimed at -- it was discarded and the
// administrator was never told to migrate.
func TestRunExecReportsStderrOfASuccessfulProgram(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	logger := logtest.GetTestLogger(t, logtest.WithWriter(logs))

	_, err := runExec(t.Context(), logger, getExecTransCtx(t), map[string]string{
		"path": shellCmd(),
		"args": shellCmdArgs(`'echo the-real-output; echo a-deprecation-notice >&2'`),
	})
	require.NoError(t, err)

	assert.Contains(t, logs.String(), "a-deprecation-notice",
		"stderr of a successful program must be journalled, not dropped")
}

// The warning must not disturb what the task reads back: EXECMOVE takes the
// last line of stdout as the transferred file's new path, and EXECOUTPUT parses
// it for a NEWFILENAME: prefix.
func TestRunExecKeepsStdoutOutOfTheStderrWarning(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	logger := logtest.GetTestLogger(t, logtest.WithWriter(logs))

	output, err := runExec(t.Context(), logger, getExecTransCtx(t), map[string]string{
		"path": shellCmd(),
		"args": shellCmdArgs(`'echo the-real-output; echo a-deprecation-notice >&2'`),
	})
	require.NoError(t, err)

	assert.Equal(t, "the-real-output"+endl(), output.String(),
		"the returned buffer must carry stdout alone")
}

// A program that writes nothing to stderr must not produce an empty warning
// line, which would be pure noise on every single transfer.
//
// The assertion is on the message rather than on a "[WARNING]" marker: the
// handler pads the level to eight characters ("%-8s"), so a literal
// "[WARNING]" never appears in the output and such a check could never fail.
func TestRunExecStaysSilentWhenStderrIsEmpty(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	logger := logtest.GetTestLogger(t, logtest.WithWriter(logs))

	_, err := runExec(t.Context(), logger, getExecTransCtx(t), map[string]string{
		"path": shellCmd(),
		"args": shellCmdArgs(`'echo only-stdout'`),
	})
	require.NoError(t, err)

	assert.NotContains(t, logs.String(), "Program wrote to stderr",
		"a program with no stderr output must not be reported as a warning")
}

func TestRunExecEnvVars(t *testing.T) {
	t.Parallel()

	logger := logtest.GetTestLogger(t)
	ctx := getExecTransCtx(t)

	const (
		tiKey = "key"
		tiVal = "val"
	)
	ctx.Transfer.TransferInfo[tiKey] = tiVal

	path, args := "sh", `-c "echo $WAARP_TRANSFERID $WAARP_TI_key"`
	if isWindowsRuntime() {
		path, args = "cmd", `/C "echo %WAARP_TRANSFERID% %WAARP_TI_key%"`
	}

	output, err := runExec(t.Context(), logger, ctx, map[string]string{
		"path": path,
		"args": args,
	})
	require.NoError(t, err)

	assert.Contains(t, output.String(), utils.FormatInt(ctx.Transfer.ID))
	assert.Contains(t, output.String(), tiVal)
}
