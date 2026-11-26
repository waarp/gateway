package gui

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/emailtemplates"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// The isolated frame of the preview of an email template sends no session:
// its stylesheet, and that file only, is served without one. The other files
// of the screens lead to the sign-in page, as the screens do.
func TestStaticFilesWithoutSession(t *testing.T) {
	router := mux.NewRouter()
	AddGUIRouter(router.PathPrefix(Prefix).Subrouter(), testhelpers.GetTestLogger(t), dbtest.TestDatabase(t))

	serve := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, Prefix+path, nil))

		return rec
	}

	rec := serve(emailtemplates.PreviewStylesheet)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/css")

	for _, path := range []string{
		"/static_v2/", "/static_v2/email_templates/", "/static_v2/email_templates/email-templates.css",
		"/static_v2/i18n/fr.yaml", "/home",
	} {
		rec := serve(path)
		assert.Equalf(t, http.StatusFound, rec.Code, "GET %s", path)
		assert.Containsf(t, rec.Header().Get("Location"), "/login", "GET %s", path)
	}
}
