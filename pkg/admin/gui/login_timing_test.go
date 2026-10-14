package gui

import (
	"testing"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The test changes utils.BcryptRounds, so it cannot run in parallel.
//
//nolint:paralleltest //changes a package variable
func TestCheckUserTiming(t *testing.T) {
	const timingCost = 10

	db := dbtest.TestDatabase(t)
	t.Cleanup(utils.ChangeBcryptRounds(timingCost))

	hash, err := utils.HashPassword("sesame")
	require.NoError(t, err)
	require.NoError(t, db.Insert(&model.User{
		Username: "timing_user", PasswordHash: hash, Permissions: model.PermAll,
	}).Run())

	measure := func(user, password string) (time.Duration, error) {
		best := time.Duration(1<<63 - 1)

		var err error

		for range 3 {
			start := time.Now()
			_, err = checkUser(db, user, password)

			if d := time.Since(start); d < best {
				best = d
			}
		}

		return best, err
	}

	_, _ = measure("unknown_user", "warm-up")

	existing, errExisting := measure("timing_user", "not_sesame")
	unknown, errUnknown := measure("unknown_user", "not_sesame")

	require.ErrorIs(t, errExisting, ErrAuthentication)
	require.ErrorIs(t, errUnknown, ErrAuthentication)
	assert.GreaterOrEqual(t, unknown, existing/2,
		"unknown account: %v, existing account: %v", unknown, existing)

	_, errGood := measure("timing_user", "sesame")
	require.NoError(t, errGood)
}
