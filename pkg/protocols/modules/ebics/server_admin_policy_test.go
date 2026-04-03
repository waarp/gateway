package ebics

import (
	"testing"

	libebics "code.waarp.fr/lib/ebics/ebics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestServerAdminPolicyRejectsDisabledSubscriber(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	agent := insertTestEBICSServer(t, db, 0)
	account := insertTestLocalAccount(t, db, agent.ID, "ebics-admin-policy")
	host := insertTestEbicsHost(t, db, "HOST-POLICY")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-POLICY", "USER-POLICY", false)
	subscriber.LocalAccountID.Valid = true
	subscriber.LocalAccountID.Int64 = account.ID
	require.NoError(t, db.Update(subscriber).Run())

	policy := &serverAdminPolicy{store: newProviderStore(db)}
	err := policy.validateOperationalSubscriber(libebics.OrderContext{
		HostID:    libebics.HostID(host.HostID),
		PartnerID: libebics.PartnerID(subscriber.PartnerID),
		UserID:    libebics.UserID(subscriber.UserID),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}
