package ebics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type fakeContractRefreshTicker struct {
	ch chan time.Time
}

func newFakeContractRefreshTicker() *fakeContractRefreshTicker {
	return &fakeContractRefreshTicker{ch: make(chan time.Time, 4)}
}

func (t *fakeContractRefreshTicker) Chan() <-chan time.Time { return t.ch }
func (t *fakeContractRefreshTicker) Stop()                  {}

func insertContractRefreshClient(t *testing.T, db *database.DB, name string) *model.Client {
	t.Helper()

	require.NoError(t, db.Exec(
		"INSERT INTO clients(id, owner, name, protocol, disabled) VALUES (?, ?, ?, ?, ?)",
		time.Now().UTC().UnixNano(), conf.GlobalConfig.GatewayName, name, "ebics", false,
	))
	client := &model.Client{}
	require.NoError(t, db.Get(client, "name=?", name).Owner().Run())

	return client
}

func attachContractRefreshRemoteAccount(t *testing.T, db *database.DB, subscriber *model.EbicsSubscriber, suffix string) {
	t.Helper()

	agentID := time.Now().UTC().UnixNano()
	require.NoError(t, db.Exec(
		"INSERT INTO remote_agents(id, owner, name, protocol, address, proto_config) VALUES (?, ?, ?, ?, ?, ?)",
		agentID, conf.GlobalConfig.GatewayName, "contract-refresh-agent-"+suffix, "https", "127.0.0.1:443", "{}",
	))
	account := &model.RemoteAccount{RemoteAgentID: agentID, Login: "contract-refresh-account-" + suffix}
	require.NoError(t, db.Insert(account).Run())

	subscriber.RemoteAccountID = subscriber.RemoteAccountID
	subscriber.RemoteAccountID.Int64 = account.ID
	subscriber.RemoteAccountID.Valid = true
	require.NoError(t, db.Update(subscriber).Run())
}

func TestContractRefreshServiceRunsDuePolicy(t *testing.T) {
	db := dbtest.TestDatabase(t)
	client := insertContractRefreshClient(t, db, "contract-refresh-client")
	host := insertTestEbicsHost(t, db, "HOST-CR")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-CR", "USER-CR", true)
	attachContractRefreshRemoteAccount(t, db, subscriber, "a")
	now := time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC)

	policy := &model.EbicsContractRefreshPolicy{
		Name:              "daily-bank-a",
		Enabled:           true,
		ClientID:          client.ID,
		EbicsSubscriberID: subscriber.ID,
		IncludeHEV:        true,
		IntervalSeconds:   3600,
		NextRunAt:         now.Add(-time.Minute),
	}
	require.NoError(t, db.Insert(policy).Run())

	service := NewContractRefreshService(db)
	service.run = func(context.Context, *database.DB, int64, int64, bool) (*ContractRefreshResult, error) {
		return &ContractRefreshResult{}, nil
	}

	require.NoError(t, service.runDuePoliciesAt(context.Background(), now))

	var refreshed model.EbicsContractRefreshPolicy
	require.NoError(t, db.Get(&refreshed, "id=?", policy.ID).Run())
	assert.Equal(t, "READY", refreshed.Status)
	assert.Equal(t, now, refreshed.LastAttemptAt.UTC())
	assert.Equal(t, now, refreshed.LastSuccessAt.UTC())
	assert.Equal(t, now.Add(time.Hour), refreshed.NextRunAt.UTC())
	assert.Empty(t, refreshed.LastError)
}

func TestContractRefreshServicePersistsFailureState(t *testing.T) {
	db := dbtest.TestDatabase(t)
	client := insertContractRefreshClient(t, db, "contract-refresh-client-failed")
	host := insertTestEbicsHost(t, db, "HOST-CR-ERR")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-CR-ERR", "USER-CR-ERR", true)
	attachContractRefreshRemoteAccount(t, db, subscriber, "b")
	now := time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)

	policy := &model.EbicsContractRefreshPolicy{
		Name:              "daily-bank-b",
		Enabled:           true,
		ClientID:          client.ID,
		EbicsSubscriberID: subscriber.ID,
		IncludeHEV:        false,
		IntervalSeconds:   7200,
		NextRunAt:         now.Add(-time.Minute),
	}
	require.NoError(t, db.Insert(policy).Run())

	service := NewContractRefreshService(db)
	service.run = func(context.Context, *database.DB, int64, int64, bool) (*ContractRefreshResult, error) {
		return nil, errors.New("bank temporarily unavailable")
	}

	require.NoError(t, service.runDuePoliciesAt(context.Background(), now))

	var refreshed model.EbicsContractRefreshPolicy
	require.NoError(t, db.Get(&refreshed, "id=?", policy.ID).Run())
	assert.Equal(t, "ERROR", refreshed.Status)
	assert.Equal(t, "bank temporarily unavailable", refreshed.LastError)
	assert.Equal(t, now.Add(2*time.Hour), refreshed.NextRunAt.UTC())
	assert.True(t, refreshed.LastSuccessAt.IsZero())
}

func TestContractRefreshServiceStartStop(t *testing.T) {
	db := dbtest.TestDatabase(t)
	service := NewContractRefreshService(db)
	ticker := newFakeContractRefreshTicker()
	service.newTicker = func(time.Duration) contractRefreshTicker { return ticker }

	require.NoError(t, service.Start())
	code, reason := service.State()
	require.Equal(t, utils.StateRunning, code)
	require.Empty(t, reason)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, service.Stop(ctx))

	code, reason = service.State()
	require.Equal(t, utils.StateOffline, code)
	require.Empty(t, reason)
}
