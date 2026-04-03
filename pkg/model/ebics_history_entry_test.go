package model

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEbicsHistoryEntryRoundTrip(t *testing.T) {
	db := dbtest.TestDatabase(t)

	host := &EbicsHost{
		Name:            "host-history",
		HostID:          "HOST-HISTORY",
		Enabled:         true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())

	subscriber := &EbicsSubscriber{
		EbicsHostID: host.ID,
		Name:        "partner:user",
		PartnerID:   "PARTNER",
		UserID:      "USER",
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	entry := &EbicsHistoryEntry{
		HistoryType:       EbicsHistoryTypeActionForRuntime(),
		OperationType:     "INITIALIZATION",
		Action:            "CANCEL",
		Status:            "CANCELLED",
		EbicsHostID:       host.ID,
		EbicsSubscriberID: subscriber.ID,
		WorkflowID:        sql.NullInt64{Int64: 0, Valid: false},
		EvidenceMap:       map[string]any{"ticket": "INIT-1"},
		MetadataMap:       map[string]any{"currentStep": "CANCELLED"},
	}

	require.NoError(t, db.Insert(entry).Run())

	var reloaded EbicsHistoryEntry
	require.NoError(t, db.Get(&reloaded, "id=?", entry.ID).Run())
	assert.Equal(t, "CANCEL", reloaded.Action)
	assert.Equal(t, "INIT-1", reloaded.EvidenceMap["ticket"])
	assert.Equal(t, "CANCELLED", reloaded.MetadataMap["currentStep"])
}

func TestEbicsHistoryEntryBeforeWriteRejectsMissingSubscriber(t *testing.T) {
	db := dbtest.TestDatabase(t)

	entry := &EbicsHistoryEntry{
		HistoryType:   EbicsHistoryTypeActionForRuntime(),
		OperationType: "INITIALIZATION",
		Status:        "READY",
		EbicsHostID:   1,
	}

	err := entry.BeforeWrite(db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "subscriber")
}
