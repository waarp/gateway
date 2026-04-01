package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestPurgeEbicsNoncesBeforeKeepsExactExpirationBoundary(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertNonceTestSubscriber(t, db, "HOST-RET-NONCE", "PARTNER-RET-NONCE", "USER-RET-NONCE")
	cutoff := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)

	require.NoError(t, db.Insert(&EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             "nonce-expired",
		Timestamp:         cutoff.Add(-2 * time.Minute),
		ExpiresAt:         cutoff.Add(-time.Nanosecond),
	}).Run())
	require.NoError(t, db.Insert(&EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             "nonce-boundary",
		Timestamp:         cutoff.Add(-time.Minute),
		ExpiresAt:         cutoff,
	}).Run())

	require.NoError(t, PurgeEbicsNoncesBefore(db, cutoff))

	count, err := db.Count(&EbicsNonce{}).Where("ebics_subscriber_id=?", subscriber.ID).Run()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	var kept EbicsNonce
	require.NoError(t, db.Get(&kept, "ebics_subscriber_id=? AND nonce=?", subscriber.ID, "nonce-boundary").Run())
}

func TestPurgeEbicsTransactionsBeforeKeepsNewerTransaction(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertNonceTestSubscriber(t, db, "HOST-RET-TX", "PARTNER-RET-TX", "USER-RET-TX")
	cutoff := time.Date(2026, 4, 1, 9, 15, 0, 0, time.UTC)

	require.NoError(t, db.Insert(&EbicsTransaction{
		EbicsHostID:       subscriber.EbicsHostID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-RET-OLD",
		OrderType:         "BTU",
		Status:            "RUNNING",
		Direction:         EbicsOperationDirectionInboundForRuntime(),
	}).Run())
	require.NoError(t, db.Insert(&EbicsTransaction{
		EbicsHostID:       subscriber.EbicsHostID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-RET-NEW",
		OrderType:         "BTU",
		Status:            "RUNNING",
		Direction:         EbicsOperationDirectionInboundForRuntime(),
	}).Run())
	require.NoError(t, db.Exec(
		"UPDATE ebics_transactions SET updated_at=? WHERE transaction_id=?",
		cutoff.Add(-time.Second), "TX-RET-OLD",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_transactions SET updated_at=? WHERE transaction_id=?",
		cutoff.Add(time.Minute), "TX-RET-NEW",
	))

	require.NoError(t, PurgeEbicsTransactionsBefore(db, cutoff))

	count, err := db.Count(&EbicsTransaction{}).
		Where("ebics_subscriber_id=? AND transaction_id IN (?, ?)", subscriber.ID, "TX-RET-OLD", "TX-RET-NEW").
		Run()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)

	var kept EbicsTransaction
	require.NoError(t, db.Get(&kept, "ebics_subscriber_id=? AND transaction_id=?", subscriber.ID, "TX-RET-NEW").Run())
}

func TestPurgeEbicsRTNEventsBeforePurgesOnlyTerminalStatuses(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertNonceTestSubscriber(t, db, "HOST-RET-RTN", "PARTNER-RET-RTN", "USER-RET-RTN")
	cutoff := time.Date(2026, 4, 1, 9, 30, 0, 0, time.UTC)

	require.NoError(t, db.Insert(&EbicsRTNEvent{
		Source:            "wss",
		IdempotenceKey:    "rtn-processed-old",
		EbicsHostID:       utils.NewNullInt64(subscriber.EbicsHostID),
		EbicsSubscriberID: utils.NewNullInt64(subscriber.ID),
		Status:            ebicsRTNEventStatusProcessed,
		ReceivedAt:        cutoff.Add(-2 * time.Minute),
		ProcessedAt:       cutoff.Add(-time.Minute),
		Attempts:          1,
	}).Run())
	require.NoError(t, db.Insert(&EbicsRTNEvent{
		Source:            "wss",
		IdempotenceKey:    "rtn-failed-old",
		EbicsHostID:       utils.NewNullInt64(subscriber.EbicsHostID),
		EbicsSubscriberID: utils.NewNullInt64(subscriber.ID),
		Status:            ebicsRTNEventStatusFailed,
		ReceivedAt:        cutoff.Add(-2 * time.Minute),
		LastError:         "failure",
		Attempts:          2,
	}).Run())
	require.NoError(t, db.Insert(&EbicsRTNEvent{
		Source:            "wss",
		IdempotenceKey:    "rtn-retryable-old",
		EbicsHostID:       utils.NewNullInt64(subscriber.EbicsHostID),
		EbicsSubscriberID: utils.NewNullInt64(subscriber.ID),
		Status:            ebicsRTNEventStatusRetryable,
		ReceivedAt:        cutoff.Add(-2 * time.Minute),
		NextRetryAt:       cutoff.Add(time.Minute),
		Attempts:          1,
	}).Run())
	require.NoError(t, db.Insert(&EbicsRTNEvent{
		Source:            "wss",
		IdempotenceKey:    "rtn-quarantine-new",
		EbicsHostID:       utils.NewNullInt64(subscriber.EbicsHostID),
		EbicsSubscriberID: utils.NewNullInt64(subscriber.ID),
		Status:            ebicsRTNEventStatusQuarantined,
		ReceivedAt:        cutoff.Add(-2 * time.Minute),
		Attempts:          1,
	}).Run())
	require.NoError(t, db.Exec(
		"UPDATE ebics_rtn_events SET updated_at=? WHERE idempotence_key=?",
		cutoff.Add(-time.Second), "rtn-processed-old",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_rtn_events SET updated_at=? WHERE idempotence_key=?",
		cutoff.Add(-time.Second), "rtn-failed-old",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_rtn_events SET updated_at=? WHERE idempotence_key=?",
		cutoff.Add(-time.Second), "rtn-retryable-old",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_rtn_events SET updated_at=? WHERE idempotence_key=?",
		cutoff.Add(time.Minute), "rtn-quarantine-new",
	))

	require.NoError(t, PurgeEbicsRTNEventsBefore(db, cutoff))

	count, err := db.Count(&EbicsRTNEvent{}).Where(
		"idempotence_key IN (?, ?, ?, ?)",
		"rtn-processed-old",
		"rtn-failed-old",
		"rtn-retryable-old",
		"rtn-quarantine-new",
	).Run()
	require.NoError(t, err)
	require.EqualValues(t, 2, count)

	var retryable EbicsRTNEvent
	require.NoError(t, db.Get(&retryable, "idempotence_key=?", "rtn-retryable-old").Run())
	require.Equal(t, ebicsRTNEventStatusRetryable, retryable.Status)

	var boundary EbicsRTNEvent
	require.NoError(t, db.Get(&boundary, "idempotence_key=?", "rtn-quarantine-new").Run())
	require.Equal(t, ebicsRTNEventStatusQuarantined, boundary.Status)
}
