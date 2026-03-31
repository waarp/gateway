package model

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEbicsTransactionBeforeWriteRejectsCurrentSegmentBeyondTotal(t *testing.T) {
	db := dbtest.TestDatabase(t)
	host := &EbicsHost{
		Name:            "host-seg",
		HostID:          "HOST-SEG",
		Enabled:         true,
		IsServer:        true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &EbicsSubscriber{
		Name:        "PARTNER-SEG:USER-SEG",
		EbicsHostID: host.ID,
		PartnerID:   "PARTNER-SEG",
		UserID:      "USER-SEG",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	tx := &EbicsTransaction{
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		TransactionID:     "TX-SEG-INVALID",
		OrderType:         "BTU",
		Status:            "RUNNING",
		Direction:         "INBOUND",
		SegmentCount:      2,
		CurrentSegment:    3,
	}

	err := db.Insert(tx).Run()
	require.Error(t, err)
	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "current segment cannot exceed")
}
