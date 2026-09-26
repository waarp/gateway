package auth

import (
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// The test changes utils.BcryptRounds, so it cannot run in parallel.
//
//nolint:paralleltest //changes a package variable
func TestPasswordAuthenticationTiming(t *testing.T) {
	const (
		timingCost = 10
		protocol   = "timing_proto"
	)

	modeltest.AddDummyProtoConfig(protocol)
	db := dbtest.TestDatabase(t)

	t.Cleanup(utils.ChangeBcryptRounds(timingCost))

	server := &model.LocalAgent{Name: "server", Protocol: protocol, Address: types.Addr("localhost", 1)}
	require.NoError(t, db.Insert(server).Run())

	existing := &model.LocalAccount{LocalAgentID: server.ID, Login: "existing"}
	require.NoError(t, db.Insert(existing).Run())
	require.NoError(t, db.Insert(&model.Credential{
		LocalAccountID: existing.NullableID(), Type: Password, Value: "sesame",
	}).Run())

	// An unknown account, as returned by LocalAgent.GetAccount.
	unknown := &model.LocalAccount{LocalAgentID: server.ID, Login: "unknown"}

	handler := BcryptAuthHandler{}
	measure := func(acc *model.LocalAccount, password string) time.Duration {
		best := time.Duration(1<<63 - 1)

		for range 3 {
			start := time.Now()
			res, err := handler.Authenticate(db, acc, password)

			d := time.Since(start)

			require.NoError(t, err)
			require.False(t, res.Success)

			if d < best {
				best = d
			}
		}

		return best
	}

	measure(unknown, "warm-up")

	withAccount := measure(existing, "not_sesame")
	withoutAccount := measure(unknown, "not_sesame")

	assert.GreaterOrEqual(t, withoutAccount, withAccount/2,
		"unknown account: %v, existing account: %v", withoutAccount, withAccount)

	res, err := handler.Authenticate(db, existing, "sesame")
	require.NoError(t, err)
	assert.True(t, res.Success)
}
