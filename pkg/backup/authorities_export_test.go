package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestAuthoritiesExport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	auth1 := &model.Authority{
		Name:           "auth1",
		Type:           auth.AuthorityTLS,
		PublicIdentity: testhelpers.LocalhostCert,
		ValidHosts:     []string{"1.1.1.1", "waarp.fr"},
	}
	auth2 := &model.Authority{
		Name:           "auth2",
		Type:           auth.AuthorityTLS,
		PublicIdentity: testhelpers.OtherLocalhostCert,
		ValidHosts:     []string{"2.2.2.2", "waarp.org"},
	}

	require.NoError(t, db.Insert(auth1).Run())
	require.NoError(t, db.Insert(auth2).Run())

	res, err := exportAuthorities(logger, db)
	require.NoError(t, err)
	require.Len(t, res, 2)

	t.Run("Auth1 exported", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, res[0].Name, auth1.Name)
		assert.Equal(t, res[0].Type, auth1.Type)
		assert.Equal(t, res[0].PublicIdentity, auth1.PublicIdentity)
		assert.Equal(t, res[0].ValidHosts, auth1.ValidHosts)
	})

	t.Run("Auth2 exported", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, res[1].Name, auth2.Name)
		assert.Equal(t, res[1].Type, auth2.Type)
		assert.Equal(t, res[1].PublicIdentity, auth2.PublicIdentity)
		assert.Equal(t, res[1].ValidHosts, auth2.ValidHosts)
	})
}
