package batch

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"

	// The protocols of the gateway, for the fixtures.
	_ "code.waarp.fr/apps/gateway/gateway/pkg/protocols"
)

// fixtures: a partner with its account and a client, a server with its
// account, a sending and a receiving rule; live transfers in every state the
// gestures tell apart, three history entries (one whose rule is gone).
type fixtures struct {
	db *database.DB

	send, receive *model.Rule
	client        *model.Client
	prod          *model.RemoteAccount
	south         *model.LocalAccount

	failed, child, paused, running, planned, available *model.Transfer
	done, cancelled, orphan                            *model.HistoryEntry
}

func at(hour int) time.Time { return time.Date(2026, 10, 21, hour, 0, 0, 0, time.Local) }

func newFixtures(t *testing.T) *fixtures {
	t.Helper()

	f := &fixtures{db: dbtest.TestDatabase(t)}

	partner := &model.RemoteAgent{Name: "banque", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)}
	require.NoError(t, f.db.Insert(partner).Run())

	f.prod = &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "prod"}
	require.NoError(t, f.db.Insert(f.prod).Run())

	f.client = &model.Client{Name: "cli", Protocol: "sftp"}
	require.NoError(t, f.db.Insert(f.client).Run())

	server := &model.LocalAgent{Name: "srv", Protocol: "sftp", Address: types.Addr("0.0.0.0", 2222)}
	require.NoError(t, f.db.Insert(server).Run())

	f.south = &model.LocalAccount{LocalAgentID: server.ID, Login: "sud"}
	require.NoError(t, f.db.Insert(f.south).Run())

	f.send = &model.Rule{Name: "releves", IsSend: true, Path: "/releves"}
	f.receive = &model.Rule{Name: "commandes", IsSend: false, Path: "/commandes"}
	require.NoError(t, f.db.Insert(f.send).Run())
	require.NoError(t, f.db.Insert(f.receive).Run())

	f.failed = f.clientTransfer(t, "releve-1.pdf", types.StatusError,
		map[string]any{model.FollowID: int64(42), "lot": "2026-10-20-B"})
	f.child = f.clientTransfer(t, "releve-2.pdf", types.StatusError,
		map[string]any{model.SyncTransferID: f.failed.ID, model.SyncTransferRank: int64(1)})
	f.paused = f.clientTransfer(t, "releve-3.pdf", types.StatusPaused, nil)
	f.running = f.clientTransfer(t, "releve-4.pdf", types.StatusRunning, nil)
	f.planned = f.clientTransfer(t, "releve-5.pdf", types.StatusPlanned, nil)

	f.available = &model.Transfer{
		RuleID: f.receive.ID, LocalAccountID: f.south.NullableID(), DestFilename: "inventaire.csv",
		Status: types.StatusAvailable, Start: at(23),
	}
	require.NoError(t, f.db.Insert(f.available).Run())

	f.done = f.history(t, 1001, "releves", types.StatusDone)
	f.cancelled = f.history(t, 1002, "releves", types.StatusCancelled)
	f.orphan = f.history(t, 1003, "vanished", types.StatusDone)

	return f
}

func (f *fixtures) clientTransfer(t *testing.T, file string, status types.TransferStatus, info map[string]any,
) *model.Transfer {
	t.Helper()

	trans := &model.Transfer{
		RuleID: f.send.ID, ClientID: f.client.NullableID(), RemoteAccountID: f.prod.NullableID(),
		SrcFilename: file, DestFilename: file, Status: status, Start: at(10), Filesize: model.UnknownSize,
		TransferInfo: info,
	}
	if status == types.StatusError {
		trans.ErrCode, trans.ErrDetails, trans.Step = types.TeDataTransfer, "connection lost", types.StepData
	}

	require.NoError(t, f.db.Insert(trans).Run())

	return trans
}

func (f *fixtures) history(t *testing.T, id int64, rule string, status types.TransferStatus) *model.HistoryEntry {
	t.Helper()

	entry := &model.HistoryEntry{
		Identifier: model.ID(id), RemoteTransferID: "r-" + strconv.FormatInt(id, 10), IsSend: true, Rule: rule,
		Client: "cli", Account: "prod", Agent: "banque", Protocol: "sftp",
		SrcFilename: "old-" + strconv.FormatInt(id, 10) + ".pdf", DestFilename: "old-" + strconv.FormatInt(id, 10) + ".pdf",
		Filesize: 1024, Progress: 1024, Start: at(8), Stop: at(9), Status: status,
	}
	require.NoError(t, f.db.Insert(entry).Run())

	return entry
}

func (f *fixtures) all() []int64 {
	return []int64{
		f.failed.ID, f.child.ID, f.paused.ID, f.running.ID, f.planned.ID, f.available.ID,
		f.done.ID, f.cancelled.ID, f.orphan.ID, 9999,
	}
}

