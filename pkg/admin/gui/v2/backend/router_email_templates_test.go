package backend

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// TestEmailTemplatesRouter checks the right each route of the "Email
// templates" screen requires: the rights on the rules, as the REST API of
// the templates, not those on the administration. The preview is a POST
// that writes nothing: reading the rules is enough.
func TestEmailTemplatesRouter(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	tpl := &model.EmailTemplate{Name: "fin-de-transfert", Subject: "Fin", MIMEType: "text/plain", Body: "Fichier reçu."}
	require.NoError(t, db.Insert(tpl).Run())

	router := mux.NewRouter()
	MakeEmailTemplatesRouter(router, db, testhelpers.GetTestLogger(t))

	serve := func(perms model.PermsMask, method, target string) int {
		ctx := context.WithValue(context.Background(), constants.ContextUserKey,
			&model.User{Username: "tester", Permissions: perms})
		ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

		return rec.Code
	}

	id := fmt.Sprintf("/email-templates/%d", tpl.ID)
	read := model.PermRulesRead
	write := read | model.PermRulesWrite
	del := read | model.PermRulesDelete
	admin := model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete

	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, "/email-templates"))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodGet, id))
	assert.Equal(t, http.StatusOK, serve(read, http.MethodPost, "/email-templates/preview"))
	assert.Equal(t, http.StatusForbidden, serve(admin, http.MethodGet, "/email-templates"),
		"the rights on the administration do not open it")
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/email-templates/999"))
	assert.Equal(t, http.StatusNotFound, serve(read, http.MethodGet, "/email-templates/abc"), "an identifier is a number")
	assert.Equal(t, http.StatusMethodNotAllowed, serve(del, http.MethodDelete, id), "every action is a POST")

	// The deletion comes last, once the other routes have answered.
	for _, route := range []struct {
		method, target string
		perm           model.PermsMask
		lacking        model.PermsMask
	}{
		{http.MethodGet, "/email-templates", read, 0},
		{http.MethodGet, id, read, 0},
		{http.MethodPost, "/email-templates/preview", read, admin},
		{http.MethodGet, "/email-templates/new", write, del},
		{http.MethodPost, "/email-templates/new", write, del},
		{http.MethodGet, id + "/edit", write, del},
		{http.MethodPost, id + "/edit", write, del},
		{http.MethodGet, id + "/delete", del, write},
		{http.MethodPost, id + "/delete", del, write},
	} {
		assert.Equalf(t, http.StatusForbidden, serve(route.lacking, route.method, route.target),
			"%s %s must be refused without the right", route.method, route.target)
		assert.NotEqualf(t, http.StatusForbidden, serve(route.perm, route.method, route.target),
			"%s %s must be served with the right", route.method, route.target)
	}
}
