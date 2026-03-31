package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEbicsNonceBeforeWriteRejectsExpirationNotAfterTimestamp(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertNonceTestSubscriber(t, db, "HOST-NONCE-1", "PARTNER-NONCE-1", "USER-NONCE-1")
	stamp := time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC)

	err := db.Insert(&EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             "nonce-expiration",
		Timestamp:         stamp,
		ExpiresAt:         stamp,
	}).Run()
	require.Error(t, err)
	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "must be after")
}

func TestEbicsNonceBeforeWriteRejectsDuplicatePerSubscriber(t *testing.T) {
	db := dbtest.TestDatabase(t)
	subscriber := insertNonceTestSubscriber(t, db, "HOST-NONCE-2", "PARTNER-NONCE-2", "USER-NONCE-2")
	stamp := time.Date(2026, 3, 31, 14, 5, 0, 0, time.UTC)

	require.NoError(t, db.Insert(&EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             "nonce-dup-model",
		Timestamp:         stamp,
		ExpiresAt:         stamp.Add(time.Minute),
	}).Run())

	err := db.Insert(&EbicsNonce{
		EbicsSubscriberID: subscriber.ID,
		Nonce:             " nonce-dup-model ",
		Timestamp:         stamp.Add(time.Second),
		ExpiresAt:         stamp.Add(2 * time.Minute),
	}).Run()
	require.Error(t, err)
	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "already exists")
}

func TestEbicsNonceBeforeWriteAllowsSameNonceForDifferentSubscribers(t *testing.T) {
	db := dbtest.TestDatabase(t)
	firstSubscriber := insertNonceTestSubscriber(t, db, "HOST-NONCE-3", "PARTNER-NONCE-3A", "USER-NONCE-3A")
	secondSubscriber := insertNonceTestSubscriber(t, db, "HOST-NONCE-3", "PARTNER-NONCE-3B", "USER-NONCE-3B")
	stamp := time.Date(2026, 3, 31, 14, 10, 0, 0, time.UTC)

	require.NoError(t, db.Insert(&EbicsNonce{
		EbicsSubscriberID: firstSubscriber.ID,
		Nonce:             "nonce-shared-model",
		Timestamp:         stamp,
		ExpiresAt:         stamp.Add(time.Minute),
	}).Run())

	require.NoError(t, db.Insert(&EbicsNonce{
		EbicsSubscriberID: secondSubscriber.ID,
		Nonce:             " nonce-shared-model ",
		Timestamp:         stamp.Add(time.Second),
		ExpiresAt:         stamp.Add(2 * time.Minute),
	}).Run())
}

func insertNonceTestSubscriber(t *testing.T, db database.Access, hostID, partnerID, userID string) *EbicsSubscriber {
	t.Helper()

	var host EbicsHost
	getErr := db.Get(&host, "host_id=?", hostID).Run()
	if database.IsNotFound(getErr) {
		host = EbicsHost{
			Name:            hostID,
			HostID:          hostID,
			Enabled:         true,
			IsServer:        true,
			ProtocolVersion: "H005",
			Transport:       "https",
		}
		require.NoError(t, db.Insert(&host).Run())
	} else {
		require.NoError(t, getErr)
	}

	subscriber := &EbicsSubscriber{
		Name:        partnerID + ":" + userID,
		EbicsHostID: host.ID,
		PartnerID:   partnerID,
		UserID:      userID,
		Enabled:     true,
	}
	require.NoError(t, db.Insert(subscriber).Run())

	return subscriber
}
