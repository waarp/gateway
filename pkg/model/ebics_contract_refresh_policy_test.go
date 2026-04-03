package model

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEbicsContractRefreshPolicyBeforeWriteDefaults(t *testing.T) {
	db := dbtest.TestDatabase(t)
	client := insertTestEBICSClientForPolicy(t, db, "policy-client")
	host := insertTestEbicsHostForPolicy(t, db, "HOST-POLICY")
	subscriber := insertTestEbicsSubscriberForPolicy(t, db, host.ID, "PARTNER-POLICY", "USER-POLICY")

	policy := &EbicsContractRefreshPolicy{
		Name:              "daily-policy",
		Enabled:           true,
		ClientID:          client.ID,
		EbicsSubscriberID: subscriber.ID,
		IncludeHEV:        true,
	}
	require.NoError(t, db.Insert(policy).Run())

	require.NotZero(t, policy.NextRunAt)
	assert.EqualValues(t, DefaultEbicsContractRefreshIntervalSeconds, policy.IntervalSeconds)
	assert.Equal(t, "READY", policy.Status)
	assert.True(t, policy.IncludeHEV)
}

func TestEbicsContractRefreshPolicyRejectsServerSubscriber(t *testing.T) {
	db := dbtest.TestDatabase(t)
	client := insertTestEBICSClientForPolicy(t, db, "policy-client-server")
	host := insertTestEbicsHostForPolicy(t, db, "HOST-POLICY-SERVER")
	subscriber := insertTestEbicsSubscriberForPolicy(t, db, host.ID, "PARTNER-SERVER", "USER-SERVER")
	require.NoError(t, db.Exec("UPDATE ebics_subscribers SET account_role=? WHERE id=?", "SERVER", subscriber.ID))

	policy := &EbicsContractRefreshPolicy{
		Name:              "server-policy",
		Enabled:           true,
		ClientID:          client.ID,
		EbicsSubscriberID: subscriber.ID,
	}
	err := db.Insert(policy).Run()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client-side remote account")
}

func insertTestEBICSClientForPolicy(t *testing.T, db *database.DB, name string) *Client {
	t.Helper()
	require.NoError(t, db.Exec(
		"INSERT INTO clients(id, owner, name, protocol, disabled) VALUES (?, ?, ?, ?, ?)",
		1, conf.GlobalConfig.GatewayName, name, "ebics", false,
	))
	client := &Client{}
	require.NoError(t, db.Get(client, "name=?", name).Owner().Run())
	return client
}

func insertTestEbicsHostForPolicy(t *testing.T, db *database.DB, hostID string) *EbicsHost {
	t.Helper()
	host := &EbicsHost{
		Name:            hostID,
		HostID:          hostID,
		Enabled:         true,
		ProtocolVersion: "H005",
		Transport:       "https",
	}
	require.NoError(t, db.Insert(host).Run())
	return host
}

func insertTestEbicsSubscriberForPolicy(
	t *testing.T,
	db *database.DB,
	hostID int64,
	partnerID, userID string,
) *EbicsSubscriber {
	t.Helper()
	const remoteAgentID int64 = 1
	require.NoError(t, db.Exec(
		"INSERT INTO remote_agents(id, owner, name, protocol, address, proto_config) VALUES (?, ?, ?, ?, ?, ?)",
		remoteAgentID, conf.GlobalConfig.GatewayName, "remote-policy-agent", "https", "127.0.0.1:443", "{}",
	))
	remoteAccount := &RemoteAccount{Login: "remote-policy-account", RemoteAgentID: remoteAgentID}
	require.NoError(t, db.Insert(remoteAccount).Run())

	subscriber := &EbicsSubscriber{
		EbicsHostID:     hostID,
		PartnerID:       partnerID,
		UserID:          userID,
		RemoteAccountID: sqlNullInt64(remoteAccount.ID),
		Enabled:         true,
	}
	require.NoError(t, db.Insert(subscriber).Run())
	return subscriber
}

func sqlNullInt64(id int64) sql.NullInt64 {
	return sql.NullInt64{Int64: id, Valid: true}
}
