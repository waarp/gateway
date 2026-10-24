package clients

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
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

	stopReason = "failed to stop the SFTP client: transfer 12 still running"
)

// fakeControl drives services in memory: a new client is offline, a started
// one runs, except the failing one, which stays offline with its cause.
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
		Status: func(cli *model.Client) (utils.StateCode, string) {
			c.mu.Lock()
			defer c.mu.Unlock()

			if state, ok := c.states[cli.ID]; ok {
				return state, c.reasons[cli.ID]
			}

			return utils.StateOffline, ""
		},
		Add: func(db *database.DB, cli *model.Client) error {
			if err := internal.InsertClient(db, cli); err != nil {
				return err
			}

			c.set(cli.ID, utils.StateOffline, "")

			return nil
		},
		Remove: func(_ context.Context, db *database.DB, cli *model.Client) error {
			return internal.DeleteClient(db, cli)
		},
		Start: func(_ context.Context, cli *model.Client) error {
			if cli.Name == c.failing {
				c.set(cli.ID, utils.StateError, "no network")

				return nil
			}

			c.set(cli.ID, utils.StateRunning, "")

			return nil
		},
		Stop: func(_ context.Context, cli *model.Client) error {
			if cli.Name == c.failing {
				return fmt.Errorf("%s", stopReason) //nolint:err113 // the service's own message
			}

			c.set(cli.ID, utils.StateOffline, "")

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

	sftp, r66, pesit *model.Client
	// The SFTP partner the sheet of the SFTP client links to (#714).
	partner *model.RemoteAgent
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), ctl: newFakeControl(), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t), s.ctl.control())

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/clients").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/clients/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/clients/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/clients/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/clients/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/clients/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/clients/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/clients/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)
	r.Path("/clients/{id:[0-9]+}/start").Methods(http.MethodPost).HandlerFunc(h.Start)
	r.Path("/clients/{id:[0-9]+}/stop").Methods(http.MethodPost).HandlerFunc(h.Stop)

	return s
}

// fixtures inserts three clients, one per state: a running SFTP client with
// retries, an R66 over TLS client in error with options, a PeSIT client
// offline with a local address and no automatic start; and two partners,
// one of the SFTP protocol, one of R66-TLS.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.sftp = &model.Client{
		Name: "cli-sftp", Protocol: "sftp", NbOfAttempts: 3, FirstRetryDelay: 3690, RetryIncrementFactor: 2,
		ProtoConfig: map[string]any{"keyExchanges": []string{"diffie-hellman-group-exchange-sha256"}},
	}
	s.r66 = &model.Client{
		Name: "cli-r66", Protocol: "r66-tls",
		ProtoConfig: map[string]any{"blockSize": 32768, "finalHashAlgo": "SHA-512", "minTLSVersion": "v1.3"},
	}
	s.pesit = &model.Client{
		Name: "cli-pesit", Protocol: "pesit", LocalAddress: types.Addr("10.20.30.40", 6000), Disabled: true,
		NbOfAttempts: 1, FirstRetryDelay: 45,
	}

	for _, client := range []*model.Client{s.sftp, s.r66, s.pesit} {
		require.NoError(t, s.db.Insert(client).Run())
	}

	s.partner = &model.RemoteAgent{Name: "part-sftp", Protocol: "sftp", Address: types.Addr("127.0.0.1", 22)}

	for _, partner := range []*model.RemoteAgent{
		s.partner,
		{Name: "part-r66", Protocol: "r66-tls", Address: types.Addr("127.0.0.2", 6667)},
	} {
		require.NoError(t, s.db.Insert(partner).Run())
	}

	s.ctl.set(s.sftp.ID, utils.StateRunning, "")
	s.ctl.set(s.r66.ID, utils.StateError, "no network")
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
// each one searched after the previous one.
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

func (s *suite) path(client *model.Client, suffix string) string {
	return fmt.Sprintf("/webui/clients/%d%s", client.ID, suffix)
}

