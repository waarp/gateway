package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
)

func TestCloudInstanceTableName(t *testing.T) {
	t.Parallel()

	assert.Equalf(t, TableCloudInstances, (&CloudInstance{}).TableName(),
		"The cloud instances table name should be equal to %q", TableCloudInstances)
}

func TestCloudInstanceBeforeWrite(t *testing.T) {
	t.Parallel()

	const (
		expName   = "test"
		expKey    = "test_key"
		expSecret = "test_secret"
	)

	expOptions := map[string]string{"key": "val"}
	kind := fstest.MakeStaticBackend(t, expName, expKey, expSecret, expOptions)
	db := dbtest.TestDatabase(t)

	validCloud := CloudInstance{
		Name:    expName,
		Type:    kind,
		Key:     expKey,
		Secret:  expSecret,
		Options: expOptions,
	}

	t.Run("Given a valid cloud instance entry", func(t *testing.T) {
		cloud := validCloud

		require.NoError(t, cloud.BeforeWrite(db),
			"Then calling 'BeforeWrite' should NOT return an error")
	})

	t.Run("Given an invalid cloud instance entry", func(t *testing.T) {
		t.Run("Given a cloud instance with no name", func(t *testing.T) {
			cloud := validCloud
			cloud.Name = ""

			require.ErrorContains(t, cloud.BeforeWrite(db),
				"the cloud instance's name cannot be empty",
				"Then calling 'BeforeWrite' should return an error")
		})

		t.Run("Given a cloud instance with an unknown type", func(t *testing.T) {
			cloud := validCloud
			cloud.Type = "unknown"

			require.ErrorContains(t, cloud.BeforeWrite(db),
				fs.ErrUnknownFSType.Error(),
				"Then calling 'BeforeWrite' should return an error")
		})

		t.Run("Given a cloud instance with an invalid configuration", func(t *testing.T) {
			cloud := validCloud
			cloud.Key = "invalid"

			require.ErrorContains(t, cloud.BeforeWrite(db),
				"invalid cloud instance configuration",
				"Then calling 'BeforeWrite' should return an error")
		})

		t.Run("Given that the cloud's name is already taken", func(t *testing.T) {
			otherCloud := validCloud
			require.NoError(t, db.Insert(&otherCloud).Run())

			t.Cleanup(func() { require.NoError(t, db.Delete(&otherCloud).Run()) })

			cloud := validCloud

			require.ErrorContains(t, cloud.BeforeWrite(db),
				fmt.Sprintf(`a cloud instance named "%s" already exist`, cloud.Name),
				"Then calling 'BeforeWrite' should return an error")
		})
	})
}
