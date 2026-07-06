package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestRemoteFilewatcher(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	client := Client{Name: "client", Protocol: testProtocol}
	require.NoError(t, db.Insert(&client).Run())

	rule := Rule{Name: "pull", IsSend: false}
	require.NoError(t, db.Insert(&rule).Run())

	partner := RemoteAgent{
		Name:     "partner",
		Protocol: testProtocol,
		Address:  types.Addr("localhost", 1111),
	}
	require.NoError(t, db.Insert(&partner).Run())

	account := RemoteAccount{RemoteAgentID: partner.ID, Login: "toto"}
	require.NoError(t, db.Insert(&account).Run())

	fw := FileWatcher{
		Flow:             "test_flow",
		Interval:         10 * time.Second,
		Pattern:          "*",
		RemoteAccount:    account,
		Client:           client,
		Rule:             rule,
		NoDuplicateCheck: false,
	}
	require.NoError(t, db.Insert(&fw).Run())

	var check FileWatcher
	require.NoError(t, db.Get(&check, "id=?", fw.ID).Eager().Run())

	assert.Equal(t, fw, check)
}
