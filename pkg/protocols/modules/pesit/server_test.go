//go:build todo

package pesit

import (
	"context"
	"io"
	"path"
	"testing"
	"time"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// Ensures Service implements the test server interface.
var _ gwtesting.TestService = &Service{}

func TestServerSelectFile(t *testing.T) {
	t.SkipNow()

	db := dbtest.TestDatabase(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	testStart := time.Now()

	getHandler := func() *transferHandler {
		return &transferHandler{
			db:      db,
			logger:  gwtesting.Logger(t),
			agent:   ctx.Server,
			account: ctx.LocalAccount,
			conf: &ServerConfig{
				CheckPointConfig: CheckPointConfig{
					DisableRestart:     false,
					DisableCheckpoints: false,
					CheckpointSize:     65535,
					CheckpointWindow:   1,
				},
			},
			ctx:    context.Background(),
			cancel: func(cause error) {},
		}
	}

	t.Run("Push transfer", func(t *testing.T) {
		const (
			fileName    = "test_push_src.txt"
			fileContent = "Hello push"
		)

		srcFilePath := fs.JoinPath(conf.GlobalConfig.Paths.GatewayHome,
			ctx.Server.RootDir, ctx.ServerRulePush.LocalDir, fileName)
		require.NoError(t, fs.WriteFullFile(srcFilePath, []byte(fileContent)))

		var (
			fileSize         = int64(len(fileContent))
			reservationSpace = uint32(fileSize / bytesPerKB)
			filePath         = path.Join(ctx.ServerRulePush.Path, fileName)
		)

		mkReq := func(transferID uint32) *pesit.ServerTransfer {
			ct := pesit.NewTransfer(pesit.MethodRecv, filePath)
			ct.SetReservationSpace(reservationSpace, pesit.UnitKB)

			buf := &testhelpers.Buffer{}
			req := pesit.ServerTransferTest(ct, buf)

			return &req
		}

		checkTransfer := func(tb testing.TB, transferID uint32, check *model.Transfer) {
			tb.Helper()

			transIDStr := utils.FormatUint(transferID)

			assert.Equal(t, transIDStr, check.RemoteTransferID)
			assert.Equal(t, ctx.ServerRulePush.ID, check.RuleID)
			assert.Equal(t, ctx.LocalAccount.ID, check.LocalAccountID.Int64)
			assert.Equal(t, fileName, check.SrcFilename)
			assert.Equal(t, srcFilePath, check.LocalPath)
			assert.WithinRange(t, check.Start, time.Now(), testStart)
			assert.Equal(t, types.StatusRunning, check.Status)
			assert.Equal(t, types.StepPreTasks, check.Step)
		}

		t.Run("New transfer", func(t *testing.T) {
			const transferID uint32 = 1

			req := mkReq(transferID)
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			checkTransfer(t, transferID, handler.pip.TransCtx.Transfer)
		})

		t.Run("Recovery", func(t *testing.T) {
			const transferID uint32 = 2

			oldTransfer := model.Transfer{
				RemoteTransferID: utils.FormatUint(transferID),
				RuleID:           ctx.ServerRulePush.ID,
				LocalAccountID:   utils.NewNullInt64(ctx.LocalAccount.ID),
				SrcFilename:      fileName,
				LocalPath:        srcFilePath,
				Filesize:         int64(len(fileContent)),
				Start:            testStart,
				Status:           types.StatusPaused,
				Step:             types.StepPreTasks,
				Progress:         0,
				TaskNumber:       0,
			}
			require.NoError(t, db.Insert(&oldTransfer).Run())

			req := mkReq(transferID)
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			checkTransfer(t, transferID, handler.pip.TransCtx.Transfer)
		})
	})

	t.Run("Pull transfer", func(t *testing.T) {
		const (
			fileName       = "test_pull_dst.txt"
			fileSize int64 = 1234
		)

		var (
			reservationSpace = uint32(fileSize / bytesPerKB)
			filePath         = path.Join(ctx.ServerRulePull.Path, fileName)
			dstFilePath      = fs.JoinPath(conf.GlobalConfig.Paths.GatewayHome,
				ctx.Server.RootDir, ctx.ServerRulePull.LocalDir, fileName)
		)

		mkReq := func() *pesit.ServerTransfer {
			ct := pesit.NewTransfer(pesit.MethodRecv, filePath)
			ct.SetReservationSpace(reservationSpace, pesit.UnitKB)

			buf := &testhelpers.Buffer{}
			req := pesit.ServerTransferTest(ct, buf)

			return &req
		}

		checkTransfer := func(tb testing.TB, check *model.Transfer) {
			tb.Helper()
			assert.Equal(t, ctx.ServerRulePush.ID, check.RuleID)
			assert.Equal(t, ctx.LocalAccount.ID, check.LocalAccountID.Int64)
			assert.Equal(t, fileName, check.SrcFilename)
			assert.Equal(t, fileName, check.DestFilename)
			assert.Equal(t, dstFilePath, check.LocalPath)
			assert.WithinRange(t, check.Start, time.Now(), testStart)
			assert.Equal(t, types.StatusRunning, check.Status)
			assert.Equal(t, types.StepPreTasks, check.Step)
		}

		t.Run("New transfer", func(t *testing.T) {
			req := mkReq()
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			checkTransfer(t, handler.pip.TransCtx.Transfer)
		})

		t.Run("Recovery", func(t *testing.T) {
			const transferID uint32 = 4

			oldTransfer := model.Transfer{
				RemoteTransferID: utils.FormatUint(transferID),
				RuleID:           ctx.ServerRulePull.ID,
				LocalAccountID:   utils.NewNullInt64(ctx.LocalAccount.ID),
				SrcFilename:      fileName,
				DestFilename:     fileName,
				LocalPath:        dstFilePath,
				Filesize:         fileSize,
				Start:            testStart,
				Status:           types.StatusPaused,
				Step:             types.StepPreTasks,
				Progress:         0,
				TaskNumber:       0,
			}
			require.NoError(t, db.Insert(&oldTransfer).Run())

			req := mkReq()
			req.SetRecoveryPoint(1)

			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			checkTransfer(t, handler.pip.TransCtx.Transfer)
		})
	})
}

func TestServerOpenFile(t *testing.T) {
	t.SkipNow()

	db := dbtest.TestDatabase(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	testStart := time.Now()

	getHandler := func() *transferHandler {
		return &transferHandler{
			db:      db,
			logger:  gwtesting.Logger(t),
			agent:   ctx.Server,
			account: ctx.LocalAccount,
			conf: &ServerConfig{
				CheckPointConfig: CheckPointConfig{
					DisableRestart:     false,
					DisableCheckpoints: false,
					CheckpointSize:     65535,
					CheckpointWindow:   1,
				},
			},
			ctx:    context.Background(),
			cancel: func(cause error) {},
		}
	}

	t.Run("Push transfer", func(t *testing.T) {
		const (
			fileName    = "test_push_src.txt"
			fileContent = "Hello push"
		)

		srcFilePath := fs.JoinPath(conf.GlobalConfig.Paths.GatewayHome,
			ctx.Server.RootDir, ctx.ServerRulePush.LocalDir, fileName)
		require.NoError(t, fs.WriteFullFile(srcFilePath, []byte(fileContent)))

		var (
			fileSize         = int64(len(fileContent))
			reservationSpace = uint32(fileSize / bytesPerKB)
			filePath         = path.Join(ctx.ServerRulePush.Path, fileName)
		)

		mkReq := func(transferID uint32) *pesit.ServerTransfer {
			ct := pesit.NewTransfer(pesit.MethodRecv, filePath)
			ct.SetReservationSpace(reservationSpace, pesit.UnitKB)

			buf := &testhelpers.Buffer{}
			req := pesit.ServerTransferTest(ct, buf)

			return &req
		}

		checkFile := func(tb testing.TB, handler *transferHandler) {
			tb.Helper()
			assert.NotNil(tb, handler.file)
			assert.Equal(t, types.StepData, handler.pip.TransCtx.Transfer.Step)
		}

		t.Run("New transfer", func(t *testing.T) {
			const transferID uint32 = 1

			req := mkReq(transferID)
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			checkFile(t, handler)
		})

		t.Run("Recovery", func(t *testing.T) {
			const transferID uint32 = 2

			oldTransfer := model.Transfer{
				RemoteTransferID: utils.FormatUint(transferID),
				RuleID:           ctx.ServerRulePush.ID,
				LocalAccountID:   utils.NewNullInt64(ctx.LocalAccount.ID),
				SrcFilename:      fileName,
				LocalPath:        srcFilePath,
				Filesize:         int64(len(fileContent)),
				Start:            testStart,
				Status:           types.StatusPaused,
				Step:             types.StepPreTasks,
				Progress:         0,
				TaskNumber:       0,
			}
			require.NoError(t, db.Insert(&oldTransfer).Run())

			req := mkReq(transferID)
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			checkFile(t, handler)
		})
	})

	t.Run("Pull transfer", func(t *testing.T) {
		const (
			fileName       = "test_pull_dst.txt"
			fileSize int64 = 1234
		)

		var (
			reservationSpace = uint32(fileSize / bytesPerKB)
			filePath         = path.Join(ctx.ServerRulePull.Path, fileName)
			dstFilePath      = fs.JoinPath(conf.GlobalConfig.Paths.GatewayHome,
				ctx.Server.RootDir, ctx.ServerRulePull.LocalDir, fileName)
		)

		mkReq := func() *pesit.ServerTransfer {
			ct := pesit.NewTransfer(pesit.MethodRecv, filePath)
			ct.SetReservationSpace(reservationSpace, pesit.UnitKB)

			buf := &testhelpers.Buffer{}
			req := pesit.ServerTransferTest(ct, buf)

			return &req
		}

		checkFile := func(tb testing.TB, handler *transferHandler) {
			tb.Helper()
			assert.NotNil(tb, handler.file)
			assert.Equal(t, types.StepData, handler.pip.TransCtx.Transfer.Step)
		}

		t.Run("New transfer", func(t *testing.T) {
			req := mkReq()
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			checkFile(t, handler)
		})

		t.Run("Recovery", func(t *testing.T) {
			const transferID uint32 = 4

			oldTransfer := model.Transfer{
				RemoteTransferID: utils.FormatUint(transferID),
				RuleID:           ctx.ServerRulePull.ID,
				LocalAccountID:   utils.NewNullInt64(ctx.LocalAccount.ID),
				SrcFilename:      fileName,
				DestFilename:     fileName,
				LocalPath:        dstFilePath,
				Filesize:         fileSize,
				Start:            testStart,
				Status:           types.StatusPaused,
				Step:             types.StepPreTasks,
				Progress:         0,
				TaskNumber:       0,
			}
			require.NoError(t, db.Insert(&oldTransfer).Run())

			req := mkReq()
			req.SetRecoveryPoint(1)

			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			checkFile(t, handler)
		})
	})
}

func TestServerStartData(t *testing.T) {
	t.SkipNow()

	db := dbtest.TestDatabase(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	testStart := time.Now()

	getHandler := func() *transferHandler {
		return &transferHandler{
			db:      db,
			logger:  gwtesting.Logger(t),
			agent:   ctx.Server,
			account: ctx.LocalAccount,
			conf: &ServerConfig{
				CheckPointConfig: CheckPointConfig{
					DisableRestart:     false,
					DisableCheckpoints: false,
					CheckpointSize:     65535,
					CheckpointWindow:   1,
				},
			},
			ctx:    context.Background(),
			cancel: func(cause error) {},
		}
	}

	t.Run("Push transfer", func(t *testing.T) {
		const (
			fileName    = "test_push_src.txt"
			fileContent = "Hello push"
		)

		srcFilePath := fs.JoinPath(conf.GlobalConfig.Paths.GatewayHome,
			ctx.Server.RootDir, ctx.ServerRulePush.LocalDir, fileName)
		require.NoError(t, fs.WriteFullFile(srcFilePath, []byte(fileContent)))

		var (
			fileSize         = int64(len(fileContent))
			reservationSpace = uint32(fileSize / bytesPerKB)
			filePath         = path.Join(ctx.ServerRulePush.Path, fileName)
			checkpointSize   = fileSize / 2
		)

		mkReq := func(transferID uint32) *pesit.ServerTransfer {
			ct := pesit.NewTransfer(pesit.MethodRecv, filePath)
			ct.SetReservationSpace(reservationSpace, pesit.UnitKB)

			buf := &testhelpers.Buffer{}
			req := pesit.ServerTransferTest(ct, buf)

			return &req
		}

		checkRecovery := func(tb testing.TB, handler *transferHandler, expectedOff int64) {
			tb.Helper()

			off, err := handler.file.Seek(0, io.SeekCurrent)
			require.NoError(tb, err)
			assert.Equal(tb, expectedOff, off)
		}

		t.Run("New transfer", func(t *testing.T) {
			const transferID uint32 = 1

			req := mkReq(transferID)
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			require.NoError(t, handler.StartDataTransfer(req))
			checkRecovery(t, handler, 0)
		})

		t.Run("Recovery", func(t *testing.T) {
			const transferID uint32 = 2

			oldTransfer := model.Transfer{
				RemoteTransferID: utils.FormatUint(transferID),
				RuleID:           ctx.ServerRulePush.ID,
				LocalAccountID:   utils.NewNullInt64(ctx.LocalAccount.ID),
				SrcFilename:      fileName,
				LocalPath:        srcFilePath,
				Filesize:         fileSize,
				Start:            testStart,
				Status:           types.StatusPaused,
				Step:             types.StepPreTasks,
				Progress:         checkpointSize,
				TaskNumber:       0,
			}
			require.NoError(t, db.Insert(&oldTransfer).Run())

			req := mkReq(transferID)
			req.SetRecoveryPoint(1)
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			require.NoError(t, handler.StartDataTransfer(req))
			checkRecovery(t, handler, checkpointSize)
		})
	})

	t.Run("Pull transfer", func(t *testing.T) {
		const (
			fileName       = "test_pull_dst.txt"
			fileSize int64 = 1234
		)

		var (
			reservationSpace = uint32(fileSize / bytesPerKB)
			filePath         = path.Join(ctx.ServerRulePull.Path, fileName)
			dstFilePath      = fs.JoinPath(conf.GlobalConfig.Paths.GatewayHome,
				ctx.Server.RootDir, ctx.ServerRulePull.LocalDir, fileName)
			checkpointSize = fileSize / 2
		)

		mkReq := func() *pesit.ServerTransfer {
			ct := pesit.NewTransfer(pesit.MethodRecv, filePath)
			ct.SetReservationSpace(reservationSpace, pesit.UnitKB)

			buf := &testhelpers.Buffer{}
			req := pesit.ServerTransferTest(ct, buf)

			return &req
		}

		checkRecovery := func(tb testing.TB, req *pesit.ServerTransfer, expectedChkpt uint32) {
			tb.Helper()
			assert.Equal(tb, expectedChkpt, req.RecoveryPoint())
		}

		t.Run("New transfer", func(t *testing.T) {
			req := mkReq()
			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			require.NoError(t, handler.StartDataTransfer(req))
			checkRecovery(t, req, 0)
		})

		t.Run("Recovery", func(t *testing.T) {
			const transferID uint32 = 4

			oldTransfer := model.Transfer{
				RemoteTransferID: utils.FormatUint(transferID),
				RuleID:           ctx.ServerRulePull.ID,
				LocalAccountID:   utils.NewNullInt64(ctx.LocalAccount.ID),
				SrcFilename:      fileName,
				DestFilename:     fileName,
				LocalPath:        dstFilePath,
				Filesize:         fileSize,
				Start:            testStart,
				Status:           types.StatusPaused,
				Step:             types.StepPreTasks,
				Progress:         checkpointSize,
				TaskNumber:       0,
			}
			require.NoError(t, db.Insert(&oldTransfer).Run())

			req := mkReq()
			req.SetRecoveryPoint(1)

			handler := getHandler()

			require.NoError(t, handler.SelectFile(req))
			require.NoError(t, handler.OpenFile(req))
			require.NoError(t, handler.StartDataTransfer(req))
			checkRecovery(t, req, 1)
		})
	})
}
