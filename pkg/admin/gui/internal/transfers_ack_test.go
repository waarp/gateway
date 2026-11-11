package internal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// The list of the web interface filters by acknowledgement with the cases of
// the model, the ones the REST API and the command line share (#873).
func TestTransferListAcknowledgement(t *testing.T) {
	db := dbtest.TestDatabase(t)

	entry := func(id int64, status types.TransferStatus, info map[string]any) *model.HistoryEntry {
		hist := &model.HistoryEntry{
			Identifier:       model.ID(id),
			RemoteTransferID: "ack",
			IsSend:           true,
			Rule:             "send",
			Account:          "foo",
			Agent:            "bar",
			Client:           "baz",
			Protocol:         "http",
			SrcFilename:      "/test/file.txt",
			DestFilename:     "/test/file.txt",
			LocalPath:        "/full/local/path",
			RemotePath:       "/full/remote/path",
			Start:            time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC),
			Stop:             time.Date(2026, 11, 11, 10, 1, 0, 0, time.UTC),
			Status:           status,
			TransferInfo:     info,
		}
		require.NoError(t, db.Insert(hist).Run())

		return hist
	}

	received := entry(1, types.StatusDone, map[string]any{model.InfoAckReceived: true})
	missed := entry(2, types.StatusCancelled, map[string]any{model.InfoAckExpected: true})
	plain := entry(3, types.StatusDone, nil)

	ids := func(ackCase model.AckCase) []int64 {
		transfers, err := StartTransferQuery(db, "id", true).Acknowledgement(ackCase).Run()
		require.NoError(t, err)

		ids := make([]int64, 0, len(transfers))
		for _, trans := range transfers {
			ids = append(ids, trans.ID)
		}

		return ids
	}

	assert.Equal(t, []int64{received.ID}, ids(model.AckReceived))
	assert.Equal(t, []int64{missed.ID}, ids(model.AckMissed))
	assert.Equal(t, []int64{plain.ID}, ids(model.AckNone))
	assert.Empty(t, ids(model.AckExpected))
	assert.Empty(t, ids(model.AckSent))
}
