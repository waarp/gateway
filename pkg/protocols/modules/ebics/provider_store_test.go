package ebics

import (
	"context"
	"testing"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestProviderStoreTransactionLifecycle(t *testing.T) {
	db := dbtest.TestDatabase(t)
	store := newProviderStore(db)
	host := insertTestEbicsHost(t, db, "HOST1")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER1", "USER1", true)
	createdAt := time.Date(2026, 3, 31, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Minute)

	tx := libebics.Transaction{
		ID:         "TX-001",
		HostID:     "HOST1",
		PartnerID:  "PARTNER1",
		UserID:     "USER1",
		OrderType:  "BTU",
		SegmentCnt: 3,
		Status:     "RUNNING",
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
	}

	require.NoError(t, store.CreateTransaction(context.Background(), tx))

	var row model.EbicsTransaction
	require.NoError(t, db.Get(&row, "transaction_id=?", "TX-001").Run())
	require.Equal(t, subscriber.ID, row.EbicsSubscriberID)
	require.Equal(t, host.ID, row.EbicsHostID)
	require.Equal(t, "RUNNING", row.Status)
	require.Equal(t, 3, row.SegmentCount)
	require.Equal(t, model.EbicsOperationDirectionInboundForRuntime(), row.Direction)

	got, err := store.GetTransaction(context.Background(), "TX-001")
	require.NoError(t, err)
	require.Equal(t, libebics.TransactionID("TX-001"), got.ID)
	require.Equal(t, libebics.HostID("HOST1"), got.HostID)
	require.Equal(t, libebics.PartnerID("PARTNER1"), got.PartnerID)
	require.Equal(t, libebics.UserID("USER1"), got.UserID)
	require.Equal(t, libebics.OrderType("BTU"), got.OrderType)
	require.Equal(t, "RUNNING", got.Status)
	require.Equal(t, 3, got.SegmentCnt)

	require.NoError(t, store.UpdateTransaction(context.Background(), libebics.Transaction{
		ID:         "TX-001",
		HostID:     "HOST1",
		PartnerID:  "PARTNER1",
		UserID:     "USER1",
		OrderType:  "BTU",
		SegmentCnt: 5,
		Status:     "COMPLETED",
		UpdatedAt:  updatedAt.Add(2 * time.Minute),
	}))

	require.NoError(t, db.Get(&row, "transaction_id=?", "TX-001").Run())
	require.Equal(t, model.EbicsTransactionStatusCompletedForRuntime(), row.Status)
	require.Equal(t, 5, row.SegmentCount)
}

func TestProviderStoreSegmentAndRecoveryLifecycle(t *testing.T) {
	db := dbtest.TestDatabase(t)
	store := newProviderStore(db)
	host := insertTestEbicsHost(t, db, "HOST1")
	insertTestEbicsSubscriber(t, db, host.ID, "PARTNER1", "USER1", true)

	require.NoError(t, store.CreateTransaction(context.Background(), libebics.Transaction{
		ID:         "TX-SEG-1",
		HostID:     "HOST1",
		PartnerID:  "PARTNER1",
		UserID:     "USER1",
		OrderType:  "BTU",
		SegmentCnt: 2,
		Status:     "RUNNING",
	}))

	ok, err := store.HasSegment(context.Background(), "TX-SEG-1", 1)
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, store.AddSegment(context.Background(), "TX-SEG-1", libebics.SegmentInfo{
		Number:     1,
		Last:       false,
		HasSegment: true,
		Total:      2,
	}, []byte("hash-1")))

	ok, err = store.HasSegment(context.Background(), "TX-SEG-1", 1)
	require.NoError(t, err)
	require.True(t, ok)

	var txRow model.EbicsTransaction
	require.NoError(t, db.Get(&txRow, "transaction_id=?", "TX-SEG-1").Run())
	require.Equal(t, 1, txRow.CurrentSegment)
	require.Equal(t, 2, txRow.SegmentCount)

	var segment model.EbicsTransactionSegment
	require.NoError(t, db.Get(&segment, "ebics_transaction_id=? AND segment_number=?", txRow.ID, 1).Run())
	require.Equal(t, model.EbicsTransactionSegmentStatusStoredForRuntime(), segment.SegmentStatus)
	require.Contains(t, segment.Checksum, "686173682d31")

	require.NoError(t, store.AddSegment(context.Background(), "TX-SEG-1", libebics.SegmentInfo{
		Number:          2,
		Last:            true,
		HasSegment:      true,
		Total:           2,
		RecoveryPoint:   7,
		RecoveryCounter: 3,
	}, []byte("hash-2")))

	require.NoError(t, db.Get(&segment, "ebics_transaction_id=? AND segment_number=?", txRow.ID, 2).Run())
	require.Equal(t, model.EbicsTransactionSegmentStatusCompletedForRuntime(), segment.SegmentStatus)
	require.EqualValues(t, 7, segment.MetadataMap["recoveryPoint"])
	require.EqualValues(t, 3, segment.MetadataMap["recoveryCounter"])

	point, counter, err := store.GetRecovery(context.Background(), "TX-SEG-1")
	require.NoError(t, err)
	require.Zero(t, point)
	require.Zero(t, counter)

	require.NoError(t, store.UpdateRecovery(context.Background(), "TX-SEG-1", 11, 4))

	point, counter, err = store.GetRecovery(context.Background(), "TX-SEG-1")
	require.NoError(t, err)
	require.Equal(t, 11, point)
	require.Equal(t, 4, counter)
}

