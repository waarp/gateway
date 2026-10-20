package servers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermServersRead
	permWrite = model.PermServersRead | model.PermServersWrite
	permAll   = model.PermServersRead | model.PermServersWrite | model.PermServersDelete

	bindReason = "listen tcp 0.0.0.0:6666: bind: address already in use"
)

// fakeControl drives services in memory: a new server is offline, a started
// one runs, except the failing one, which reports an error with its reason.
type fakeControl struct {
	mu      sync.Mutex
	states  map[int64]utils.StateCode
	reasons map[int64]string
	failing string
}

func newFakeControl() *fakeControl {
	return &fakeControl{states: map[int64]utils.StateCode{}, reasons: map[int64]string{}}
}

func (c *fakeControl) set(id int64, state utils.StateCode, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.states[id], c.reasons[id] = state, reason
}

func (c *fakeControl) control() Control {
	return Control{
		Status: func(s *model.LocalAgent) (utils.StateCode, string) {
			c.mu.Lock()
			defer c.mu.Unlock()

			if state, ok := c.states[s.ID]; ok {
				return state, c.reasons[s.ID]
			}

			return utils.StateOffline, ""
		},
		Add: func(db *database.DB, s *model.LocalAgent) error {
			if err := internal.InsertServer(db, s); err != nil {
				return err
			}

			c.set(s.ID, utils.StateOffline, "")

			return nil
		},
		Remove: func(_ context.Context, db *database.DB, s *model.LocalAgent) error {
			return internal.DeleteServer(db, s)
		},
		Start: func(_ context.Context, s *model.LocalAgent) error {
			if s.Name == c.failing {
				// As the gateway does when a listener cannot open: the service
				// stays offline without a reason, the error carries the cause.
				c.set(s.ID, utils.StateOffline, "")

				return fmt.Errorf("failed to start service: %s", bindReason) //nolint:err113 // the service's own message
			}

			c.set(s.ID, utils.StateRunning, "")

			return nil
		},
		Stop: func(_ context.Context, s *model.LocalAgent) error {
			c.set(s.ID, utils.StateOffline, "")

			return nil
		},
	}
}

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	ctl    *fakeControl
	router *mux.Router

	sftp, r66, pesit *model.LocalAgent
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), ctl: newFakeControl(), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t), s.ctl.control())

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/servers").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/servers/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/servers/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/servers/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/servers/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/servers/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/servers/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/servers/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)
	r.Path("/servers/{id:[0-9]+}/start").Methods(http.MethodPost).HandlerFunc(h.Start)
	r.Path("/servers/{id:[0-9]+}/stop").Methods(http.MethodPost).HandlerFunc(h.Stop)

	return s
}

// fixtures inserts three servers, one per state: a running SFTP server, an
// R66 over TLS server in error, a PeSIT server offline with non-default
// options.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.sftp = &model.LocalAgent{
		Name: "srv-sftp-01", Protocol: "sftp", Address: types.Addr("0.0.0.0", 10022), RootDir: "/data/sftp",
	}
	s.r66 = &model.LocalAgent{
		Name: "srv-r66-01", Protocol: "r66-tls", Address: types.Addr("0.0.0.0", 6666),
		ProtoConfig: map[string]any{"serverLogin": "waarp", "blockSize": 65536, "minTLSVersion": "v1.2"},
	}
	s.pesit = &model.LocalAgent{
		Name: "srv-pesit-01", Protocol: "pesit", Address: types.Addr("10.20.30.40", 5000), Disabled: true,
		ProtoConfig: map[string]any{"compatibilityMode": "non-standard", "disableCheckpoints": true, "maxMessageSize": 32000},
	}

	for _, agent := range []*model.LocalAgent{s.sftp, s.r66, s.pesit} {
		require.NoError(t, s.db.Insert(agent).Run())
	}

	s.ctl.set(s.sftp.ID, utils.StateRunning, "")
	s.ctl.set(s.r66.ID, utils.StateError, bindReason)
	s.ctl.set(s.pesit.ID, utils.StateOffline, "")
}

// do sends a request as a user with the given rights, in the given language.
func (s *suite) do(t *testing.T, perms model.PermsMask, lang, method, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, lang)

	var req *http.Request
	if form != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		req = httptest.NewRequestWithContext(ctx, method, target, nil)
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *suite) get(t *testing.T, perms model.PermsMask, target string) *httptest.ResponseRecorder {
	t.Helper()

	return s.do(t, perms, "fr", http.MethodGet, target, nil)
}

func (s *suite) post(t *testing.T, perms model.PermsMask, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	if form == nil {
		form = url.Values{}
	}

	return s.do(t, perms, "fr", http.MethodPost, target, form)
}

// indexes asserts that the markers appear in the body in the given order,
// each one searched after the previous one (a name also appears earlier in
// the page, in the suggestions of the search).
func indexes(t *testing.T, body string, markers ...string) {
	t.Helper()

	from := 0

	for _, marker := range markers {
		pos := strings.Index(body[from:], marker)
		require.NotEqualf(t, -1, pos, "%q not found in order in the page", marker)
		from += pos + len(marker)
	}
}

func rows(body string) int { return strings.Count(body, "<tr>") - 1 }

