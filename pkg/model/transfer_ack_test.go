package model

import (
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// The names of the cases are the values the parameters accept, and nothing
// else (#873).
func TestParseAckCase(t *testing.T) {
	t.Parallel()

	for _, c := range AckCases() {
		parsed, err := ParseAckCase(string(c))
		require.NoError(t, err)
		assert.Equal(t, c, parsed)
	}

	_, err := ParseAckCase("later")
	require.ErrorIs(t, err, ErrUnknownAckCase)
	assert.ErrorContains(t, err, `"later" (accepted: expected, received, missed, sent, none)`)
	assert.Equal(t, "expected, received, missed, sent, none", AckCaseNames())
}

// Each case keeps the transfers it names, on the transfers table, on the
// history and on the normalized view alike, whatever set the keys (#873).
func TestAckCaseCondition(t *testing.T) {
	db := dbtest.TestDatabase(t)

	client := &Client{Name: "ack_client", Protocol: testProtocol}
	require.NoError(t, db.Insert(client).Run())

	partner := &RemoteAgent{Name: "ack_partner", Protocol: testProtocol, Address: types.Addr("localhost", 1)}
	require.NoError(t, db.Insert(partner).Run())

	account := &RemoteAccount{RemoteAgentID: partner.ID, Login: "ack_login"}
	require.NoError(t, db.Insert(account).Run())

	rule := &Rule{Name: "ack_rule", IsSend: true, Path: "ack"}
	require.NoError(t, db.Insert(rule).Run())

	type fixture struct {
		status types.TransferStatus
		info   map[string]any
	}

	fixtures := map[int64]fixture{}

	transfer := func(name string, status types.TransferStatus, info map[string]any) int64 {
		trans := &Transfer{
			RuleID:          rule.ID,
			ClientID:        client.NullableID(),
			RemoteAccountID: account.NullableID(),
			SrcFilename:     name,
			DestFilename:    name,
			Status:          status,
			TransferInfo:    info,
		}
		require.NoError(t, db.Insert(trans).Run(), name)
		fixtures[trans.ID] = fixture{status, info}

		return trans.ID
	}

	history := func(id int64, status types.TransferStatus, info map[string]any) int64 {
		hist := &HistoryEntry{
			Identifier:       ID(id),
			RemoteTransferID: "hist",
			IsSend:           true,
			Rule:             rule.Name,
			Client:           client.Name,
			Account:          account.Login,
			Agent:            partner.Name,
			Protocol:         testProtocol,
			SrcFilename:      "hist",
			DestFilename:     "hist",
			LocalPath:        "hist",
			RemotePath:       "hist",
			Start:            time.Date(2026, 11, 11, 10, 0, 0, 0, time.UTC),
			Stop:             time.Date(2026, 11, 11, 10, 1, 0, 0, time.UTC),
			Status:           status,
			TransferInfo:     info,
		}
		require.NoError(t, db.Insert(hist).Run())
		fixtures[hist.ID] = fixture{status, info}

		return hist.ID
	}

	// Running transfers: the keys set by the partner option (at the
	// connection) or by a SETINFO pre-task read the same.
	waiting := transfer("waiting", types.StatusRunning, map[string]any{InfoAckExpected: true})
	waitingSince := transfer("waiting_since", types.StatusRunning, map[string]any{
		InfoAckExpected: true, InfoAckWaitTimeout: "5m", InfoAckWaitSince: time.Now(),
	})
	interrupted := transfer("interrupted", types.StatusInterrupted, map[string]any{InfoAckExpected: true})
	failed := transfer("failed", types.StatusError, map[string]any{InfoAckExpected: true, InfoAckWaitSince: time.Now()})
	plain := transfer("plain", types.StatusRunning, nil)
	declined := transfer("declined", types.StatusPlanned, map[string]any{InfoAckExpected: false})

	// Finished transfers, in the history.
	received := history(101, types.StatusDone, map[string]any{InfoAckReceived: true, InfoAckReceivedOn: time.Now()})
	cancelled := history(102, types.StatusCancelled, map[string]any{InfoAckExpected: true})
	sent := history(103, types.StatusDone, map[string]any{
		InfoAckSent: true, InfoAckSentTo: partner.Name, InfoAckSentAs: account.Login,
	})
	done := history(104, types.StatusDone, nil)

	type expectation struct {
		transfers, history []int64
	}

	cases := map[AckCase]expectation{
		AckExpected: {transfers: []int64{waiting, waitingSince, interrupted}},
		AckReceived: {history: []int64{received}},
		AckMissed:   {transfers: []int64{failed}, history: []int64{cancelled}},
		AckSent:     {history: []int64{sent}},
		AckNone:     {transfers: []int64{plain, declined}, history: []int64{done}},
	}

	for ackCase, want := range cases {
		cond, args := ackCase.Condition()

		var transfers Transfers
		require.NoError(t, db.Select(&transfers).Where(cond, args...).Run(), ackCase)
		assert.ElementsMatch(t, want.transfers, transferIDs(transfers), "%s on the transfers table", ackCase)

		var entries HistoryEntries
		require.NoError(t, db.Select(&entries).Where(cond, args...).Run(), ackCase)
		assert.ElementsMatch(t, want.history, historyIDs(entries), "%s on the history", ackCase)

		var view NormalizedTransfers
		require.NoError(t, db.Select(&view).Where(cond, args...).Run(), ackCase)
		assert.ElementsMatch(t, append(append([]int64{}, want.transfers...), want.history...),
			viewIDs(view), "%s on the normalized view", ackCase)
	}

	// Every transfer is in a case, and in one case only among expected,
	// received, missed and none; "sent" may add to one of them on a relay.
	var all Transfers
	require.NoError(t, db.Select(&all).Run())

	var allHist HistoryEntries
	require.NoError(t, db.Select(&allHist).Run())

	covered, exclusive := map[int64]bool{}, map[int64]int{}

	for ackCase, want := range cases {
		for _, id := range slices.Concat(want.transfers, want.history) {
			covered[id] = true

			if ackCase != AckSent {
				exclusive[id]++
			}
		}
	}

	for _, id := range slices.Concat(transferIDs(all), historyIDs(allHist)) {
		assert.Truef(t, covered[id], "transfer %d is in no case", id)
		assert.LessOrEqualf(t, exclusive[id], 1, "transfer %d is in several exclusive cases", id)
	}

	// The rules in Go (AckOf, AckWasSent: the lists read the keys of a page)
	// agree with the SQL on every fixture.
	for id, f := range fixtures {
		want := AckNone

		for _, ackCase := range []AckCase{AckExpected, AckReceived, AckMissed} {
			if slices.Contains(cases[ackCase].transfers, id) || slices.Contains(cases[ackCase].history, id) {
				want = ackCase
			}
		}

		sent := slices.Contains(cases[AckSent].transfers, id) || slices.Contains(cases[AckSent].history, id)
		if want == AckNone && sent {
			want = AckSent
		}

		assert.Equalf(t, want, AckOf(f.status, f.info), "AckOf on %d", id)
		assert.Equalf(t, sent, AckWasSent(f.info), "AckWasSent on %d", id)
	}

	// An unknown case keeps nothing.
	cond, args := AckCase("later").Condition()

	var none NormalizedTransfers
	require.NoError(t, db.Select(&none).Where(cond, args...).Run())
	assert.Empty(t, none)
}

func transferIDs(transfers Transfers) []int64 {
	ids := make([]int64, 0, len(transfers))
	for _, trans := range transfers {
		ids = append(ids, trans.ID)
	}

	return ids
}

func historyIDs(entries HistoryEntries) []int64 {
	ids := make([]int64, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}

	return ids
}

func viewIDs(view NormalizedTransfers) []int64 {
	ids := make([]int64, 0, len(view))
	for _, trans := range view {
		ids = append(ids, trans.ID)
	}

	return ids
}
