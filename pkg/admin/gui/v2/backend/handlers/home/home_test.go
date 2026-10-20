package home

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// serve runs a page handler as an authenticated user holding the given rights.
func serve(t *testing.T, handler http.Handler, perms model.PermsMask, lang, target string) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, lang)

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, target, nil))

	return rec
}

func TestHomePage(t *testing.T) {
	t.Parallel()

	logger := testhelpers.GetTestLogger(t)
	page := Page(logger)

	t.Run("Given a user with every right", func(t *testing.T) {
		t.Parallel()

		rec := serve(t, page, model.PermAll, "fr", "/webui/home")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Accueil · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<h1 class="w-page-header__title">Accueil</h1>`)
		assert.Equal(t, 4, strings.Count(body, `<section class="w-card"`), "one card per section")
		assert.Equal(t, 14, strings.Count(body, `class="w-card w-link-card"`), "one link card per screen")
		assert.Contains(t, body, `<h2 class="w-card__title" id="home-connection">Connexions</h2>`)
		assert.Contains(t, body, `href="/webui/servers"`, "a reworked screen leads to its new page")
		assert.Contains(t, body, `href="/webui/partners"`, "the partners too (#714)")
		assert.Contains(t, body, `href="/webui/transfer_rules_management"`, "a screen not reworked yet leads to its historic page")
		assert.Contains(t, body, `<span class="w-link-card__label">Serveurs locaux</span>`)
		assert.Contains(t, body, `<span class="w-link-card__text">Points d&#39;écoute, un par protocole</span>`)

		assert.NotContains(t, body, `aria-current="page"`, "no sidebar entry is the home page, and no breadcrumb")
		assert.NotContains(t, body, "w-crumbs", "no breadcrumb: it would repeat the title")
		assert.Contains(t, body, `<div class="w-grid w-grid--4">`, "the sections side by side")
		assert.NotContains(t, body, "w-btn--primary", "no action")
		assert.Contains(t, body, `<a href="/webui/about" data-dialog="w-about">`, "the About entry opens the dialog")
		assert.Contains(t, body, `<dialog class="w-dialog w-dialog--wide" id="w-about"`)
		assert.Contains(t, body, `href="https://doc.waarp.org/waarp-gateway/latest/fr/" aria-label="Documentation"`)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a user with the servers right only", func(t *testing.T) {
		t.Parallel()

		body := serve(t, page, model.PermServersRead, "en", "/webui/home").Body.String()
		assert.Contains(t, body, "<title>Home · gw-test · Waarp Gateway</title>")
		assert.Equal(t, 2, strings.Count(body, `<section class="w-card"`), "connections and monitoring only")
		assert.Equal(t, 3, strings.Count(body, `class="w-card w-link-card"`), "servers, clients and the service status")
		assert.Contains(t, body, `<span class="w-link-card__label">Local servers</span>`)
		assert.NotContains(t, body, "Remote partners", "no partners without the partners right")
		assert.NotContains(t, body, `id="home-admin"`, "an empty section is not rendered")
		assert.Contains(t, body, "State of the services, servers and clients")
	})

	t.Run("Given a user without any right", func(t *testing.T) {
		t.Parallel()

		body := serve(t, page, 0, "fr", "/webui/home").Body.String()
		assert.Equal(t, 1, strings.Count(body, `<section class="w-card"`))
		assert.Equal(t, 1, strings.Count(body, `class="w-card w-link-card"`), "the service status alone")
		assert.Contains(t, body, `href="/webui/services"`)
	})
}

func TestAboutPage(t *testing.T) {
	t.Parallel()

	rec := serve(t, About(testhelpers.GetTestLogger(t)), 0, "fr", "/webui/about")
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "<title>À propos de Waarp Gateway · gw-test · Waarp Gateway</title>")
	assert.Contains(t, body, `<a href="/webui/home">Accueil</a>`)
	assert.Contains(t, body, "<dt>Version</dt>")
	assert.Contains(t, body, "<dt>Révision</dt>")
	assert.Contains(t, body, `<a class="w-link" href="https://www.waarp.fr" target="_blank" rel="noopener">Waarp</a> fournit un support et des services professionnels pour Waarp Gateway.`)
	assert.Contains(t, body, "Waarp Gateway est sous licence GPLv3.")
	assert.Contains(t, body, `href="https://dl.waarp.org/dist/waarp-gateway/" target="_blank" rel="noopener">Télécharger Waarp Gateway`)
	assert.Contains(t, body, `href="mailto:info@waarp.fr"`)
	assert.Equal(t, 2, strings.Count(body, "fournit un support"), "the page and the dialog of the shell")
	assert.NotContains(t, body, "¿")
}

// Every screen of the screen map has its sentence on the home page, in
// every language: the catalogue follows the map.
func TestEveryScreenHasItsSentence(t *testing.T) {
	t.Parallel()

	catalog, err := locale.LoadCatalog()
	require.NoError(t, err)

	for _, section := range common.Screens(model.PermAll) {
		for _, screen := range section.Screens {
			assert.Truef(t, catalog.Has(descKeyPrefix+screen.Key), "no sentence for the screen %q", screen.Key)
		}
	}
}
