package services

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// sampleServices is a lister with every state of the sheet: a core service
// without a catalogue entry, servers in error, offline and in an unknown
// state, given out of order.
func sampleServices() (core, servers, clients []internal.Service) {
	core = []internal.Service{
		{Name: "SNMP", State: utils.StateRunning.String()},
		{Name: "Database", State: utils.StateRunning.String()},
		{Name: "Mystery", State: utils.StateRunning.String()},
	}
	servers = []internal.Service{
		{Name: "srv-sftp-01", State: utils.StateRunning.String()},
		{Name: "srv-https-01", State: utils.StateOffline.String()},
		{Name: "srv-r66-01", State: utils.StateError.String(), Reason: "listen tcp 0.0.0.0:6666: bind: address already in use"},
		{Name: "srv-ftp-01", State: "Weird", Reason: "<raw & escaped>"},
	}
	clients = []internal.Service{{Name: "cli-sftp", State: utils.StateRunning.String()}}

	return core, servers, clients
}

func noServices() (core, servers, clients []internal.Service) {
	return []internal.Service{{Name: "Database", State: utils.StateRunning.String()}}, nil, nil
}

// manyServers is a lister with more servers than a page holds.
func manyServers() (core, servers, clients []internal.Service) {
	for i := range 30 {
		servers = append(servers, internal.Service{Name: fmt.Sprintf("srv-%02d", i+1), State: utils.StateRunning.String()})
	}

	return nil, servers, nil
}

// serve runs the page handler as an authenticated user without any right.
func serve(t *testing.T, list Lister, method, lang, target string) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "reader", Owner: "gw-test", Permissions: 0}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, lang)

	rec := httptest.NewRecorder()
	Page(testhelpers.GetTestLogger(t), list).ServeHTTP(rec, httptest.NewRequestWithContext(ctx, method, target, nil))

	return rec
}

// indexes asserts that the markers appear in the body in the given order.
func indexes(t *testing.T, body string, markers ...string) {
	t.Helper()

	last := -1

	for _, marker := range markers {
		pos := strings.Index(body, marker)
		require.NotEqualf(t, -1, pos, "%q not found in the page", marker)
		assert.Greaterf(t, pos, last, "%q is out of order", marker)
		last = pos
	}
}

// panel returns the markup of the tab panel of a group.
func panel(t *testing.T, body, groupID string) string {
	t.Helper()

	match := regexp.MustCompile(`(?s)<section class="services-panel" role="tabpanel" id="` + groupID + `".*?</section>`).FindString(body)
	require.NotEmptyf(t, match, "panel %q not found", groupID)

	return match
}

