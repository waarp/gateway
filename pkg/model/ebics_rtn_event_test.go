package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestEbicsRTNEventBeforeWriteRejectsDuplicateIdempotenceKey(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertNonceTestSubscriber(t, db, "HOST-RTN-1", "PARTNER-RTN-1", "USER-RTN-1")
	receivedAt := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)

	require.NoError(t, db.Insert(&EbicsRTNEvent{
		Source:            "wss",
		IdempotenceKey:    "rtn-dup-key",
		EbicsHostID:       utils.NewNullInt64(subscriber.EbicsHostID),
		EbicsSubscriberID: utils.NewNullInt64(subscriber.ID),
		Status:            ebicsRTNEventStatusReceived,
		ReceivedAt:        receivedAt,
	}).Run())

	err := db.Insert(&EbicsRTNEvent{
		Source:            " wss ",
		IdempotenceKey:    " rtn-dup-key ",
		EbicsHostID:       utils.NewNullInt64(subscriber.EbicsHostID),
		EbicsSubscriberID: utils.NewNullInt64(subscriber.ID),
		Status:            ebicsRTNEventStatusDuplicate,
		ReceivedAt:        receivedAt.Add(time.Second),
	}).Run()
	require.Error(t, err)

	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "already exists")
}

func TestEbicsRTNEventBeforeWriteRejectsSubscriberOutsideSelectedHost(t *testing.T) {
	db := dbtest.TestDatabase(t)
	hostA := insertNonceTestSubscriber(t, db, "HOST-RTN-A", "PARTNER-RTN-A", "USER-RTN-A")
	hostB := insertNonceTestSubscriber(t, db, "HOST-RTN-B", "PARTNER-RTN-B", "USER-RTN-B")

	err := db.Insert(&EbicsRTNEvent{
		Source:            "wss",
		IdempotenceKey:    "rtn-host-mismatch",
		EbicsHostID:       utils.NewNullInt64(hostA.EbicsHostID),
		EbicsSubscriberID: utils.NewNullInt64(hostB.ID),
		Status:            ebicsRTNEventStatusReceived,
		ReceivedAt:        time.Date(2026, 4, 1, 10, 5, 0, 0, time.UTC),
	}).Run()
	require.Error(t, err)

	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "does not belong")
}
