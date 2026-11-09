package model

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// setFixtures inserts what a client transfer needs, and gives a failed one.
type setFixtures struct {
	rule    *Rule
	client  *Client
	account *RemoteAccount
}

func newSetFixtures(t *testing.T, db *database.DB) *setFixtures {
	t.Helper()

	f := &setFixtures{rule: &Rule{Name: "rule", IsSend: true}, client: &Client{Name: "client", Protocol: testProtocol}}
	require.NoError(t, db.Insert(f.rule).Run())
	require.NoError(t, db.Insert(f.client).Run())

	partner := &RemoteAgent{Name: "partner", Protocol: testProtocol, Address: types.Addr("localhost", 0)}
	require.NoError(t, db.Insert(partner).Run())

	f.account = &RemoteAccount{RemoteAgentID: partner.ID, Login: "toto"}
	require.NoError(t, db.Insert(f.account).Run())

	return f
}

func (f *setFixtures) failed(t *testing.T, db *database.DB, file string) *Transfer {
	t.Helper()

	trans := &Transfer{
		RuleID: f.rule.ID, ClientID: f.client.NullableID(), RemoteAccountID: f.account.NullableID(),
		SrcFilename: file, Status: types.StatusError, ErrCode: types.TeDataTransfer, ErrDetails: "connection lost",
		Step: types.StepData, Progress: 1024, RemainingTries: 2, NextRetryDelay: 30,
	}
	require.NoError(t, db.Insert(trans).Run())

	return trans
}

func transferOf(t *testing.T, db *database.DB, id int64) *Transfer {
	t.Helper()

	var trans Transfer
	require.NoError(t, db.Get(&trans, "id=?", id).Run())

	return &trans
}

// TestResumeTransfers checks that the resumption of a set leaves each
// transfer as the resumption of one does: the same columns, the same values,
// the others untouched; that it runs in the transaction it is given; and that
// it writes nothing without an identifier.
func TestResumeTransfers(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	f := newSetFixtures(t, db)

	t.Run("the same state as the resumption of one transfer", func(t *testing.T) {
		one := f.failed(t, db, "one.txt")
		set := f.failed(t, db, "set.txt")
		other := f.failed(t, db, "other.txt")
		when := time.Now().Truncate(time.Second)

		require.NoError(t, transferOf(t, db, one.ID).Resume(db, when))
		require.NoError(t, ResumeTransfers(db, []int64{set.ID}, when))

		a, b := transferOf(t, db, one.ID), transferOf(t, db, set.ID)
		assert.Equal(t, types.StatusPlanned, b.Status)
		assert.Equal(t, types.TeOk, b.ErrCode)
		assert.Empty(t, b.ErrDetails)
		assert.True(t, b.NextRetry.Equal(when), "the next attempt at when")
		assert.True(t, b.Stop.IsZero())

		// Everything else as the unit resumption leaves it (the identifiers
		// and the start, given at the insertion, apart).
		b.ID, b.SrcFilename, b.RemoteTransferID, b.Start = a.ID, a.SrcFilename, a.RemoteTransferID, a.Start
		assert.Equal(t, a, b)

		untouched := transferOf(t, db, other.ID)
		assert.Equal(t, types.StatusError, untouched.Status, "a transfer out of the set is intact")
		assert.Equal(t, "connection lost", untouched.ErrDetails)
	})

	t.Run("in the transaction it is given", func(t *testing.T) {
		trans := f.failed(t, db, "tx.txt")
		boom := errors.New("boom")

		err := db.Transaction(func(ses *database.Session) error {
			require.NoError(t, ResumeTransfers(ses, []int64{trans.ID}, time.Now()))

			return boom
		})
		require.ErrorIs(t, err, boom)
		assert.Equal(t, types.StatusError, transferOf(t, db, trans.ID).Status, "rolled back with the transaction")
	})

	t.Run("nothing without an identifier", func(t *testing.T) {
		trans := f.failed(t, db, "none.txt")
		require.NoError(t, ResumeTransfers(db, nil, time.Now()))
		assert.Equal(t, types.StatusError, transferOf(t, db, trans.ID).Status)
	})
}

// TestNormalizedTransferChecks checks the preconditions of the retry and of
// the cancellation, those the unit actions of the web interface apply.
func TestNormalizedTransferChecks(t *testing.T) {
	t.Parallel()

	view := func(status types.TransferStatus, server, live bool) *NormalizedTransferView {
		return &NormalizedTransferView{Status: status, IsServer: server, IsTransfer: live}
	}

	t.Run("retry", func(t *testing.T) {
		for _, status := range []types.TransferStatus{
			types.StatusPaused, types.StatusInterrupted, types.StatusError, types.StatusDone, types.StatusCancelled,
		} {
			require.NoError(t, view(status, false, true).CheckRetryable(), status)
		}

		for _, status := range []types.TransferStatus{types.StatusPlanned, types.StatusAvailable, types.StatusRunning} {
			require.ErrorIs(t, view(status, false, true).CheckRetryable(), ErrRetryRunning, status)
		}

		require.ErrorIs(t, view(types.StatusError, true, true).CheckRetryable(), ErrRetryServer)
	})

	t.Run("cancel", func(t *testing.T) {
		for _, status := range []types.TransferStatus{
			types.StatusPlanned, types.StatusAvailable, types.StatusRunning, types.StatusPaused,
			types.StatusInterrupted, types.StatusError,
		} {
			require.NoError(t, view(status, false, true).CheckCancellable(), status)
		}

		require.ErrorIs(t, view(types.StatusDone, false, false).CheckCancellable(), ErrCancelDone)
		require.ErrorIs(t, view(types.StatusCancelled, true, false).CheckCancellable(), ErrCancelDone)
	})
}