func TestProviderStoreNonceLifecycle(t *testing.T) {
	db := dbtest.TestDatabase(t)
	store := newProviderStore(db)
	host := insertTestEbicsHost(t, db, "HOST1")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER1", "USER1", true)
	ts := time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC)

	seen, err := store.SeenNonce(context.Background(), "HOST1", "PARTNER1", "USER1", "nonce-1")
	require.NoError(t, err)
	require.False(t, seen)

	require.NoError(t, store.StoreNonce(context.Background(), "HOST1", "PARTNER1", "USER1", " nonce-1 ", ts))

	seen, err = store.SeenNonce(context.Background(), "HOST1", "PARTNER1", "USER1", "nonce-1")
	require.NoError(t, err)
	require.True(t, seen)

	var nonce model.EbicsNonce
	require.NoError(t, db.Get(&nonce, "ebics_subscriber_id=? AND nonce=?", subscriber.ID, "nonce-1").Run())
	require.True(t, nonce.Timestamp.UTC().Equal(ts))
	require.True(t, nonce.ExpiresAt.UTC().Equal(ts.Add(defaultNonceTTL)))

	require.NoError(t, store.PurgeBefore(context.Background(), ts.Add(defaultNonceTTL).Add(time.Second)))

	count, err := db.Count(&model.EbicsNonce{}).Where("ebics_subscriber_id=?", subscriber.ID).Run()
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestProviderStorePurgeTransactionsBefore(t *testing.T) {
	db := dbtest.TestDatabase(t)
	store := newProviderStore(db)
	host := insertTestEbicsHost(t, db, "HOST1")
	insertTestEbicsSubscriber(t, db, host.ID, "PARTNER1", "USER1", true)
	cutoff := time.Date(2026, 3, 31, 12, 0, 0, 0, time.UTC)

	require.NoError(t, store.CreateTransaction(context.Background(), libebics.Transaction{
		ID:        "TX-OLD",
		HostID:    "HOST1",
		PartnerID: "PARTNER1",
		UserID:    "USER1",
		OrderType: "BTU",
		Status:    "RUNNING",
		UpdatedAt: cutoff.Add(-time.Minute),
	}))
	require.NoError(t, store.CreateTransaction(context.Background(), libebics.Transaction{
		ID:        "TX-NEW",
		HostID:    "HOST1",
		PartnerID: "PARTNER1",
		UserID:    "USER1",
		OrderType: "BTU",
		Status:    "RUNNING",
		UpdatedAt: cutoff.Add(time.Minute),
	}))

	require.NoError(t, db.Exec(
		"UPDATE ebics_transactions SET updated_at=? WHERE transaction_id=?",
		cutoff.Add(-time.Minute), "TX-OLD",
	))
	require.NoError(t, db.Exec(
		"UPDATE ebics_transactions SET updated_at=? WHERE transaction_id=?",
		cutoff.Add(time.Minute), "TX-NEW",
	))

	require.NoError(t, store.PurgeTransactionsBefore(context.Background(), cutoff))

	tx, err := store.GetTransaction(context.Background(), "TX-OLD")
	require.NoError(t, err)
	require.Empty(t, tx.ID)

	tx, err = store.GetTransaction(context.Background(), "TX-NEW")
	require.NoError(t, err)
	require.Equal(t, libebics.TransactionID("TX-NEW"), tx.ID)

	count, err := db.Count(&model.EbicsTransaction{}).Where("transaction_id IN (?, ?)", "TX-OLD", "TX-NEW").Run()
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}