func (f *fixtures) status(t *testing.T, id int64) types.TransferStatus {
	t.Helper()

	var view model.NormalizedTransferView
	require.NoError(t, f.db.Get(&view, "id=?", id).Run())

	return view.Status
}

func (f *fixtures) live(t *testing.T, id int64) bool {
	t.Helper()

	var view model.NormalizedTransferView
	require.NoError(t, f.db.Get(&view, "id=?", id).Run())

	return view.IsTransfer
}

// TestPreviewOf checks the count of what each gesture would do on the set of
// every fixture and an unknown identifier.
func TestPreviewOf(t *testing.T) {
	t.Parallel()

	f := newFixtures(t)

	resume, err := PreviewOf(f.db, Resume, f.all())
	require.NoError(t, err)
	assert.Equal(t, &Preview{Accepted: 3, Running: 0, Refused: map[Reason]int{ReasonState: 5, ReasonServer: 1, ReasonMissing: 1}},
		resume, "the failed, the child (its information unread) and the paused; the done, cancelled, running, planned refused")
	assert.Equal(t, 10, resume.Total())

	retry, err := PreviewOf(f.db, Retry, f.all())
	require.NoError(t, err)
	assert.Equal(t, &Preview{Accepted: 6, Refused: map[Reason]int{ReasonState: 2, ReasonServer: 1, ReasonMissing: 1}}, retry,
		"the stopped client transfers, live or in the history; the running and the planned refused")

	cancel, err := PreviewOf(f.db, Cancel, f.all())
	require.NoError(t, err)
	assert.Equal(t, &Preview{Accepted: 6, Running: 1, Refused: map[Reason]int{ReasonState: 3, ReasonMissing: 1}}, cancel,
		"every live transfer, one of them running; the history refused")
}

// TestResumeSet checks the resumption of a set: exactly the resumable
// transfers, the others counted by reason, in one transaction.
func TestResumeSet(t *testing.T) {
	t.Parallel()

	f := newFixtures(t)
	when := time.Now().Truncate(time.Second)

	result, err := ResumeSet(f.db, append(f.all(), f.failed.ID), when)
	require.NoError(t, err)
	assert.Equal(t, []int64{f.failed.ID, f.paused.ID}, result.Done, "the identifiers deduplicated, in order")
	assert.Equal(t, map[Reason]int{ReasonSync: 1, ReasonState: 5, ReasonServer: 1, ReasonMissing: 1}, result.Counts())
	assert.Contains(t, result.Refused, Refusal{ID: f.child.ID, Reason: ReasonSync})
	assert.Contains(t, result.Refused, Refusal{ID: f.available.ID, Reason: ReasonServer})
	assert.Contains(t, result.Refused, Refusal{ID: 9999, Reason: ReasonMissing})

	for _, id := range []int64{f.failed.ID, f.paused.ID} {
		var trans model.Transfer
		require.NoError(t, f.db.Get(&trans, "id=?", id).Run())
		assert.Equal(t, types.StatusPlanned, trans.Status)
		assert.Equal(t, types.TeOk, trans.ErrCode)
		assert.Empty(t, trans.ErrDetails)
		assert.True(t, trans.NextRetry.Equal(when))
	}

	assert.Equal(t, types.StatusError, f.status(t, f.child.ID), "the child of a synchronous transfer is intact")
	assert.Equal(t, types.StatusRunning, f.status(t, f.running.ID))
	assert.Equal(t, types.StatusAvailable, f.status(t, f.available.ID))

	t.Run("in the transaction it is given", func(t *testing.T) {
		failed := f.clientTransfer(t, "tx.pdf", types.StatusError, nil)
		boom := errors.New("boom")

		err := f.db.Transaction(func(ses *database.Session) error {
			res, err := ResumeSet(ses, []int64{failed.ID, f.child.ID}, when)
			require.NoError(t, err)
			require.Equal(t, []int64{failed.ID}, res.Done)
			require.Len(t, res.Refused, 1, "the child is refused")

			return boom
		})
		require.ErrorIs(t, err, boom)
		assert.Equal(t, types.StatusError, f.status(t, failed.ID), "rolled back with the transaction")
	})
}

// TestResumeManyChunks checks that a set larger than a chunk is resumed whole.
func TestResumeManyChunks(t *testing.T) {
	t.Parallel()

	f := newFixtures(t)
	set := make([]int64, 0, chunkSize+100)

	for i := range cap(set) {
		set = append(set, f.clientTransfer(t, "many-"+strconv.Itoa(i)+".pdf", types.StatusError, nil).ID)
	}

	result, err := ResumeSet(f.db, set, time.Now())
	require.NoError(t, err)
	assert.Len(t, result.Done, cap(set))
	assert.Empty(t, result.Refused)

	count, err := f.db.Count(&model.Transfer{}).Where("status=?", types.StatusError).Run()
	require.NoError(t, err)
	assert.EqualValues(t, 2, count, "the failed and the child of the fixtures alone")
}

