package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEbicsTransactionSegmentBeforeWriteRejectsDuplicateSegmentNumber(t *testing.T) {
	db := dbtest.TestDatabase(t)
	host := &EbicsHost{
		Name:            "host-dup",
		HostID:          "HOST-DUP",
		Enabled:         true,
		IsServer:        true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &EbicsSubscriber{
		Name:        "PARTNER-DUP:USER-DUP",
		EbicsHostID: host.ID,
		PartnerID:   "PARTNER-DUP",
		UserID:      "USER-DUP",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	tx := &EbicsTransaction{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-DUP",
		OrderType:         "BTU",
		Status:            "RUNNING",
		Direction:         "INBOUND",
		SegmentCount:      2,
		CurrentSegment:    1,
	}
	require.NoError(t, db.Insert(tx).Run())

	segment := &EbicsTransactionSegment{
		EbicsTransactionID: tx.ID,
		SegmentNumber:      1,
		SegmentStatus:      EbicsTransactionSegmentStatusStoredForRuntime(),
		Checksum:           "first",
	}
	require.NoError(t, db.Insert(segment).Run())

	err := db.Insert(&EbicsTransactionSegment{
		EbicsTransactionID: tx.ID,
		SegmentNumber:      1,
		SegmentStatus:      EbicsTransactionSegmentStatusCompletedForRuntime(),
		Checksum:           "duplicate",
	}).Run()
	require.Error(t, err)
	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "already exists for segment 1")
}