// TestListFragment: the region of the list alone, for the live list of the
// kit (#766): the three elements of the part, with the ids of the page, no
// shell; the page itself declares the live region and its filters form.
func TestListFragment(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permRead, "/webui/clients?protocol=r66&fragment=list")
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.NotContains(t, body, "<html", "no shell around a fragment")
	assert.Contains(t, body, `<span id="clients-filters" data-live-part="list">Filtres · 1</span>`)
	assert.Contains(t, body, `<span class="w-toolbar__count" id="clients-count" data-live-part="list" aria-live="polite">`)
	assert.Contains(t, body, `<div id="clients-list" data-live-part="list" data-live-paged>`)
	assert.Contains(t, body, `<table class="w-table">`)

	body = s.get(t, permRead, "/webui/clients").Body.String()
	assert.Contains(t, body, `<div data-live data-live-period="0">`, "read on the user's actions only")
	assert.Contains(t, body, `data-autosubmit data-live-filter>`, "the filters apply in place")
	assert.Contains(t, body, `<template data-live-error>`)
	assert.Contains(t, body, `live.js`)
	assert.Equal(t, 1, strings.Count(body, `id="clients-list"`))
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given a user with every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, "/webui/clients")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Clients locaux · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/clients" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.Contains(t, body, `<li><span aria-current="page">Clients locaux</span></li>`)
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")

		// One coloured action, the search with its suggestions, the family filters and TLS, the count.
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/clients/new">`)
		assert.Contains(t, body, `<option value="cli-pesit"></option><option value="cli-r66"></option><option value="cli-sftp"></option>`)
		assert.Equal(t, 7, strings.Count(body, `type="checkbox" name="protocol"`), "seven families")
		assert.Equal(t, 3, strings.Count(body, `type="radio" name="tls"`), "all, with and without TLS")
		assert.Contains(t, body, `aria-live="polite">3 clients</span>`)

		// Four sortable headers, sorted by name ascending; the rows by name, each name a link.
		assert.Equal(t, 1, rows(body)-2, "one table")
		assert.Contains(t, body, `<th scope="col" aria-sort="ascending"><a href="/webui/clients?sort=name&amp;dir=desc">Client local</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/clients?sort=address&amp;dir=asc">Adresse locale</a></th>`)
		indexes(t, body, "<tbody>", "cli-pesit", "cli-r66", "cli-sftp", "</tbody>")
		assert.Contains(t, body, `<a class="w-cell-title w-nowrap" href="`+s.path(s.sftp, "")+`">cli-sftp</a>`)

		// States, the raw reason next to it; protocols with TLS as a badge; the automatic address.
		assert.Contains(t, body, `w-status w-status--ok">En ligne<`)
		assert.Contains(t, body, `w-status w-status--ko">Erreur<`)
		assert.Contains(t, body, `<span class="w-cell-sub w-mono">no network</span>`)
		assert.Contains(t, body, `<td><span class="proto-cell"><span>R66</span> <span class="proto-lock" role="img" aria-label="Chiffré par TLS" title="Chiffré par TLS">`)
		assert.Contains(t, body, `<td><span class="proto-cell"><span>SFTP</span></span></td>`)
		assert.Contains(t, body, `<span class="w-mono w-nowrap">10.20.30.40:6000</span>`)
		assert.Equal(t, 2, strings.Count(body, `<span class="w-muted">Automatique</span>`), "two clients without a local address")

		// The "⋯" menu: stop the running one, start the others, edit, delete last.
		indexes(t, body, "cli-r66", "Voir le détail", "Démarrer le client", "Modifier le client", "Supprimer le client")
		indexes(t, body, "cli-sftp", "Voir le détail", "Arrêter le client", "Modifier le client", "Supprimer le client")
		assert.Contains(t, body, `action="`+s.path(s.sftp, "/stop")+`"`)
	})

	t.Run("Given a user with the read right only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/clients").Body.String()
		assert.NotContains(t, body, "w-btn--primary", "no coloured action at all")
		assert.NotContains(t, body, "Modifier le client")
		assert.NotContains(t, body, "Supprimer le client")
		assert.NotContains(t, body, "Démarrer le client")
		assert.Contains(t, body, "Voir le détail")
	})

	t.Run("Given a search, filters and a sort", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, "/webui/clients?q=CLI-s").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, `aria-live="polite">3 clients · 1 affiché</span>`)

		body = s.get(t, permAll, "/webui/clients?protocol=r66&tls=yes").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, `<details class="w-disclose" open>`)
		assert.Contains(t, body, `name="protocol" value="r66" checked>`)
		assert.Contains(t, body, `<input type="radio" name="tls" value="yes" checked>`, "with TLS")
		assert.Contains(t, body, "Filtres · 2")

		body = s.get(t, permAll, "/webui/clients?tls=no").Body.String()
		assert.Equal(t, 2, rows(body), "the plain protocols")

		body = s.get(t, permAll, "/webui/clients?sort=state&dir=desc").Body.String()
		indexes(t, body, "<tbody>", "cli-r66", "cli-pesit", "cli-sftp", "</tbody>")

		body = s.get(t, permAll, "/webui/clients?q=nothing").Body.String()
		assert.Contains(t, body, "Aucun client ne correspond à « nothing »")
		assert.Contains(t, body, `href="/webui/clients?reset=1">Effacer la recherche</a>`)
	})

	t.Run("Given no client at all", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		body := s.get(t, permWrite, "/webui/clients").Body.String()
		assert.Contains(t, body, "Aucun client local pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/clients/new">`)

		body = s.get(t, permRead, "/webui/clients").Body.String()
		assert.NotContains(t, body, "/webui/clients/new", "no action without the write right")
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given the SFTP client with retries and a partner", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, s.path(s.sftp, "")+"?ret=q%3Dcli")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Client « cli-sftp » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="/webui/clients?q=cli">Clients locaux</a>`, "the breadcrumb keeps the list state")
		assert.Contains(t, body, "<dt>Protocole</dt><dd>SFTP</dd>")
		assert.Contains(t, body, "<dt>Adresse locale</dt><dd>Automatique</dd>")
		assert.Contains(t, body, "<dt>Démarrage automatique</dt><dd>Oui</dd>")
		assert.Contains(t, body, "<dt>Tentatives automatiques après un échec</dt><dd>3</dd>")
		assert.Contains(t, body, "<dt>Délai avant la première reprise</dt><dd>1 h 1 min 30 s</dd>")
		assert.Contains(t, body, "<dt>Facteur de progression du délai</dt><dd>2</dd>")
		assert.Contains(t, body, `<p class="clients-lead">3 tentatives : la dernière 7 h 10 min 30 s après le premier échec, avec des attentes de 1 h 1 min 30 s à 4 h 6 min.</p>`,
			"the sheet says how long the retries take")
		assert.Contains(t, body, "Options du protocole SFTP")
		assert.Contains(t, body, `<dt>Algorithmes d&#39;échange de clés</dt><dd><code class="w-code">diffie-hellman-group-exchange-sha256</code></dd>`, "a client-only algorithm")
		assert.Contains(t, body, "Partenaires joignables par ce client")
		assert.Contains(t, body, `href="/webui/partners/`+strconv.FormatInt(s.partner.ID, 10)+`">part-sftp</a>`)
		assert.NotContains(t, body, "part-r66", "only the partners of its protocol")
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/edit")+`?ret=q%3Dcli">`)
		assert.Contains(t, body, `action="`+s.path(s.sftp, "/stop")+`?ret=q%3Dcli"`)
		assert.NotContains(t, body, "Voir le détail", "no link to itself")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given the R66 over TLS client", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.path(s.r66, "")).Body.String()
		assert.Contains(t, body, "<dt>Protocole</dt><dd>R66, chiffré par TLS (R66-TLS)</dd>")
		assert.Contains(t, body, `<code class="w-code">no network</code>`)
		assert.Contains(t, body, "Aucune reprise automatique")
		assert.Contains(t, body, "Options du protocole R66")
		assert.Contains(t, body, `<dt>Taille d&#39;un bloc de données</dt><dd><code class="w-code">32768</code></dd>`)
		assert.Contains(t, body, "<dt>Algorithme du hachage en fin de transfert</dt><dd>SHA-512</dd>")
		assert.Contains(t, body, "<dt>Version TLS minimale</dt><dd>TLS 1.3</dd>")
		assert.Contains(t, body, "part-r66")
		assert.NotContains(t, body, "Modifier le client", "no edit without the write right")
	})

	t.Run("Given the PeSIT client without partner", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.path(s.pesit, "")).Body.String()
		assert.Contains(t, body, `<dd><code class="w-code">10.20.30.40:6000</code></dd>`)
		assert.Contains(t, body, "<dt>Démarrage automatique</dt><dd>Non</dd>")
		assert.Contains(t, body, "<dt>Délai avant la première reprise</dt><dd>45 s</dd>")
		assert.Contains(t, body, "<dt>Facteur de progression du délai</dt><dd>1</dd>", "the product applies 1")
		assert.Contains(t, body, "Aucun partenaire de ce protocole pour l&#39;instant.")
		assert.NotContains(t, body, "Mode PeSIT", "a partner option, not a client one")
	})

	t.Run("Given an unknown client", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permRead, "/webui/clients/999?ret=page%3D2")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce client n&#39;existe pas ou a été supprimé.")
		assert.Contains(t, rec.Body.String(), `href="/webui/clients?page=2"`, "the way back keeps the list state")
	})
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("Given the empty form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.get(t, permWrite, "/webui/clients/new")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Nouveau client · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<form class="w-form" method="post" action="/webui/clients/new" novalidate>`)
		assert.Contains(t, body, `<option value="" selected>Choisissez un protocole</option>`)
		assert.Contains(t, body, `<input type="checkbox" id="tls" name="tls" value="1" data-switch="tls"`)
		assert.Contains(t, body, `id="port" name="port" value="" type="number"`, "the port is optional")
		assert.Contains(t, body, `id="attempts" name="attempts" value="" type="number" inputmode="numeric" step="1" min="0" max="127"`)
		assert.Contains(t, body, `id="delay_h" name="delay_h"`)
		assert.Contains(t, body, `id="delay_s" name="delay_s" value="" type="number" inputmode="numeric" step="1" min="0" max="59"`)
		assert.Contains(t, body, `id="factor" name="factor"`)
		assert.Equal(t, 2, strings.Count(body, `<details class="w-card w-card--fold">`), "the address and the retries fold when empty")
		assert.Contains(t, body, `<span class="w-fold__meta">Automatique</span>`)
		assert.Contains(t, body, `<span class="w-fold__meta">Aucune reprise automatique</span>`)
		assert.Contains(t, body, `<p class="w-retry__text" id="retry-text" aria-live="polite">Aucune reprise automatique.</p>`, "the sentence of the retries, rendered by the server")
		assert.Contains(t, body, `<details class="w-retry__detail" hidden>`, "no table without retries")
		assert.Contains(t, body, `<span class="w-field__hint" id="delay-h">Vide : la passerelle attend 1 s</span>`)
		assert.Contains(t, body, `clients/clients.js" defer>`, "the script that draws the retries live")
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-case="r66">`, "the options fold at the product defaults")
		assert.Equal(t, 5, strings.Count(body, `<span class="w-fold__meta">Options par défaut de la passerelle</span>`), "one summary per family with options")
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-case="ftp">`)
		assert.Contains(t, body, `name="ftp_enable_active"`, "the client side of FTP: the active mode")
		assert.NotContains(t, body, `name="ftp_disable_passive"`, "the server side is not offered")
		assert.Contains(t, body, `id="r66_final_hash_algo" name="r66_final_hash_algo"`)
		assert.Contains(t, body, `<option value="SHA-256" selected>SHA-256</option>`)
		assert.NotContains(t, body, `name="r66_login"`)
		assert.NotContains(t, body, `name="pesit_mode"`)
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-case="on" data-of="tls">`)
		assert.Contains(t, body, `>Valider</button>`)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a valid form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/clients/new?ret=sort%3Dstate%26dir%3Ddesc", url.Values{
			"name": {" cli-new "}, "protocol": {"ftp"}, "tls": {"1"}, "host": {"10.0.0.5"}, "port": {"40000"},
			"attempts": {"5"}, "delay_h": {"1"}, "delay_m": {"15"}, "delay_s": {"30"}, "factor": {"1,5"},
			"ftp_enable_active": {"1"}, "ftp_active_min": {"50000"}, "min_tls": {"v1.2"},
			"r66_block": {"not a number"}, "sftp_kex": {"curve25519-sha256"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/clients?dir=desc&done=created&name=cli-new&sort=state", rec.Header().Get("Location"))

		client, err := internal.GetClient(s.db, "cli-new")
		require.NoError(t, err)
		assert.Equal(t, "ftps", client.Protocol)
		assert.Equal(t, "10.0.0.5:40000", client.LocalAddress.String())
		assert.EqualValues(t, 5, client.NbOfAttempts)
		assert.EqualValues(t, 4530, client.FirstRetryDelay, "1 h 15 min 30 s in seconds")
		assert.InDelta(t, 1.5, client.RetryIncrementFactor, 0.001, "a decimal comma is accepted")
		assert.False(t, client.Disabled)
		assert.True(t, protoform.CfgBool(client.ProtoConfig, "enableActiveMode"))
		assert.Equal(t, "50000", protoform.CfgString(client.ProtoConfig, "activeModeMinPort"))
		assert.Equal(t, "v1.2", protoform.CfgString(client.ProtoConfig, "minTLSVersion"))
		assert.NotContains(t, client.ProtoConfig, "blockSize", "the options of the other protocols are ignored")
		assert.NotContains(t, client.ProtoConfig, "keyExchanges")

		body := s.get(t, permWrite, "/webui/clients?done=created&name=cli-new").Body.String()
		assert.Contains(t, body, "Client « cli-new » créé")
	})

	t.Run("Given a client without address nor retries", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/clients/new", url.Values{"name": {"cli-bare"}, "protocol": {"http"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		client, err := internal.GetClient(s.db, "cli-bare")
		require.NoError(t, err)
		assert.Equal(t, "http", client.Protocol)
		assert.False(t, client.LocalAddress.IsSet())
		assert.Zero(t, client.NbOfAttempts)
		assert.Zero(t, client.FirstRetryDelay)
	})

	t.Run("Given an invalid form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/clients/new", url.Values{
			"name": {""}, "protocol": {"bogus"}, "port": {"70000"}, "attempts": {"300"}, "delay_m": {"-1"},
			"factor": {"0"}, "host": {"kept.example"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<div class="w-banner w-banner--ko" role="alert">`)
		assert.Contains(t, body, "Le client n&#39;a pas été enregistré.")
		assert.Contains(t, body, "Ce champ est obligatoire.")
		assert.Contains(t, body, "Choisissez un protocole.")
		assert.Contains(t, body, "Indiquez un port entre 1 et 65535.")
		assert.Contains(t, body, "Indiquez un nombre de tentatives entre 0 et 127.")
		assert.Contains(t, body, "Indiquez un nombre entier positif.")
		assert.Contains(t, body, "Indiquez un facteur supérieur à 0.")
		assert.Contains(t, body, `value="kept.example"`, "the values are kept")
		assert.Contains(t, body, `<span class="w-fold__meta">kept.example:70000</span>`, "the address card opens on its error")

		count, err := s.db.Count(&model.Client{}).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 3, count, "nothing created")

		rec = s.post(t, permWrite, "/webui/clients/new", url.Values{"name": {"cli-sftp"}, "protocol": {"sftp"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre client porte déjà ce nom.")
	})
}

