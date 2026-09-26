package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestDeletionsRequireDeletePermission(t *testing.T) {
	t.Parallel()

	const password = "sesame"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	require.NoError(t, err)

	cases := []struct {
		name        string
		path        string
		writeOnly   model.PermsMask
		withDelete  model.PermsMask
		makeObjects func(t *testing.T, insert func(database.InsertBean))
	}{
		{
			name:       "client",
			path:       "/api/clients/test_client",
			writeOnly:  model.PermServersRead | model.PermServersWrite,
			withDelete: model.PermServersRead | model.PermServersWrite | model.PermServersDelete,
			makeObjects: func(_ *testing.T, insert func(database.InsertBean)) {
				insert(&model.Client{Name: "test_client", Protocol: testProto1})
			},
		},
		{
			name:       "authentication authority",
			path:       "/api/authorities/test_authority",
			writeOnly:  model.PermAdminRead | model.PermAdminWrite,
			withDelete: model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete,
			makeObjects: func(_ *testing.T, insert func(database.InsertBean)) {
				insert(&model.Authority{
					Name: "test_authority", Type: auth.AuthorityTLS, PublicIdentity: testhelpers.LocalhostCert,
				})
			},
		},
	}

	for _, tc := range cases {
		t.Run("Deleting a "+tc.name, func(t *testing.T) {
			t.Parallel()

			db := dbtest.TestDatabase(t)
			insert := func(obj database.InsertBean) { require.NoError(t, db.Insert(obj).Run()) }

			tc.makeObjects(t, insert)
			require.NoError(t, db.Insert(&model.User{
				Username: "writer", PasswordHash: string(hash), Permissions: tc.writeOnly,
			}).Run())
			require.NoError(t, db.Insert(&model.User{
				Username: "deleter", PasswordHash: string(hash), Permissions: tc.withDelete,
			}).Run())

			router := mux.NewRouter()
			MakeRESTHandler(testhelpers.GetTestLogger(t), db, router.PathPrefix("/api").Subrouter())

			del := func(user string) int {
				req := httptest.NewRequest(http.MethodDelete, tc.path, nil)
				req.SetBasicAuth(user, password)

				rec := httptest.NewRecorder()
				router.ServeHTTP(rec, req)

				return rec.Code
			}

			assert.Equal(t, http.StatusForbidden, del("writer"),
				"write permission alone must not allow the deletion")
			assert.Equal(t, http.StatusNoContent, del("deleter"),
				"the delete permission must allow the deletion")
		})
	}
}