// TestRetrySet checks the retry of a set: a new transfer for each stopped
// client transfer, live or in the history, at the date; the others counted.
func TestRetrySet(t *testing.T) {
	t.Parallel()

	f := newFixtures(t)
	date := at(18)

	result, err := RetrySet(f.db, f.all(), date)
	require.NoError(t, err)
	assert.Equal(t, []int64{f.failed.ID, f.child.ID, f.paused.ID, f.done.ID, f.cancelled.ID}, result.Done)
	assert.Len(t, result.New, 5)
	assert.Equal(t, map[Reason]int{ReasonState: 2, ReasonServer: 1, ReasonGone: 1, ReasonMissing: 1}, result.Counts())
	assert.Contains(t, result.Refused, Refusal{ID: f.orphan.ID, Reason: ReasonGone})

	for i, id := range result.New {
		var trans model.Transfer
		require.NoError(t, f.db.Get(&trans, "id=?", id).Run())
		assert.Equal(t, types.StatusPlanned, trans.Status)
		assert.True(t, trans.Start.Equal(date), "at the date")
		assert.Equal(t, f.send.ID, trans.RuleID)
		assert.Equal(t, f.prod.NullableID(), trans.RemoteAccountID)
		assert.NotEqual(t, result.Done[i], trans.ID, "a new transfer")
	}

	assert.Equal(t, types.StatusError, f.status(t, f.failed.ID), "the transfer of origin stays as it is")
	assert.Equal(t, types.StatusDone, f.status(t, f.done.ID))

	t.Run("in the transaction it is given", func(t *testing.T) {
		boom := errors.New("boom")
		err := f.db.Transaction(func(ses *database.Session) error {
			res, err := RetrySet(ses, []int64{f.done.ID}, date)
			require.NoError(t, err)
			require.Len(t, res.New, 1)

			return boom
		})
		require.ErrorIs(t, err, boom)

		count, err := f.db.Count(&model.Transfer{}).Where("status=?", types.StatusPlanned).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 6, count, "the planned fixture and the five retries alone: the sixth rolled back")
	})
}

// TestCancelSet checks the cancellation of a set: the live transfers go to
// the history, cancelled, with their information; a running one without its
// pipeline is cancelled in the base on a single node and left to another node
// in a cluster; the history and the unknown are counted.
func TestCancelSet(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)

	t.Run("on a single node", func(t *testing.T) {
		t.Parallel()

		f := newFixtures(t)
		set := []int64{f.failed.ID, f.running.ID, f.available.ID, f.done.ID, 9999}

		result, err := CancelSet(context.Background(), f.db, logger, set)
		require.NoError(t, err)
		assert.Equal(t, []int64{f.failed.ID, f.running.ID, f.available.ID}, result.Done)
		assert.Equal(t, map[Reason]int{ReasonState: 1, ReasonMissing: 1}, result.Counts())

		for _, id := range result.Done {
			assert.False(t, f.live(t, id), "moved to the history")
			assert.Equal(t, types.StatusCancelled, f.status(t, id))
		}

		var entry model.HistoryEntry
		require.NoError(t, f.db.Get(&entry, "id=?", f.failed.ID).Eager().Run())
		assert.Equal(t, "2026-10-20-B", entry.TransferInfo["lot"], "the information follows the transfer")
		assert.Equal(t, "releves", entry.Rule)

		count, err := f.db.Count(&model.TransferInfo{}).Where("transfer_id=?", f.failed.ID).Run()
		require.NoError(t, err)
		assert.Zero(t, count, "no information left on the live side")

		assert.Equal(t, types.StatusPaused, f.status(t, f.paused.ID), "a transfer out of the set is intact")
		assert.Equal(t, types.StatusDone, f.status(t, f.done.ID))
	})

	t.Run("in a cluster", func(t *testing.T) {
		t.Parallel()

		f := newFixtures(t)
		f.db.Config.NodeID = "gw-2"

		result, err := CancelSet(context.Background(), f.db, logger, []int64{f.failed.ID, f.running.ID})
		require.NoError(t, err)
		assert.Equal(t, []int64{f.failed.ID}, result.Done)
		assert.Equal(t, []Refusal{{ID: f.running.ID, Reason: ReasonElsewhere}}, result.Refused)
		assert.Equal(t, types.StatusRunning, f.status(t, f.running.ID), "left to the node that runs it")
	})
}

// TestReasons checks that every reason is reported, once.
func TestReasons(t *testing.T) {
	t.Parallel()

	reasons := Reasons()
	assert.Len(t, reasons, 7)

	seen := map[Reason]bool{}
	for _, reason := range reasons {
		assert.False(t, seen[reason], reason)
		seen[reason] = true
	}
}
