package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEnsureDefaultEbicsRuntimePolicyCreatesSingleton(t *testing.T) {
	db := dbtest.TestDatabase(t)

	policy, err := EnsureDefaultEbicsRuntimePolicy(db)
	require.NoError(t, err)
	require.NotZero(t, policy.ID)
	assert.Equal(t, "default", policy.Name)
	assert.True(t, policy.Enabled)
	assert.EqualValues(t, DefaultEbicsMaintenanceIntervalSeconds, policy.MaintenanceIntervalSeconds)
	assert.EqualValues(t, DefaultEbicsTransactionRetentionSeconds, policy.TransactionRetentionSeconds)
	assert.EqualValues(t, DefaultEbicsRTNEventRetentionSeconds, policy.RTNEventRetentionSeconds)

	reloaded, err := EnsureDefaultEbicsRuntimePolicy(db)
	require.NoError(t, err)
	assert.Equal(t, policy.ID, reloaded.ID)
}

func TestEbicsRuntimePolicyBeforeWriteRejectsNegativeDurations(t *testing.T) {
	db := dbtest.TestDatabase(t)

	policy := DefaultEbicsRuntimePolicy()
	policy.MaintenanceIntervalSeconds = -1

	err := policy.BeforeWrite(db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maintenance interval")
}

func TestEbicsRuntimePolicyDurationHelpers(t *testing.T) {
	policy := DefaultEbicsRuntimePolicy()
	policy.MaintenanceIntervalSeconds = 3600
	policy.TransactionRetentionSeconds = 7200
	policy.RTNEventRetentionSeconds = 10800

	assert.Equal(t, time.Hour, policy.MaintenanceInterval())
	assert.Equal(t, 2*time.Hour, policy.TransactionRetention())
	assert.Equal(t, 3*time.Hour, policy.RTNEventRetention())
}