func TestPage(t *testing.T) {
	t.Parallel()

	t.Run("Given a user without any permission", func(t *testing.T) {
		t.Parallel()

		rec := serve(t, sampleServices, http.MethodGet, "fr", "/webui/services")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/html; charset=utf-8", rec.Header().Get("Content-Type"))

		body := rec.Body.String()
		assert.Contains(t, body, `<html lang="fr">`)
		assert.Contains(t, body, `<title>État des services · gw-test · Waarp Gateway</title>`)
		assert.Contains(t, body, `class="w-skip"`)
		assert.Contains(t, body, `<main class="w-main" id="contenu"`)
		assert.Regexp(t, `data-refreshed-at="\d{2}:\d{2}:\d{2}"`, body)
		assert.Regexp(t, `Actualisé à \d{2}:\d{2}:\d{2} · toutes les 10 s</span>`, body)
		assert.NotContains(t, body, "w-page-header__desc", "no sentence under the title")

		// Three tiles as tabs, the first selected; three panels, the first shown.
		assert.Equal(t, 3, strings.Count(body, `role="tab"`))
		assert.Equal(t, 1, strings.Count(body, `aria-selected="true"`))
		assert.Contains(t, body, `id="tab-core" aria-controls="core" aria-selected="true"`)
		assert.Contains(t, body, `href="/webui/services?group=servers" role="tab"`)
		assert.Equal(t, 3, strings.Count(body, `role="tabpanel"`))
		assert.Equal(t, 2, strings.Count(body, `aria-labelledby="tab-`+""+`servers" hidden`)+strings.Count(body, `aria-labelledby="tab-clients" hidden`))
		assert.Contains(t, body, `id="core" aria-labelledby="tab-core">`)

		// Each panel sorted by name, the first column named after the object.
		indexes(t, panel(t, body, "core"), `<th scope="col">Service système</th>`,
			`<code class="w-code">Database</code>`, "Mystery", `<code class="w-code">SNMP</code>`)
		indexes(t, panel(t, body, "servers"), `<th scope="col">Serveur local</th>`,
			"srv-ftp-01", "srv-https-01", "srv-r66-01", "srv-sftp-01")
		indexes(t, panel(t, body, "clients"), `<th scope="col">Client local</th>`, "cli-sftp")

		// States: shape, colour and label; the raw reason, escaped; a dash with a hidden text otherwise.
		assert.Contains(t, body, `w-status w-status--ok">En ligne<`)
		assert.Contains(t, body, `w-status w-status--off">Hors ligne<`)
		assert.Contains(t, body, `w-status w-status--ko">Erreur<`)
		assert.Contains(t, body, `w-status w-status--warn">Inconnu<`)
		assert.Contains(t, body, `<td class="w-mono">listen tcp 0.0.0.0:6666: bind: address already in use</td>`)
		assert.Contains(t, body, `<td class="w-mono">&lt;raw &amp; escaped&gt;</td>`)
		assert.Contains(t, body, `<span class="visually-hidden">Aucune raison</span>`)

		// System services: functional name with the technical one, raw name otherwise.
		assert.Contains(t, body, `<span class="w-cell-title">Base de données</span>`)
		assert.Contains(t, body, `<span class="w-cell-title w-nowrap">Mystery</span>`)

		// Tiles: the count of running services, the problems named.
		assert.Contains(t, body, "w-tile w-tile--ok")
		assert.Contains(t, body, "w-tile w-tile--ko")
		assert.Contains(t, body, "1 en erreur")
		assert.Contains(t, body, "1 hors ligne")
		assert.Contains(t, body, "1 inconnu")
		assert.Contains(t, body, "Tous en ligne")
		assert.Contains(t, body, "1 / 4 en ligne")

		// Chrome: the active entry, the breadcrumb, the documentation page.
		assert.Contains(t, body, `href="/webui/services" aria-current="page"`)
		assert.Contains(t, body, `<li><span aria-current="page">État des services</span></li>`)
		assert.Contains(t, body, "https://doc.waarp.org/waarp-gateway/latest/fr/user/webui/etat-des-services/")
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")
	})

	t.Run("Given the English language", func(t *testing.T) {
		t.Parallel()

		rec := serve(t, sampleServices, http.MethodGet, "en", "/webui/services")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<html lang="en">`)
		assert.Contains(t, body, "<title>Service status · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `w-status w-status--ok">Running<`)
		assert.Contains(t, body, `w-status w-status--warn">Unknown<`)
		assert.Contains(t, body, `<th scope="col">Local server</th>`)
		assert.Contains(t, body, `<th scope="col">Status</th>`)
		assert.Contains(t, body, "1 / 4 running")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a group in the request", func(t *testing.T) {
		t.Parallel()

		rec := serve(t, sampleServices, http.MethodGet, "fr", "/webui/services?group=servers")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `id="tab-servers" aria-controls="servers" aria-selected="true"`)
		assert.Contains(t, body, `id="servers" aria-labelledby="tab-servers">`)
		assert.Contains(t, body, `id="core" aria-labelledby="tab-core" hidden>`)

		rec = serve(t, sampleServices, http.MethodGet, "fr", "/webui/services?group=unknown")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `id="tab-core" aria-controls="core" aria-selected="true"`,
			"an unknown group falls back to the first one")
	})

	t.Run("Given more servers than a page holds", func(t *testing.T) {
		t.Parallel()

		rec := serve(t, manyServers, http.MethodGet, "fr", "/webui/services?group=servers")
		require.Equal(t, http.StatusOK, rec.Code)

		servers := panel(t, rec.Body.String(), "servers")
		assert.Equal(t, 25, strings.Count(servers, "<tr>")-1, "the first page holds 25 rows")
		assert.Contains(t, servers, "État de chaque serveur local, page 1 sur 2")
		assert.Contains(t, servers, "Serveurs locaux 1 à 25 sur 30")
		assert.Contains(t, servers, `href="/webui/services?group=servers&amp;page=2" rel="next"`)
		assert.Contains(t, servers, `<input type="hidden" name="group" value="servers">`)
		assert.Contains(t, rec.Body.String(), "30 / 30 en ligne", "the tile counts the whole group")

		rec = serve(t, manyServers, http.MethodGet, "fr", "/webui/services?group=servers&page=2")
		require.Equal(t, http.StatusOK, rec.Code)

		servers = panel(t, rec.Body.String(), "servers")
		assert.Equal(t, 5, strings.Count(servers, "<tr>")-1)
		assert.Contains(t, servers, "srv-26")
		assert.NotContains(t, servers, "srv-25")
		assert.Contains(t, servers, "Serveurs locaux 26 à 30 sur 30")
		assert.Contains(t, servers, `href="/webui/services?group=servers" rel="prev"`)

		rec = serve(t, manyServers, http.MethodGet, "fr", "/webui/services?group=servers&page=99")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, panel(t, rec.Body.String(), "servers"), "Serveurs locaux 26 à 30 sur 30",
			"a page out of range is brought back to the last one")

		rec = serve(t, manyServers, http.MethodGet, "fr", "/webui/services?group=core&page=2")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, panel(t, rec.Body.String(), "servers"), "Serveurs locaux 1 à 25 sur 30",
			"the page applies to the group shown only")
	})

	t.Run("Given no server and no client", func(t *testing.T) {
		t.Parallel()

		rec := serve(t, noServices, http.MethodGet, "fr", "/webui/services")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Equal(t, 1, strings.Count(body, `<table class="w-table">`), "only the system services have a table")
		assert.Equal(t, 2, strings.Count(body, `class="w-empty"`))
		assert.Contains(t, body, "Aucun serveur local pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/server_management">Voir les serveurs locaux</a>`)
		assert.Contains(t, body, "Aucun client local pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/local_client_management">Voir les clients locaux</a>`)
		assert.Contains(t, body, "Aucun pour l&#39;instant")
	})

	t.Run("Given a request for the state fragment", func(t *testing.T) {
		t.Parallel()

		rec := serve(t, sampleServices, http.MethodGet, "fr", "/webui/services?fragment=state&group=clients")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.True(t, strings.HasPrefix(body, `<div id="etat"`), "the fragment is the states block alone")
		assert.NotContains(t, body, "<html")
		assert.NotContains(t, body, "w-header")
		assert.Contains(t, body, `id="tab-clients" aria-controls="clients" aria-selected="true"`)
		assert.Contains(t, body, "Base de données")
	})
}
