package model

import (
	"errors"
	"io/fs"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/filesystems"
)

func TestCloudInstanceTableName(t *testing.T) {
	t.Parallel()

	assert.Equal(t, TableCloudInstances, (&CloudInstance{}).TableName(),
		"The cloud instances table name should be equal to 'cloud_instances'")
}

func TestCloudInstanceBeforeWrite(t *testing.T) {
	t.Parallel()

	const (
		expKey    = "test_key"
		expSecret = "test_secret"
	)

	var (
		scheme = t.Name()

		expOptions = map[string]any{"key": "val"}
		//nolint:goerr113 //this is only valid for this test
		errInvalidConf = errors.New("invalid cloud instance configuration")
	)

	//nolint:unparam //cannot change the function's signature
	filesystems.FileSystems[scheme] = func(key, secret string,
		options map[string]any,
	) (fs.FS, error) {
		if key != expKey || secret != expSecret ||
			!reflect.DeepEqual(options, expOptions) {
			return nil, errInvalidConf
		}

		return nil, nil
	}

	t.Cleanup(func() { delete(filesystems.FileSystems, scheme) })

	db := dbtest.TestDatabase(t)

	validCloud := CloudInstance{
		Name:    "cloud",
		Type:    scheme,
		Key:     expKey,
		Secret:  expSecret,
		Options: expOptions,
	}

	t.Run("Given a valid cloud instance entry", func(t *testing.T) {
		t.Parallel()

		cloud := validCloud

		require.NoError(t, cloud.BeforeWrite(db),
			"Then calling 'BeforeWrite' should NOT return an error")
	})

	t.Run("Given an invalid cloud instance entry", func(t *testing.T) {
		t.Parallel()

		t.Run("Given a cloud instance with no name", func(t *testing.T) {
			t.Parallel()

			cloud := validCloud
			cloud.Name = ""

			require.ErrorContains(t, cloud.BeforeWrite(db),
				"the cloud instance's name cannot be empty",
				"Then calling 'BeforeWrite' should return an error")
		})

		t.Run("Given a cloud instance with an unknown type", func(t *testing.T) {
			t.Parallel()

			cloud := validCloud
			cloud.Type = "unknown"

			require.ErrorContains(t, cloud.BeforeWrite(db),
				"unknown cloud instance type",
				"Then calling 'BeforeWrite' should return an error")
		})

		t.Run("Given a cloud instance with an invalid configuration", func(t *testing.T) {
			t.Parallel()

			cloud := validCloud
			cloud.Key = "invalid"

			require.ErrorContains(t, cloud.BeforeWrite(db),
				"invalid cloud instance configuration",
				"Then calling 'BeforeWrite' should return an error")
		})

		t.Run("Given that the cloud's name is already taken", func(t *testing.T) {
			t.Parallel()

			otherCloud := validCloud
			require.NoError(t, db.Insert(&otherCloud).Run())

			t.Cleanup(func() { require.NoError(t, db.Delete(&otherCloud).Run()) })

			cloud := validCloud

			require.ErrorContains(t, cloud.BeforeWrite(db),
				`a cloud instance named "cloud" already exist`,
				"Then calling 'BeforeWrite' should return an error")
		})
	})
}
