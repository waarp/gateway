package pesit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

// The records of the text file sent by the delimited-articles tests: very
// different lengths, an empty one, and no separator after the last one.
var delimitedRecords = []string{ //nolint:gochecknoglobals // test data
	"first record", "", "the third record is much longer than the others", "last",
}

// delimitedFile overwrites the test file with the delimited records and returns
// the number of bytes to expect on the wire and the length of each record.
func delimitedFile(t *testing.T, ctx *gwtesting.TransferCtx, name string) (int64, []uint16) {
	t.Helper()

	paths := &ctx.DB.Config.Paths
	path := filepath.Join(paths.GatewayHome, paths.DefaultOutDir, name)
	require.NoError(t, os.WriteFile(path, []byte(strings.Join(delimitedRecords, "\n")), 0o600))

	var (
		lengths []uint16
		total   int64
	)

	for _, record := range delimitedRecords {
		lengths = append(lengths, uint16(len(record)))
		total += int64(len(record))
	}

	return total, lengths
}

func TestDelimitedArticlesPush(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	total, lengths := delimitedFile(t, ctx, filepath.Join("push_src_dir", "push.file"))

	ctx.TransferPush.TransferInfo[articlesSeparatorKey] = "LF"
	pip := ctx.PushPipeline(t)

	require.NoError(t, pip.Run(), "the transfer should execute without error")

	var serverTrans model.HistoryEntry
	require.NoError(t, db.Get(&serverTrans,
		"is_server=true AND is_send=? AND agent=? AND account=?",
		ctx.ServerRulePush.IsSend, ctx.Server.Name, ctx.LocalAccount.Login).Eager().Run())

	assert.Equal(t, types.StatusDone, serverTrans.Status)
	assert.Equal(t, total, serverTrans.Progress, "the separators are not sent")
	gwtesting.JSONEqual(t, lengths, serverTrans.TransferInfo[articlesLengthsKey])
}

func TestDelimitedArticlesPull(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	total, lengths := delimitedFile(t, ctx, filepath.Join("pull_src_dir", "pull.file"))

	serverTransfer := &model.Transfer{
		RuleID:         ctx.ServerRulePull.ID,
		LocalAccountID: ctx.LocalAccount.NullableID(),
		SrcFilename:    ctx.TransferPull.SrcFilename,
		Start:          time.Now().Add(time.Hour),
		Status:         types.StatusAvailable,
		TransferInfo:   map[string]any{articlesSeparatorKey: "LF"},
	}
	require.NoError(t, db.Insert(serverTransfer).Run())

	pip := ctx.PullPipeline(t)

	require.NoError(t, pip.Run(), "the transfer should execute without error")

	var clientTrans model.HistoryEntry
	require.NoError(t, db.Get(&clientTrans, "id=?", ctx.TransferPull.ID).Eager().Run())

	assert.Equal(t, types.StatusDone, clientTrans.Status)
	assert.Equal(t, total, clientTrans.Progress, "the separators are not sent")
	gwtesting.JSONEqual(t, lengths, clientTrans.TransferInfo[articlesLengthsKey])
}

func TestDelimitedArticlesTooLong(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, Pesit, nil, nil, nil)
	delimitedFile(t, ctx, filepath.Join("push_src_dir", "push.file"))

	// The configured length bounds the records: the third one is longer.
	ctx.TransferPush.TransferInfo[articlesSeparatorKey] = "LF"
	ctx.TransferPush.TransferInfo[articlesLengthsKey] = []uint16{20}
	pip := ctx.PushPipeline(t)

	require.Error(t, pip.Run(), "the transfer should fail")

	var clientTrans model.Transfer
	require.NoError(t, db.Get(&clientTrans, "id=?", ctx.TransferPush.ID).Run())

	assert.Equal(t, types.StatusError, clientTrans.Status)
	assert.Contains(t, clientTrans.ErrDetails, "record longer than the article size")
}
