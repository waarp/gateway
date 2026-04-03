package model

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEbicsServerContractSetBeforeWriteRejectsSubscriberScopeForHPD(t *testing.T) {
	db := dbtest.TestDatabase(t)
	host := insertNonceTestSubscriber(t, db, "HOST-SCS-HPD", "PARTNER-SCS-HPD", "USER-SCS-HPD")

	err := db.Insert(&EbicsServerContractSet{
		EbicsHostID:       host.EbicsHostID,
		EbicsSubscriberID: sql.NullInt64{Int64: host.ID, Valid: true},
		SourceOrderType:   "HPD",
		Status:            "ACTIVE",
	}).Run()
	require.Error(t, err)

	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "must stay host-scoped")
}

func TestEbicsServerContractSetBeforeWriteRequiresSubscriberForHKD(t *testing.T) {
	db := dbtest.TestDatabase(t)
	host := insertNonceTestSubscriber(t, db, "HOST-SCS-HKD", "PARTNER-SCS-HKD", "USER-SCS-HKD")

	err := db.Insert(&EbicsServerContractSet{
		EbicsHostID:     host.EbicsHostID,
		SourceOrderType: "HKD",
		Status:          "ACTIVE",
	}).Run()
	require.Error(t, err)

	var validationErr *database.ValidationError
	require.ErrorAs(t, err, &validationErr)
	require.ErrorContains(t, err, "requires a subscriber")
}