func TestEditClient(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given the form of the SFTP client", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, s.path(s.sftp, "/edit")+"?ret=page%3D2").Body.String()
		assert.Contains(t, body, "<title>Modifier « cli-sftp » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `id="name" name="name" value="cli-sftp"`)
		assert.Contains(t, body, `<option value="sftp" selected>SFTP</option>`)
		assert.Contains(t, body, `id="attempts" name="attempts" value="3"`)
		assert.Contains(t, body, `id="delay_h" name="delay_h" value="1"`)
		assert.Contains(t, body, `id="delay_m" name="delay_m" value="1"`)
		assert.Contains(t, body, `id="delay_s" name="delay_s" value="30"`)
		assert.Contains(t, body, `id="factor" name="factor" value="2"`)
		assert.Contains(t, body, `<details class="w-card w-card--fold" open><summary>`, "the retries card opens with its values")
		assert.Contains(t, body, `<span class="w-fold__meta">3 tentatives · 1 h 1 min 30 s · ×2</span>`)
		assert.Contains(t, body, `>3 tentatives : la dernière 7 h 10 min 30 s après le premier échec, avec des attentes de 1 h 1 min 30 s à 4 h 6 min.</p>`,
			"the sentence says how long the retries take")
		assert.Contains(t, body, `<ol class="w-retry__rows"><li>Après 1 h 1 min 30 s d&#39;attente, à +1 h 1 min 30 s du premier échec.</li>`+
			`<li>Après 2 h 3 min d&#39;attente, à +3 h 4 min 30 s du premier échec.</li>`, "the list of the attempts")
		assert.NotContains(t, body, "<table", "no table in a card")
		assert.Contains(t, body, `<option value="diffie-hellman-group-exchange-sha256" selected>`)
		assert.Contains(t, body, `href="`+s.path(s.sftp, "")+`?ret=page%3D2">`, "cancel goes back to the client")
	})

	t.Run("Given the form of the R66 over TLS client", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, s.path(s.r66, "/edit")).Body.String()
		assert.Contains(t, body, `<option value="r66" selected>R66</option>`)
		assert.Contains(t, body, `name="tls" value="1" data-switch="tls" aria-describedby="tls-h" checked>`)
		assert.Contains(t, body, `id="r66_block" name="r66_block" value="32768"`)
		assert.Contains(t, body, `<option value="SHA-512" selected>SHA-512</option>`)
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-case="r66" open>`, "the options card opens with its values")
		assert.Contains(t, body, `<span class="w-fold__meta">2 options renseignées</span>`, "the block size and the hash algorithm")
		assert.Contains(t, body, `<option value="v1.3" selected>TLS 1.3</option>`)
	})

	t.Run("Given a saved edition", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.path(s.pesit, "/edit")+"?ret=q%3Dcli", url.Values{
			"name": {"cli-pesit"}, "protocol": {"pesit"}, "tls": {"1"}, "no_autostart": {"1"}, "attempts": {"0"},
			"pesit_disable_restart": {"1"}, "min_tls": {"v1.3"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, s.path(s.pesit, "")+"?done=updated&name=cli-pesit&ret=q%3Dcli", rec.Header().Get("Location"),
			"an edited client comes back to its page")

		saved, err := internal.GetClientByID(s.db, s.pesit.ID)
		require.NoError(t, err)
		assert.Equal(t, "pesit-tls", saved.Protocol)
		assert.True(t, saved.Disabled)
		assert.False(t, saved.LocalAddress.IsSet(), "the address was cleared")
		assert.Zero(t, saved.FirstRetryDelay, "the delay was cleared")
		assert.True(t, protoform.CfgBool(saved.ProtoConfig, "disableRestart"))
		assert.Equal(t, "v1.3", protoform.CfgString(saved.ProtoConfig, "minTLSVersion"))
	})

	t.Run("Given a name already taken", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.path(s.r66, "/edit"), url.Values{"name": {"cli-sftp"}, "protocol": {"r66"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre client porte déjà ce nom.")

		client, err := internal.GetClientByID(s.db, s.r66.ID)
		require.NoError(t, err)
		assert.Equal(t, "cli-r66", client.Name, "nothing changed")
	})
}

func TestDeleteClient(t *testing.T) {
	t.Parallel()

	t.Run("Given the confirmation page and the deletion", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, s.path(s.sftp, "/delete")+"?ret=q%3Dcli").Body.String()
		assert.Contains(t, body, "<title>Supprimer le client « cli-sftp » ? · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Il est en ligne : il sera arrêté et ses transferts en cours interrompus.")
		assert.Contains(t, body, `action="`+s.path(s.sftp, "/delete")+`?ret=q%3Dcli"`)
		assert.Contains(t, body, ">Supprimer</button>")

		rec := s.post(t, permAll, s.path(s.sftp, "/delete")+"?ret=q%3Dcli", nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/clients?done=deleted&name=cli-sftp&q=cli", rec.Header().Get("Location"))

		_, err := internal.GetClientByID(s.db, s.sftp.ID)
		assert.True(t, database.IsNotFound(err))
	})

	t.Run("Given a client used by a folder watcher", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		partner, err := internal.GetPartner(s.db, "part-sftp")
		require.NoError(t, err)
		account := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "me"}
		require.NoError(t, s.db.Insert(account).Run())
		rule := &model.Rule{Name: "pull", IsSend: false, Path: "/pull"}
		require.NoError(t, s.db.Insert(rule).Run())
		require.NoError(t, s.db.Insert(&model.FileWatcher{
			Flow: "watch", Pattern: "*.txt", RemoteAccount: *account, Client: *s.sftp, Rule: *rule,
		}).Run())

		rec := s.get(t, permAll, s.path(s.sftp, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce client ne peut pas être supprimé.")
		assert.NotContains(t, rec.Body.String(), ">Supprimer</button>")

		rec = s.post(t, permAll, s.path(s.sftp, "/delete"), nil)
		assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		_, err = internal.GetClientByID(s.db, s.sftp.ID)
		require.NoError(t, err, "still there")
	})
}

func TestStartStop(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.ctl.failing = "cli-r66"

	rec := s.post(t, permWrite, s.path(s.pesit, "/start")+"?ret=page%3D2", nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/webui/clients?done=started&name=cli-pesit&page=2", rec.Header().Get("Location"))

	rec = s.post(t, permWrite, s.path(s.pesit, "/stop"), nil)
	assert.Equal(t, "/webui/clients?done=stopped&name=cli-pesit", rec.Header().Get("Location"))

	rec = s.post(t, permWrite, s.path(s.r66, "/start"), nil)
	assert.Equal(t, "/webui/clients?failed=start&name=cli-r66&reason=no+network", rec.Header().Get("Location"))

	body := s.get(t, permWrite, "/webui/clients?failed=start&name=cli-r66&reason=no+network").Body.String()
	assert.Contains(t, body, "Le client « cli-r66 » n&#39;a pas démarré")
	assert.Contains(t, body, "no network")

	s.ctl.set(s.r66.ID, utils.StateRunning, "")
	rec = s.post(t, permWrite, s.path(s.r66, "/stop"), nil)
	assert.Contains(t, rec.Header().Get("Location"), "failed=stop&name=cli-r66&reason=")

	body = s.get(t, permWrite, "/webui/clients?failed=stop&name=cli-r66&reason="+url.QueryEscape(stopReason)).Body.String()
	assert.Contains(t, body, "n&#39;a pas pu être arrêté")
}

func TestDelay(t *testing.T) {
	t.Parallel()

	translate := func(key string, args ...any) string {
		if len(args) == 2 {
			return fmt.Sprintf("%v%s", args[1], strings.TrimPrefix(key, "clients.retries.delay_"))
		}

		return key
	}

	assert.Equal(t, "clients.retries.delay_none", formatDelay(translate, 0))
	assert.Equal(t, "45seconds", formatDelay(translate, 45))
	assert.Equal(t, "1hours 1minutes 30seconds", formatDelay(translate, 3690))
	assert.Equal(t, "2hours", formatDelay(translate, 7200))
	assert.Equal(t, "1", formatFactor(0))
	assert.Equal(t, "1.5", formatFactor(1.5))
}
