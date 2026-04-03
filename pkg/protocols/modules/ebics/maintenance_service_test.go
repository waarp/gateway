package ebics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type fakeMaintenanceTicker struct {
	ch chan time.Time
}

func newFakeMaintenanceTicker() *fakeMaintenanceTicker {
	return &fakeMaintenanceTicker{ch: make(chan time.Time, 4)}
}

func (t *fakeMaintenanceTicker) Chan() <-chan time.Time { return t.ch }
func (t *fakeMaintenanceTicker) Stop()                  {}

func TestMaintenanceServiceRunMaintenancePurgesOnlySafeRows(t *testing.T) {
	db := dbtest.TestDatabase(t)
	service := NewMaintenanceService(db)

	host := insertTestEbicsHost(t, db, "HOST-MAINT")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-MAINT", "USER-MAINT", true)
	now := time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC)

	policy := model.DefaultEbicsRuntimePolicy()
	policy.TransactionRetentionSeconds = int64((24 * time.Hour) / time.Second)
	policy.RTNEventRetentionSeconds = int64((48 * time.Hour) / time.Second)
	require.NoError(t, db.Insert(policy).Run())

	require.NoError(t, db.Insert(&model.EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             "nonce-old",
		Timestamp:         now.Add(-2 * time.Minute),
		ExpiresAt:         now.Add(-time.Nanosecond),
	}).Run())
	require.NoError(t, db.Insert(&model.EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             "nonce-boundary",
		Timestamp:         now.Add(-time.Minute),
		ExpiresAt:         now,
	}).Run())

	require.NoError(t, db.Insert(&model.EbicsTransaction{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-OLD-COMPLETED",
		OrderType:         "BTU",
		Status:            model.EbicsTransactionStatusCompletedForRuntime(),
		Direction:         model.EbicsOperationDirectionInboundForRuntime(),
	}).Run())
	require.NoError(t, db.Insert(&model.EbicsTransaction{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-OLD-RUNNING",
		OrderType:         "BTU",
		Status:            "RUNNING",
		Direction:         model.EbicsOperationDirectionInboundForRuntime(),
	}).Run())
	require.NoError(t, db.Insert(&model.EbicsTransaction{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-NEW-COMPLETED",
		OrderType:         "BTU",
		Status:            model.EbicsTransactionStatusCompletedForRuntime(),
		Direction:         model.EbicsOperationDirectionInboundForRuntime(),
	}).Run())
	require.NoError(t, db.Exec(
		"UPDATE ebics_transactions SET updated_at=? WHERE transaction_id=?",
		now.Add(-25*time.Hour), "TX-OLD-COMPLETED",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_transactions SET updated_at=? WHERE transaction_id=?",
		now.Add(-25*time.Hour), "TX-OLD-RUNNING",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_transactions SET updated_at=? WHERE transaction_id=?",
		now.Add(-time.Hour), "TX-NEW-COMPLETED",
	))

	require.NoError(t, db.Insert(&model.EbicsRTNEvent{
		Source:         "wss",
		IdempotenceKey: "rtn-old-processed",
		Status:         "PROCESSED",
		ReceivedAt:     now.Add(-72 * time.Hour),
		ProcessedAt:    now.Add(-71 * time.Hour),
	}).Run())
	require.NoError(t, db.Insert(&model.EbicsRTNEvent{
		Source:         "wss",
		IdempotenceKey: "rtn-old-retryable",
		Status:         "RETRYABLE",
		ReceivedAt:     now.Add(-72 * time.Hour),
		NextRetryAt:    now.Add(-71 * time.Hour),
	}).Run())
	require.NoError(t, db.Insert(&model.EbicsRTNEvent{
		Source:         "wss",
		IdempotenceKey: "rtn-new-failed",
		Status:         "FAILED",
		ReceivedAt:     now.Add(-12 * time.Hour),
		LastError:      "bank timeout",
	}).Run())
	require.NoError(t, db.Exec(
		"UPDATE ebics_rtn_events SET updated_at=? WHERE idempotence_key=?",
		now.Add(-72*time.Hour), "rtn-old-processed",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_rtn_events SET updated_at=? WHERE idempotence_key=?",
		now.Add(-72*time.Hour), "rtn-old-retryable",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_rtn_events SET updated_at=? WHERE idempotence_key=?",
		now.Add(-12*time.Hour), "rtn-new-failed",
	))

	require.NoError(t, service.runMaintenanceAt(now, policy))

	var nonceCount uint64
	nonceCount, err := db.Count(&model.EbicsNonce{}).
		Where("nonce IN (?, ?)", "nonce-old", "nonce-boundary").
		Run()
	require.NoError(t, err)
	require.EqualValues(t, 1, nonceCount)

	var txCount uint64
	txCount, err = db.Count(&model.EbicsTransaction{}).
		Where("transaction_id IN (?, ?, ?)", "TX-OLD-COMPLETED", "TX-OLD-RUNNING", "TX-NEW-COMPLETED").
		Run()
	require.NoError(t, err)
	require.EqualValues(t, 2, txCount)

	var runningTx model.EbicsTransaction
	require.NoError(t, db.Get(&runningTx, "transaction_id=?", "TX-OLD-RUNNING").Run())
	assert.Equal(t, "RUNNING", runningTx.Status)

	var rtnCount uint64
	rtnCount, err = db.Count(&model.EbicsRTNEvent{}).
		Where("idempotence_key IN (?, ?, ?)", "rtn-old-processed", "rtn-old-retryable", "rtn-new-failed").
		Run()
	require.NoError(t, err)
	require.EqualValues(t, 2, rtnCount)
}

func TestMaintenanceServiceStartStop(t *testing.T) {
	db := dbtest.TestDatabase(t)
	service := NewMaintenanceService(db)
	ticker := newFakeMaintenanceTicker()
	service.newTicker = func(time.Duration) maintenanceTicker { return ticker }

	require.NoError(t, service.Start())
	code, reason := service.State()
	require.Equal(t, utils.StateRunning, code)
	require.Empty(t, reason)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, service.Stop(ctx))

	code, reason = service.State()
	require.Equal(t, utils.StateOffline, code)
	require.Empty(t, reason)
}
