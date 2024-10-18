package backup

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs/fstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestExportClouds(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	fsType := fstest.MakeDummyBackend(t)

	db := dbtest.TestDatabase(t)

	dbCloud := model.CloudInstance{
		Name:    "remote-fs",
		Type:    fsType,
		Key:     "access-key",
		Secret:  "access-secret",
		Options: map[string]string{"key1": "val1", "key2": "val2"},
	}
	require.NoError(t, db.Insert(&dbCloud).Run())

	res, err := exportClouds(logger, db)
	require.NoError(t, err)

	require.Len(t, res, 1)

	require.Equal(t, dbCloud.Name, res[0].Name)
	require.Equal(t, dbCloud.Type, res[0].Type)
	require.Equal(t, dbCloud.Key, res[0].Key)
	require.Equal(t, dbCloud.Secret.String(), res[0].Secret)
	require.Equal(t, dbCloud.Options, res[0].Options)
}
