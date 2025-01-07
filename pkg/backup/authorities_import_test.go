package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestNewAuthoritiesImport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	dbAuth := &model.Authority{
		Name:           "auth1",
		Type:           auth.AuthorityTLS,
		PublicIdentity: testhelpers.LocalhostCert,
		ValidHosts:     []string{"1.1.1.1", "waarp.fr"},
	}
	require.NoError(t, db.Insert(dbAuth).Run())

	t.Run("No reset", func(t *testing.T) {
		newAuthority := &file.Authority{
			Name:           "auth2",
			Type:           auth.AuthorityTLS,
			PublicIdentity: testhelpers.OtherLocalhostCert,
			ValidHosts:     []string{"2.2.2.2", "waarp.org"},
		}
		newAuthorities := []*file.Authority{newAuthority}

		require.NoError(t, importAuthorities(logger, db, newAuthorities, false))

		t.Run("New authority is imported", func(t *testing.T) {
			var check model.Authority
			require.NoError(t, db.Get(&check, "name=?", newAuthority.Name).Run())

			assert.Equal(t, check.Name, newAuthority.Name)
			assert.Equal(t, check.Type, newAuthority.Type)
			assert.Equal(t, check.PublicIdentity, newAuthority.PublicIdentity)
			assert.Equal(t, check.ValidHosts, newAuthority.ValidHosts)
		})

		t.Run("Other authorities are untouched", func(t *testing.T) {
			var check model.Authorities
			require.NoError(t, db.Select(&check).Where("name<>?", newAuthority.Name).Run())
			require.Len(t, check, 1)
			assert.Equal(t, check[0], dbAuth)
		})
	})

	t.Run("With reset", func(t *testing.T) {
		newAuthority := &file.Authority{
			Name:           "auth3",
			Type:           auth.AuthorityTLS,
			PublicIdentity: testhelpers.OtherLocalhostCert,
			ValidHosts:     []string{"3.3.3.3", "waarp.it"},
		}
		newAuthorities := []*file.Authority{newAuthority}

		require.NoError(t, importAuthorities(logger, db, newAuthorities, true))

		t.Run("New authority is imported", func(t *testing.T) {
			var check model.Authority
			require.NoError(t, db.Get(&check, "name=?", newAuthority.Name).Run())

			assert.Equal(t, check.Name, newAuthority.Name)
			assert.Equal(t, check.Type, newAuthority.Type)
			assert.Equal(t, check.PublicIdentity, newAuthority.PublicIdentity)
			assert.Equal(t, check.ValidHosts, newAuthority.ValidHosts)
		})

		t.Run("Other authorities are gone", func(t *testing.T) {
			var check model.Authorities
			require.NoError(t, db.Select(&check).Where("name<>?", newAuthority.Name).Run())
			assert.Empty(t, check)
		})
	})
}

func TestExistingAuthoritiesImport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	dbAuth1 := &model.Authority{
		Name:           "auth1",
		Type:           auth.AuthorityTLS,
		PublicIdentity: testhelpers.LocalhostCert,
		ValidHosts:     []string{"1.1.1.1", "waarp.fr"},
	}
	require.NoError(t, db.Insert(dbAuth1).Run())

	dbAuth2 := &model.Authority{
		Name:           "auth2",
		Type:           auth.AuthorityTLS,
		PublicIdentity: testhelpers.LocalhostCert,
		ValidHosts:     []string{"2.2.2.2", "waarp.org"},
	}
	require.NoError(t, db.Insert(dbAuth2).Run())

	t.Run("No reset", func(t *testing.T) {
		newAuthority := &file.Authority{
			Name:           dbAuth2.Name,
			Type:           auth.AuthorityTLS,
			PublicIdentity: testhelpers.OtherLocalhostCert,
			ValidHosts:     []string{"3.3.3.3", "waarp.it"},
		}
		newAuthorities := []*file.Authority{newAuthority}

		require.NoError(t, importAuthorities(logger, db, newAuthorities, false))

		t.Run("Affected authority is updated", func(t *testing.T) {
			var check model.Authority
			require.NoError(t, db.Get(&check, "name=?", newAuthority.Name).Run())

			assert.Equal(t, check.Name, newAuthority.Name)
			assert.Equal(t, check.Type, newAuthority.Type)
			assert.Equal(t, check.PublicIdentity, newAuthority.PublicIdentity)
			assert.Equal(t, check.ValidHosts, newAuthority.ValidHosts)
		})

		t.Run("Other authorities are untouched", func(t *testing.T) {
			var check model.Authorities
			require.NoError(t, db.Select(&check).Where("name<>?", newAuthority.Name).Run())
			require.Len(t, check, 1)
			assert.Equal(t, check[0], dbAuth1)
		})
	})
}