func (s *suite) path(agent *model.LocalAgent, suffix string) string {
	return fmt.Sprintf("/webui/servers/%d%s", agent.ID, suffix)
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given servers in every state and a user with every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, "/webui/servers")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<html lang="fr">`)
		assert.Contains(t, body, "<title>Serveurs locaux · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `class="w-skip"`)
		assert.Contains(t, body, `<main class="w-main" id="contenu"`)
		assert.Contains(t, body, `href="/webui/servers" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.Contains(t, body, `<li><span aria-current="page">Serveurs locaux</span></li>`)
		assert.Contains(t, body, "https://doc.waarp.org/waarp-gateway/latest/fr/user/webui/serveurs-locaux/")
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")

		// One coloured action, the search with its suggestions, thirteen protocol filters, the count.
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/servers/new">`)
		assert.Contains(t, body, `<input type="search" name="q" value="" placeholder="Rechercher par nom" list="servers-names"`)
		assert.Contains(t, body, `<option value="srv-pesit-01"></option><option value="srv-r66-01"></option><option value="srv-sftp-01"></option>`)
		assert.Equal(t, 13, strings.Count(body, `type="checkbox" name="protocol"`))
		assert.Contains(t, body, `<details class="w-disclose">`, "the filters are closed when none is active")
		assert.Contains(t, body, `aria-live="polite">3 serveurs</span>`)

		// Four sortable headers, sorted by name ascending; the rows by name.
		assert.Equal(t, 1, rows(body)-2, "one table")
		assert.Contains(t, body, `<th scope="col" aria-sort="ascending"><a href="/webui/servers?sort=name&amp;dir=desc">Serveur local</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/servers?sort=state&amp;dir=asc">État</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/servers?sort=protocol&amp;dir=asc">Protocole</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/servers?sort=address&amp;dir=asc">Adresse</a></th>`)
		indexes(t, body, "<tbody>", "srv-pesit-01", "srv-r66-01", "srv-sftp-01", "</tbody>")

		// States as shape + colour + label, the raw reason next to it; protocols and addresses.
		assert.Contains(t, body, `w-status w-status--ok">En ligne<`)
		assert.Contains(t, body, `w-status w-status--ko">Erreur<`)
		assert.Contains(t, body, `w-status w-status--off">Hors ligne<`)
		assert.Contains(t, body, `<span class="w-cell-sub w-mono">`+bindReason+`</span>`)
		assert.Contains(t, body, "<td>R66-TLS</td>")
		assert.Contains(t, body, "<td>PeSIT</td>")
		assert.Contains(t, body, `<td class="w-mono w-nowrap">0.0.0.0:10022</td>`)
		assert.Contains(t, body, `<td class="w-mono w-nowrap">10.20.30.40:5000</td>`)
		assert.Contains(t, body, `Serveurs locaux, page 1 sur 1, triés par nom`)

		// The "⋯" menu: stop the running one, start the others, the historic
		// pages, edit, delete last.
		assert.Equal(t, 3, strings.Count(body, `class="w-icon-btn w-icon-btn--round"`))
		assert.Contains(t, body, `href="`+s.path(s.sftp, "")+`">`)
		assert.Contains(t, body, `<form method="post" action="`+s.path(s.sftp, "/stop")+`">`)
		assert.Contains(t, body, `<form method="post" action="`+s.path(s.r66, "/start")+`">`)
		assert.Contains(t, body, `<form method="post" action="`+s.path(s.pesit, "/start")+`">`)
		assert.Contains(t, body, fmt.Sprintf(`href="/webui/server_authentication?serverID=%d"`, s.sftp.ID))
		assert.Contains(t, body, fmt.Sprintf(`href="/webui/local_account_management?serverID=%d"`, s.sftp.ID))
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/edit")+`">`)
		assert.Contains(t, body, `<hr>`)
		assert.Contains(t, body, `class="is-danger" href="`+s.path(s.sftp, "/delete")+`">`)
		assert.Contains(t, body, `name="size"`, "the rows per page are offered as soon as there are rows")
		assert.Contains(t, body, "Serveurs 1 à 3 sur 3")
	})

	t.Run("Given a user with the read right only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permRead, "/webui/servers")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "w-btn--primary", "no coloured action without the write right")
		assert.NotContains(t, body, "/start", "no start without the write right")
		assert.NotContains(t, body, "/stop")
		assert.NotContains(t, body, "/edit")
		assert.NotContains(t, body, "/delete")
		assert.Contains(t, body, "Voir le détail")
		assert.Contains(t, body, "Authentification")
		assert.Contains(t, body, "Comptes locaux")
	})

	t.Run("Given the English language", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.do(t, permAll, "en", http.MethodGet, "/webui/servers", nil)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<html lang="en">`)
		assert.Contains(t, body, "<title>Local servers · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `>Local server</a></th>`)
		assert.Contains(t, body, `w-status w-status--ok">Running<`)
		assert.Contains(t, body, "3 servers</span>")
		assert.Contains(t, body, "Add a server")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a sort in the request", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, "/webui/servers?sort=state").Body.String()
		indexes(t, body, "<tbody>", "srv-sftp-01", "srv-pesit-01", "srv-r66-01", "</tbody>")
		assert.Contains(t, body, `<th scope="col" aria-sort="ascending"><a href="/webui/servers?sort=state&amp;dir=desc">État</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/servers?sort=name&amp;dir=asc">Serveur local</a></th>`)
		assert.Contains(t, body, "triés par état")

		body = s.get(t, permAll, "/webui/servers?sort=state&dir=desc").Body.String()
		indexes(t, body, "<tbody>", "srv-r66-01", "srv-pesit-01", "srv-sftp-01", "</tbody>")
		assert.Contains(t, body, `aria-sort="descending"`)

		body = s.get(t, permAll, "/webui/servers?sort=protocol").Body.String()
		indexes(t, body, "<tbody>", "srv-pesit-01", "srv-r66-01", "srv-sftp-01", "</tbody>")

		body = s.get(t, permAll, "/webui/servers?sort=address").Body.String()
		indexes(t, body, "<tbody>", "srv-r66-01", "srv-sftp-01", "srv-pesit-01", "</tbody>")

		body = s.get(t, permAll, "/webui/servers?sort=name&dir=desc").Body.String()
		indexes(t, body, "<tbody>", "srv-sftp-01", "srv-r66-01", "srv-pesit-01", "</tbody>")

		body = s.get(t, permAll, "/webui/servers?sort=bogus&dir=sideways").Body.String()
		indexes(t, body, "<tbody>", "srv-pesit-01", "srv-r66-01", "srv-sftp-01", "</tbody>")
		assert.Contains(t, body, `aria-sort="ascending"><a href="/webui/servers?sort=name&amp;dir=desc">`)
	})

	t.Run("Given a search and filters", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, "/webui/servers?q=SRV-s").Body.String()
		assert.Equal(t, 1, rows(body), "the search is a case-insensitive prefix")
		assert.Contains(t, body, "srv-sftp-01")
		assert.Contains(t, body, `aria-live="polite">3 serveurs · 1 affiché</span>`)
		assert.Contains(t, body, `<input type="search" name="q" value="SRV-s"`)
		assert.Contains(t, body, `href="/webui/servers?q=SRV-s&amp;sort=state&amp;dir=asc"`, "the sort links keep the search")

		body = s.get(t, permAll, "/webui/servers?protocol=sftp&protocol=pesit").Body.String()
		assert.Equal(t, 2, rows(body))
		assert.Contains(t, body, `<details class="w-disclose" open>`)
		assert.Contains(t, body, `name="protocol" value="sftp" checked>`)
		assert.Contains(t, body, `name="protocol" value="pesit" checked>`)
		assert.Contains(t, body, `name="protocol" value="r66-tls">`)
		assert.Contains(t, body, "Filtres · 2")
		assert.Contains(t, body, `3 serveurs · 2 affichés`)
		assert.Contains(t, body, `href="/webui/servers?protocol=sftp%2Cpesit&amp;sort=name&amp;dir=desc"`)
		assert.Contains(t, body, `href="/webui/servers/new?ret=protocol%3Dsftp%252Cpesit"`, "the action pages carry the list state")

		body = s.get(t, permAll, "/webui/servers?protocol=bogus").Body.String()
		assert.Equal(t, 3, rows(body), "an unknown protocol is ignored")

		body = s.get(t, permAll, "/webui/servers?q=zzz").Body.String()
		assert.NotContains(t, body, "<table")
		assert.Contains(t, body, `<p class="w-empty__title">Aucun serveur ne correspond à « zzz »</p>`)
		assert.Contains(t, body, `href="/webui/servers?reset=1">Effacer la recherche</a>`, "explicit, so that the kept state is cleared")
		assert.Contains(t, body, `3 serveurs · 0 affichés`)

		body = s.get(t, permAll, "/webui/servers?protocol=ftp").Body.String()
		assert.Contains(t, body, `<p class="w-empty__title">Aucun serveur ne correspond aux filtres</p>`)
		assert.Contains(t, body, `href="/webui/servers?reset=1">Réinitialiser</a>`)
	})

	t.Run("Given no server at all", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		body := s.get(t, permAll, "/webui/servers").Body.String()
		assert.NotContains(t, body, "w-toolbar")
		assert.Contains(t, body, `<p class="w-empty__title">Aucun serveur local pour l&#39;instant</p>`)
		assert.Contains(t, body, `<p class="w-empty__text">Un serveur local est un point d&#39;écoute`)
		assert.Equal(t, 2, strings.Count(body, `href="/webui/servers/new"`), "the header and the empty state lead to the creation")

		body = s.get(t, permRead, "/webui/servers").Body.String()
		assert.NotContains(t, body, "/webui/servers/new", "no creation without the write right")
	})

	t.Run("Given more servers than a page holds", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		for i := range 30 {
			agent := &model.LocalAgent{Name: fmt.Sprintf("srv-%02d", i+1), Protocol: "sftp", Address: types.Addr("0.0.0.0", uint16(20000+i))}
			require.NoError(t, s.db.Insert(agent).Run())
		}

		body := s.get(t, permAll, "/webui/servers").Body.String()
		assert.Equal(t, 10, rows(body))
		assert.Contains(t, body, "Serveurs 1 à 10 sur 30")
		assert.Contains(t, body, "Serveurs locaux, page 1 sur 3, triés par nom")
		assert.Contains(t, body, `href="/webui/servers?page=2" rel="next"`)
		assert.Contains(t, body, `<select class="w-select" name="size"><option value="10" selected>10</option><option value="25">25</option>`)

		body = s.get(t, permAll, "/webui/servers?page=2&size=10").Body.String()
		indexes(t, body, "<tbody>", "srv-11", "srv-20", "</tbody>")
		assert.NotContains(t, body, "srv-10<")
		assert.Contains(t, body, `href="/webui/servers" rel="prev"`)
		assert.Contains(t, body, `href="/webui/servers?page=3" rel="next"`)
		assert.Contains(t, body, `href="/webui/servers/new?ret=page%3D2"`)

		body = s.get(t, permAll, "/webui/servers?size=25").Body.String()
		assert.Equal(t, 25, rows(body))
		assert.Contains(t, body, "Serveurs 1 à 25 sur 30")
		assert.Contains(t, body, `<option value="25" selected>`)
		assert.Contains(t, body, `href="/webui/servers?page=2&amp;size=25" rel="next"`)
		assert.Contains(t, body, `<input type="hidden" name="size" value="25">`, "the toolbar keeps the rows per page")

		body = s.get(t, permAll, "/webui/servers?page=99").Body.String()
		assert.Contains(t, body, "Serveurs 21 à 30 sur 30", "a page out of range is brought back to the last one")

		body = s.get(t, permAll, "/webui/servers?page=0&size=7").Body.String()
		assert.Contains(t, body, "Serveurs 1 à 10 sur 30", "an invalid page and size fall back to the default")
	})

	t.Run("Given a list state kept for the session", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		serveWith := func(target, cookie string) *httptest.ResponseRecorder {
			user := &model.User{Username: "tester", Owner: "gw-test", Permissions: permAll}
			ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
			ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")
			req := httptest.NewRequestWithContext(ctx, http.MethodGet, target, nil)

			if cookie != "" {
				req.Header.Set("Cookie", cookie)
			}

			rec := httptest.NewRecorder()
			s.router.ServeHTTP(rec, req)

			return rec
		}

		rec := serveWith("/webui/servers", "")
		require.Equal(t, http.StatusOK, rec.Code, "nothing kept yet: the bare list")
		assert.Empty(t, rec.Header().Values("Set-Cookie"))

		rec = serveWith("/webui/servers?q=srv&sort=state", "")
		require.Equal(t, http.StatusOK, rec.Code)
		cookies := rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, "webui_servers_list", cookies[0].Name)
		assert.Equal(t, "dir%3Dasc%26q%3Dsrv%26sort%3Dstate", cookies[0].Value)
		assert.Equal(t, "/webui/servers", cookies[0].Path)
		assert.True(t, cookies[0].HttpOnly)
		assert.Zero(t, cookies[0].MaxAge, "a session cookie")
		kept := cookies[0].Name + "=" + cookies[0].Value

		rec = serveWith("/webui/servers", kept)
		require.Equal(t, http.StatusSeeOther, rec.Code, "the bare list comes back as it was left")
		assert.Equal(t, "/webui/servers?dir=asc&q=srv&sort=state", rec.Header().Get("Location"))

		rec = serveWith("/webui/servers?done=created&name=srv-x", kept)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/servers?dir=asc&done=created&name=srv-x&q=srv&sort=state", rec.Header().Get("Location"),
			"the result of an action travels with the restored state")

		rec = serveWith("/webui/servers?protocol=sftp", kept)
		require.Equal(t, http.StatusOK, rec.Code, "an explicit address wins")
		assert.Equal(t, "protocol%3Dsftp", rec.Result().Cookies()[0].Value)

		body := serveWith("/webui/servers?q=zzz", kept).Body.String()
		assert.Contains(t, body, `href="/webui/servers?reset=1">Effacer la recherche</a>`, "clearing is explicit")
		assert.Contains(t, body, `href="/webui/servers?reset=1">Réinitialiser</a>`)

		rec = serveWith("/webui/servers?reset=1", kept)
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, 3, rows(rec.Body.String()), "the default list")
		cookies = rec.Result().Cookies()
		require.Len(t, cookies, 1)
		assert.Equal(t, -1, cookies[0].MaxAge, "the kept state is forgotten")

		rec = serveWith("/webui/servers", "webui_servers_list=%ZZ")
		assert.Equal(t, http.StatusOK, rec.Code, "an unreadable cookie is ignored")
	})

	t.Run("Given the result of an action in the request", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, "/webui/servers?done=created&name=srv-x").Body.String()
		assert.Contains(t, body, `<div class="w-toast w-toast--ok" role="status">`)
		assert.Contains(t, body, "<strong>Serveur « srv-x » créé</strong>")

		body = s.get(t, permAll, "/webui/servers?failed=start&name=srv-x&reason=boom").Body.String()
		assert.Contains(t, body, `<div class="w-toast w-toast--ko" role="alert">`)
		assert.Contains(t, body, "<strong>Le serveur « srv-x » n&#39;a pas démarré</strong><span>boom</span>")

		body = s.get(t, permAll, "/webui/servers?done=bogus&name=srv-x").Body.String()
		assert.NotContains(t, body, "w-toast ", "an unknown result shows nothing")
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	require.NoError(t, s.db.Insert(&model.LocalAccount{LocalAgentID: s.pesit.ID, Login: "partner"}).Run())

	t.Run("Given a PeSIT server with options", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(s.pesit, "")+"?ret=dir%3Ddesc%26q%3Dsrv%26sort%3Dstate")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Serveur « srv-pesit-01 » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/servers?dir=desc&amp;q=srv&amp;sort=state">`, "back to the list as the user left it")
		assert.Contains(t, body, `<a href="/webui/servers?dir=desc&amp;q=srv&amp;sort=state">Serveurs locaux</a>`, "the breadcrumb too")
		assert.Contains(t, body, `href="`+s.path(s.pesit, "/edit")+`?ret=dir%3Ddesc%26q%3Dsrv%26sort%3Dstate">`)
		assert.Contains(t, body, `w-status w-status--off">Hors ligne<`)
		assert.Contains(t, body, "<dt>Protocole</dt><dd>PeSIT</dd>")
		assert.Contains(t, body, `<dt>Adresse d&#39;écoute</dt><dd><code class="w-code">10.20.30.40:5000</code></dd>`)
		assert.Contains(t, body, "<dt>Démarrage automatique</dt><dd>Non</dd>")
		assert.Contains(t, body, "<dt>Comptes locaux</dt><dd>1</dd>")
		assert.Contains(t, body, `<dt>Dossier racine</dt><dd><span class="w-muted">Non défini</span></dd>`)
		assert.Contains(t, body, "Options du protocole PeSIT")
		assert.Contains(t, body, "<dt>Reprise des transferts interrompus</dt><dd>Désactivé</dd>",
			"the module disables the restart along with the checkpoints: the page shows what is stored")
		assert.Contains(t, body, "<dt>Points de synchronisation</dt><dd>Désactivé</dd>")
		assert.NotContains(t, body, "Intervalle entre deux points", "no checkpoint detail when they are disabled")
		assert.Contains(t, body, `<dt>Taille maximale d&#39;un message</dt><dd><code class="w-code">32000</code></dd>`)
		assert.Contains(t, body, "<dt>Mode PeSIT</dt><dd>Non standard</dd>")
		assert.NotContains(t, body, "Version TLS minimale", "no TLS options for a plain PeSIT server")
		assert.NotContains(t, body, "Voir le détail", "no link to itself")
		assert.Contains(t, body, `action="`+s.path(s.pesit, "/start")+`?ret=dir%3Ddesc%26q%3Dsrv%26sort%3Dstate"`)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given an R66 over TLS server", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.path(s.r66, "")).Body.String()
		assert.Contains(t, body, `<dd><span class="w-status w-status--ko">Erreur</span> <code class="w-code">`+bindReason+`</code></dd>`)
		assert.Contains(t, body, "Options du protocole R66-TLS")
		assert.Contains(t, body, `<dt>Identifiant attendu du partenaire</dt><dd><code class="w-code">waarp</code></dd>`)
		assert.Contains(t, body, `<dt>Taille d&#39;un bloc de données</dt><dd><code class="w-code">65536</code></dd>`)
		assert.Contains(t, body, "<dt>Hachage en fin de transfert</dt><dd>Vérifié</dd>")
		assert.Contains(t, body, "<dt>Hachage de chaque bloc</dt><dd>Non vérifié</dd>")
		assert.Contains(t, body, "<dt>Version TLS minimale</dt><dd>TLS 1.2</dd>")
		assert.Contains(t, body, "<dt>Suites de chiffrement TLS</dt><dd>Toutes les suites sûres de la passerelle</dd>")
		assert.NotContains(t, body, "/edit", "no edition without the write right")
		assert.Contains(t, body, "<dt>Dossier racine</dt>")
	})

	t.Run("Given an SFTP server with default algorithms", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.sftp, "")).Body.String()
		assert.Contains(t, body, `<dt>Dossier racine</dt><dd><code class="w-code">/data/sftp</code></dd>`)
		assert.Contains(t, body, `<dt>Dossier de réception</dt><dd><code class="w-code">in</code></dd>`, "the model fills the default directories")
		assert.Contains(t, body, "<dt>Algorithmes d&#39;échange de clés</dt><dd>Tous les algorithmes de la passerelle</dd>")
	})

	t.Run("Given an unknown server", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, "/webui/servers/999?ret=q%3Dsrv")
		require.Equal(t, http.StatusNotFound, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Serveur introuvable · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<p class="w-empty__title">Ce serveur n&#39;existe pas ou a été supprimé.</p>`)
		assert.Contains(t, body, `href="/webui/servers?q=srv">`)
	})
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	t.Run("Given the empty form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.get(t, permWrite, "/webui/servers/new")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Nouveau serveur · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<form class="w-form" method="post" action="/webui/servers/new" novalidate>`)
		assert.Contains(t, body, `<option value="" selected>Choisissez un protocole</option>`)
		assert.Equal(t, 13, strings.Count(protocolSelect(body), `<option value="`)-1, "thirteen protocols after the placeholder")
		assert.Contains(t, body, `<input class="w-input" id="name" name="name" value="" required`)
		assert.Contains(t, body, `id="port" name="port" value="" required type="number" inputmode="numeric" step="1" min="1" max="65535"`)
		assert.Contains(t, body, `<div class="w-card" data-case="r66 r66-tls">`)
		assert.Contains(t, body, `<div class="w-card" data-case="sftp">`)
		assert.Contains(t, body, `<option value="v1.2" selected>TLS 1.2</option>`)
		assert.Contains(t, body, `data-case="v1.0 v1.1 v1.2" data-of="tls-r66"`)
		assert.Contains(t, body, `<option value="standard" selected>Standard</option><option value="non-standard">Non standard</option>`)
		assert.Contains(t, body, `<option value="Optional" selected>TLS explicite facultatif</option>`)
		assert.Contains(t, body, `<option value="" selected>Aucune signature</option>`)
		assert.Contains(t, body, `name="sftp_kex" multiple size="5" data-dual="Disponibles|Retenus|Ajouter|Retirer"`)
		assert.Contains(t, body, `<option value="TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256">49195 - TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256</option>`,
			"a suite is named by its decimal code then its name")
		assert.Less(t, strings.Index(body, `>52393 - `), strings.Index(body, `>49195 - `), "the suites are sorted by decreasing code")
		assert.Contains(t, body, `(non sécurisée)</option>`, "the insecure suites are flagged")
		assert.NotContains(t, body, `TLS_AES_128_GCM_SHA256`, "the TLS 1.3 suites are not configurable")
		assert.Contains(t, body, `>Valider</button>`)
		assert.NotContains(t, body, "w-banner")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a valid form with the options of other protocols", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/servers/new?ret=sort%3Dstate%26dir%3Ddesc", url.Values{
			"name": {" srv-new "}, "protocol": {"r66-tls"}, "host": {"0.0.0.0"}, "port": {"6670"}, "root_dir": {"/data/new"},
			"r66_login": {"waarp2"}, "r66_block": {"32768"}, "r66_check_block_hash": {"1"},
			"r66_min_tls": {"v1.3"}, "r66_tls_ciphers": {"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
			"pesit_checkpoint_size": {"not a number"}, "sftp_kex": {"curve25519-sha256"}, "ftps_min_tls": {"v1.0"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/servers?dir=desc&done=created&name=srv-new&sort=state", rec.Header().Get("Location"))

		agent, err := internal.GetServer(s.db, "srv-new")
		require.NoError(t, err)
		assert.Equal(t, "r66-tls", agent.Protocol)
		assert.Equal(t, "0.0.0.0:6670", agent.Address.String())
		assert.Equal(t, "/data/new", agent.RootDir)
		assert.Equal(t, "in", agent.ReceiveDir, "the model fills the default directories")
		assert.False(t, agent.Disabled)
		assert.Equal(t, "waarp2", cfgString(agent.ProtoConfig, "serverLogin"))
		assert.Equal(t, "32768", cfgString(agent.ProtoConfig, "blockSize"))
		assert.True(t, cfgBool(agent.ProtoConfig, "checkBlockHash"))
		assert.False(t, cfgBool(agent.ProtoConfig, "noFinalHash"))
		assert.Equal(t, "v1.3", cfgString(agent.ProtoConfig, "minTLSVersion"))
		assert.Equal(t, []string{"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256", "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"}, cfgList(agent.ProtoConfig, "cipherSuites"))
		assert.NotContains(t, agent.ProtoConfig, "checkpointSize", "the options of the other protocols are ignored")
		assert.NotContains(t, agent.ProtoConfig, "keyExchanges")

		body := s.get(t, permWrite, "/webui/servers?done=created&name=srv-new").Body.String()
		assert.Contains(t, body, "<strong>Serveur « srv-new » créé</strong>")
		assert.Contains(t, body, `w-status w-status--off">Hors ligne<`, "a new server is not started")
	})

	t.Run("Given an SFTP server with chosen algorithms", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/servers/new", url.Values{
			"name": {"srv-sftp"}, "protocol": {"sftp"}, "host": {"0.0.0.0"}, "port": {"10023"}, "no_autostart": {"1"},
			"sftp_kex": {"curve25519-sha256@libssh.org", ""}, "sftp_macs": {"hmac-sha2-256-etm@openssh.com"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		agent, err := internal.GetServer(s.db, "srv-sftp")
		require.NoError(t, err)
		assert.True(t, agent.Disabled)
		assert.Equal(t, []string{"curve25519-sha256@libssh.org"}, cfgList(agent.ProtoConfig, "keyExchanges"))
		assert.Equal(t, []string{"hmac-sha2-256-etm@openssh.com"}, cfgList(agent.ProtoConfig, "macs"))
		assert.NotContains(t, agent.ProtoConfig, "ciphers", "an empty list means every algorithm")
	})

	t.Run("Given an invalid form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/servers/new", url.Values{
			"name": {""}, "protocol": {"bogus"}, "host": {""}, "port": {"70000"}, "root_dir": {"/kept"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<div class="w-banner w-banner--ko" role="alert">`)
		assert.Contains(t, body, "<strong>Le serveur n&#39;a pas été enregistré.</strong>Corrigez les champs signalés ci-dessous.")
		assert.Equal(t, 1, strings.Count(body, "Ce champ est obligatoire."), "the name; the host is optional")
		assert.Contains(t, body, `id="name" name="name" value="" required aria-invalid="true" aria-describedby="name-e"`)
		assert.Contains(t, body, "Indiquez un port entre 1 et 65535.")
		assert.Contains(t, body, `value="70000"`, "the values are kept")
		assert.Contains(t, body, `value="/kept"`)
		assert.Contains(t, body, "Choisissez le protocole du serveur.")
		assert.Contains(t, body, `<option value="" selected>Choisissez un protocole</option>`)

		count, err := s.db.Count(&model.LocalAgent{}).Run()
		require.NoError(t, err)
		assert.Zero(t, count, "nothing is saved")
	})

	t.Run("Given a number that is not one", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/servers/new", url.Values{
			"name": {"srv-pesit"}, "protocol": {"pesit"}, "host": {"0.0.0.0"}, "port": {"5001"},
			"pesit_checkpoint_size": {"12abc"}, "pesit_checkpoint_window": {"300"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Equal(t, 2, strings.Count(body, "Indiquez un nombre entier."), "the size and the window (above 255)")
		assert.Contains(t, body, `value="12abc"`)
		assert.Contains(t, body, `<option value="pesit" selected>`)
	})

	t.Run("Given a name or an address already used", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/servers/new", url.Values{
			"name": {"srv-sftp-01"}, "protocol": {"sftp"}, "host": {"0.0.0.0"}, "port": {"30000"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre serveur porte déjà ce nom. Choisissez-en un autre.")
		assert.NotContains(t, rec.Body.String(), "already exist", "never the model&#39;s own message when a field carries it")

		rec = s.post(t, permWrite, "/webui/servers/new", url.Values{
			"name": {"srv-other"}, "protocol": {"sftp"}, "host": {"0.0.0.0"}, "port": {"10022"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre serveur écoute déjà sur cette adresse et ce port.")
	})
}

// protocolSelect returns the markup of the protocol select.
func protocolSelect(body string) string {
	start := strings.Index(body, `id="protocol"`)
	end := strings.Index(body[start:], "</select>") + start

	return body[start:end]
}

func TestEditServer(t *testing.T) {
	t.Parallel()

	t.Run("Given the form of an existing server", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, s.path(s.r66, "/edit")+"?ret=page%3D2")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Modifier « srv-r66-01 » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `action="`+s.path(s.r66, "/edit")+`?ret=page%3D2" novalidate>`)
		assert.Contains(t, body, `href="`+s.path(s.r66, "")+`?ret=page%3D2">`, "cancel goes back to the server")
		assert.Contains(t, body, `id="name" name="name" value="srv-r66-01"`)
		assert.Contains(t, body, `<option value="r66-tls" selected>R66-TLS</option>`)
		assert.Contains(t, body, `id="host" name="host" value="0.0.0.0"`)
		assert.Contains(t, body, `id="port" name="port" value="6666"`)
		assert.Contains(t, body, `id="r66_login" name="r66_login" value="waarp"`)
		assert.Contains(t, body, `id="r66_block" name="r66_block" value="65536"`)
		assert.Contains(t, body, `id="r66_min_tls" name="r66_min_tls" data-switch="tls-r66" aria-describedby="r66_min_tls-h"><option value="v1.0">TLS 1.0</option><option value="v1.1">TLS 1.1</option><option value="v1.2" selected>`)
		assert.Contains(t, body, `name="no_autostart" value="1"> `, "autostart is on")

		body = s.get(t, permWrite, s.path(s.pesit, "/edit")).Body.String()
		assert.Contains(t, body, `name="no_autostart" value="1" checked>`)
		assert.Contains(t, body, `name="pesit_disable_checkpoints" value="1" checked>`)
		assert.Contains(t, body, `<option value="non-standard" selected>Non standard</option>`)
		assert.Contains(t, body, `id="pesit_max_message" name="pesit_max_message" value="32000"`)
	})

	t.Run("Given a valid change", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, s.path(s.r66, "/edit")+"?ret=page%3D2", url.Values{
			"name": {"srv-r66-02"}, "protocol": {"r66"}, "host": {"127.0.0.1"}, "port": {"6667"},
			"r66_no_final_hash": {"1"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/servers?done=updated&name=srv-r66-02&page=2", rec.Header().Get("Location"))

		agent, err := internal.GetServerByID(s.db, s.r66.ID)
		require.NoError(t, err)
		assert.Equal(t, "srv-r66-02", agent.Name)
		assert.Equal(t, "r66", agent.Protocol)
		assert.Equal(t, "127.0.0.1:6667", agent.Address.String())
		assert.True(t, cfgBool(agent.ProtoConfig, "noFinalHash"))
		assert.NotContains(t, agent.ProtoConfig, "serverLogin", "the options are those of the form")
		assert.NotContains(t, agent.ProtoConfig, "minTLSVersion", "no TLS option for a plain R66 server")
	})

	t.Run("Given a server listening on every interface", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		agent := &model.LocalAgent{Name: "srv-any", Protocol: "pesit-tls", Address: types.Addr("", 1765)}
		require.NoError(t, s.db.Insert(agent).Run())

		body := s.get(t, permWrite, s.path(agent, "/edit")).Body.String()
		assert.Contains(t, body, `id="host" name="host" value=""`)
		assert.Contains(t, body, `id="port" name="port" value="1765"`)

		rec := s.post(t, permWrite, s.path(agent, "/edit"), url.Values{
			"name": {"srv-any"}, "protocol": {"pesit-tls"}, "host": {"10.0.0.5"}, "port": {"1765"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		saved, err := internal.GetServerByID(s.db, agent.ID)
		require.NoError(t, err)
		assert.Equal(t, "10.0.0.5:1765", saved.Address.String())

		rec = s.post(t, permWrite, s.path(agent, "/edit"), url.Values{"name": {"srv-any"}, "protocol": {"pesit-tls"}, "host": {""}, "port": {"1765"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, "an empty host is accepted: every interface")
		assert.Contains(t, s.get(t, permWrite, "/webui/servers").Body.String(), `<td class="w-mono w-nowrap">*:1765</td>`)
		assert.Contains(t, s.get(t, permWrite, s.path(agent, "")).Body.String(), `<code class="w-code">*:1765</code>`)

		rec = s.post(t, permWrite, s.path(agent, "/edit"), url.Values{"name": {"srv-any"}, "protocol": {"pesit-tls"}, "host": {"10.0.0.5"}, "port": {"1765"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		body = s.get(t, permWrite, s.path(agent, "/edit")).Body.String()
		assert.Contains(t, body, `id="host" name="host" value="10.0.0.5"`, "the host comes back in the form")
	})

	t.Run("Given a refused change", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, s.path(s.r66, "/edit"), url.Values{
			"name": {"srv-sftp-01"}, "protocol": {"r66-tls"}, "host": {"0.0.0.0"}, "port": {"6666"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre serveur porte déjà ce nom.")
		assert.Contains(t, rec.Body.String(), `value="srv-sftp-01"`, "the refused value is kept")

		agent, err := internal.GetServerByID(s.db, s.r66.ID)
		require.NoError(t, err)
		assert.Equal(t, "srv-r66-01", agent.Name, "nothing changed")
	})

	t.Run("Given an unknown server", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		assert.Equal(t, http.StatusNotFound, s.get(t, permWrite, "/webui/servers/999/edit").Code)
		assert.Equal(t, http.StatusNotFound, s.post(t, permWrite, "/webui/servers/999/edit", nil).Code)
	})
}

func TestDeleteServer(t *testing.T) {
	t.Parallel()

	t.Run("Given a server that can be deleted", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, s.path(s.sftp, "/delete")+"?ret=q%3Dsrv")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Supprimer le serveur « srv-sftp-01 » ? · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Ses comptes locaux et leurs moyens d&#39;authentification seront supprimés avec lui.")
		assert.Contains(t, body, "Il est en ligne : il sera arrêté.")
		assert.Contains(t, body, `<dt>Serveur</dt><dd>srv-sftp-01</dd>`)
		assert.Contains(t, body, `<dt>Comptes locaux</dt><dd>0</dd>`)
		assert.Contains(t, body, `<form class="servers-form-actions" method="post" action="`+s.path(s.sftp, "/delete")+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, `<button class="w-btn w-btn--danger" type="submit" aria-label="Supprimer le serveur « srv-sftp-01 » ?">Supprimer</button>`)
		assert.NotContains(t, body, "w-banner")

		rec = s.post(t, permAll, s.path(s.sftp, "/delete")+"?ret=q%3Dsrv", nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/servers?done=deleted&name=srv-sftp-01&q=srv", rec.Header().Get("Location"))

		_, err := internal.GetServerByID(s.db, s.sftp.ID)
		assert.True(t, database.IsNotFound(err), "the server is gone")

		body = s.get(t, permAll, s.path(s.pesit, "/delete")).Body.String()
		assert.NotContains(t, body, "Il est en ligne", "an offline server is not stopped")
	})

	t.Run("Given a server used by a running transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		account := &model.LocalAccount{LocalAgentID: s.pesit.ID, Login: "partner"}
		require.NoError(t, s.db.Insert(account).Run())

		rule := &model.Rule{Name: "receive", IsSend: false, Path: "receive"}
		require.NoError(t, s.db.Insert(rule).Run())
		require.NoError(t, s.db.Insert(&model.Transfer{RuleID: rule.ID, LocalAccountID: account.NullableID(), DestFilename: "file"}).Run())

		rec := s.get(t, permAll, s.path(s.pesit, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<div class="w-banner w-banner--ko" role="alert">`)
		assert.Contains(t, body, "<strong>Ce serveur ne peut pas être supprimé.</strong>Des transferts en cours l&#39;utilisent.")
		assert.Contains(t, body, `href="/webui/transfer_monitoring">`)
		assert.Contains(t, body, "Voir les transferts en cours")
		assert.NotContains(t, body, "w-btn--danger", "no deletion offered")
		assert.NotContains(t, body, "cannot be deleted", "never the model&#39;s own message")

		rec = s.post(t, permAll, s.path(s.pesit, "/delete"), nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce serveur ne peut pas être supprimé.")

		_, err := internal.GetServerByID(s.db, s.pesit.ID)
		require.NoError(t, err, "the server is still there")
	})

	t.Run("Given an unknown server", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, "/webui/servers/999/delete").Code)
		assert.Equal(t, http.StatusNotFound, s.post(t, permAll, "/webui/servers/999/delete", nil).Code)
	})
}

func TestStartStop(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.ctl.failing = s.r66.Name

	t.Run("Given an offline server", func(t *testing.T) {
		rec := s.post(t, permWrite, s.path(s.pesit, "/start")+"?ret=dir%3Ddesc%26q%3Dsrv%26sort%3Dstate", nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/servers?dir=desc&done=started&name=srv-pesit-01&q=srv&sort=state", rec.Header().Get("Location"),
			"back to the list as the user left it")

		state, _ := s.ctl.control().Status(s.pesit)
		assert.Equal(t, utils.StateRunning, state)

		rec = s.post(t, permWrite, s.path(s.pesit, "/stop"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/servers?done=stopped&name=srv-pesit-01", rec.Header().Get("Location"))

		state, _ = s.ctl.control().Status(s.pesit)
		assert.Equal(t, utils.StateOffline, state)
	})

	t.Run("Given a server that fails to start", func(t *testing.T) {
		rec := s.post(t, permWrite, s.path(s.r66, "/start"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)

		location, err := url.Parse(rec.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "/webui/servers", location.Path)
		assert.Equal(t, "start", location.Query().Get("failed"))
		assert.Equal(t, "srv-r66-01", location.Query().Get("name"))
		assert.Equal(t, "failed to start service: "+bindReason, location.Query().Get("reason"),
			"the cause of the failed start, as the service gives none")

		body := s.get(t, permWrite, rec.Header().Get("Location")).Body.String()
		assert.Contains(t, body, `<div class="w-toast w-toast--ko" role="alert">`)
		assert.Contains(t, body, "<strong>Le serveur « srv-r66-01 » n&#39;a pas démarré</strong><span>failed to start service: "+bindReason+"</span>")
	})

	t.Run("Given an unknown server", func(t *testing.T) {
		assert.Equal(t, http.StatusNotFound, s.post(t, permWrite, "/webui/servers/999/start", nil).Code)
		assert.Equal(t, http.StatusNotFound, s.post(t, permWrite, "/webui/servers/999/stop", nil).Code)
	})
}

func TestListQuery(t *testing.T) {
	t.Parallel()

	q := parseListQuery(url.Values{"q": {" srv "}, "protocol": {"sftp,bogus", "pesit"}, "sort": {"address"}, "dir": {"desc"}, "page": {"3"}, "size": {"50"}})
	assert.Equal(t, listQuery{Q: "srv", Protocols: []string{"sftp", "pesit"}, Sort: "address", Dir: "desc", Page: 3, Size: 50}, q)
	assert.Equal(t, "dir=desc&page=3&protocol=sftp%2Cpesit&q=srv&size=50&sort=address", q.encode(q.Page))
	assert.Equal(t, map[string]string{"sort": "address", "dir": "desc", "size": "50"}, q.toolbarHidden())
	assert.Equal(t, map[string]string{"q": "srv", "protocol": "sftp,pesit", "sort": "address", "dir": "desc"}, q.pagerHidden())
	assert.Equal(t, q, retOf(q.encode(q.Page)), "the carried state reads back")

	dflt := parseListQuery(url.Values{"sort": {"name"}, "dir": {"asc"}, "size": {"10"}, "page": {"1"}})
	assert.Empty(t, dflt.encode(dflt.Page), "the default state is carried as nothing")
	assert.Equal(t, "/webui/servers", listURL(""))
	assert.Equal(t, "/webui/servers?q=srv", listURL("q=srv&bogus=1&page=1"), "unknown parameters are dropped")
	assert.Equal(t, "/webui/servers/new?ret=q%3Dsrv", withRet("/webui/servers/new", "q=srv"))
	assert.Equal(t, "/webui/servers/new", withRet("/webui/servers/new", ""))
}
