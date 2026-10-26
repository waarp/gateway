package watchers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
)

// fakeRegistry stands for the services of the gateway: the state of each
// watcher, the calls made, the failure of a run.
type fakeRegistry struct {
	states  map[int64]string
	reasons map[int64]string
	calls   []string
	fireErr error
}

func newFakeRegistry() *fakeRegistry {
	return &fakeRegistry{states: map[int64]string{}, reasons: map[int64]string{}}
}

func (f *fakeRegistry) State(fw *model.FileWatcher) (string, string) {
	if state, ok := f.states[fw.ID]; ok {
		return state, f.reasons[fw.ID]
	}

	return utils.StateOffline.String(), ""
}

func (f *fakeRegistry) Start(_ context.Context, fw *model.FileWatcher) error {
	f.calls = append(f.calls, "start:"+fw.Flow)
	f.states[fw.ID] = utils.StateRunning.String()

	return nil
}

func (f *fakeRegistry) Stop(_ context.Context, fw *model.FileWatcher) (bool, error) {
	f.calls = append(f.calls, "stop:"+fw.Flow)
	was := f.states[fw.ID] == utils.StateRunning.String()
	f.states[fw.ID] = utils.StateOffline.String()

	return was, nil
}

func (f *fakeRegistry) Remove(_ context.Context, fw *model.FileWatcher) error {
	f.calls = append(f.calls, "remove:"+fw.Flow)
	delete(f.states, fw.ID)

	return nil
}

func (f *fakeRegistry) Fire(fw *model.FileWatcher) error {
	f.calls = append(f.calls, "fire:"+fw.Flow)

	return f.fireErr
}

func (f *fakeRegistry) running(fw *model.FileWatcher) { f.states[fw.ID] = utils.StateRunning.String() }

func (f *fakeRegistry) failed(fw *model.FileWatcher, reason string) {
	f.states[fw.ID] = utils.StateError.String()
	f.reasons[fw.ID] = reason
}

type suite struct {
	db     *database.DB
	router *mux.Router
	reg    *fakeRegistry

	sftp, r66, http             *model.RemoteAgent
	prod, test, r66Acc, httpAcc *model.RemoteAccount
	cliSFTP, cliR66, cliHTTP    *model.Client
	receive, send               *model.Rule
	releves, factures           *model.FileWatcher
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter(), reg: newFakeRegistry()}
	h := New(s.db, testhelpers.GetTestLogger(t))
	h.services = s.reg

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/watchers").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/watchers/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/watchers/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/watchers/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/watchers/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/watchers/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/watchers/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/watchers/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)
	r.Path("/watchers/{id:[0-9]+}/start").Methods(http.MethodPost).HandlerFunc(h.Start)
	r.Path("/watchers/{id:[0-9]+}/stop").Methods(http.MethodPost).HandlerFunc(h.Stop)
	r.Path("/watchers/{id:[0-9]+}/fire").Methods(http.MethodPost).HandlerFunc(h.Fire)

	return s
}

// fixtures inserts an SFTP partner with two accounts, an R66 partner with
// one (lists files, another protocol) and an HTTP partner with one (HTTP
// does not list files), an SFTP client, an R66 client and an HTTP client, a receive rule and a send
// rule, and two watchers of the SFTP partner: "releves" every hour on
// "*.csv", running; "factures" every 30 minutes on everything, disabled.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.sftp = &model.RemoteAgent{Name: "part-sftp", Protocol: "sftp", Address: types.Addr("127.0.0.1", 2222)}
	s.r66 = &model.RemoteAgent{
		Name: "part-r66", Protocol: "r66", Address: types.Addr("127.0.0.2", 6666),
		ProtoConfig: map[string]any{"serverLogin": "waarp"},
	}
	s.http = &model.RemoteAgent{Name: "part-http", Protocol: "http", Address: types.Addr("127.0.0.3", 80)}

	for _, p := range []*model.RemoteAgent{s.sftp, s.r66, s.http} {
		require.NoError(t, s.db.Insert(p).Run())
	}

	s.prod = &model.RemoteAccount{RemoteAgentID: s.sftp.ID, Login: "gw-prod"}
	s.test = &model.RemoteAccount{RemoteAgentID: s.sftp.ID, Login: "gw-test"}
	s.r66Acc = &model.RemoteAccount{RemoteAgentID: s.r66.ID, Login: "gw-r66"}
	s.httpAcc = &model.RemoteAccount{RemoteAgentID: s.http.ID, Login: "gw-http"}

	for _, a := range []*model.RemoteAccount{s.prod, s.test, s.r66Acc, s.httpAcc} {
		require.NoError(t, s.db.Insert(a).Run())
	}

	s.cliSFTP = &model.Client{Name: "cli-sftp", Protocol: "sftp"}
	s.cliR66 = &model.Client{Name: "cli-r66", Protocol: "r66"}
	s.cliHTTP = &model.Client{Name: "cli-http", Protocol: "http"}

	for _, c := range []*model.Client{s.cliSFTP, s.cliR66, s.cliHTTP} {
		require.NoError(t, s.db.Insert(c).Run())
	}

	s.receive = &model.Rule{Name: "recv-releves", IsSend: false, Path: "/recv"}
	s.send = &model.Rule{Name: "send-paie", IsSend: true, Path: "/send"}

	for _, r := range []*model.Rule{s.receive, s.send} {
		require.NoError(t, s.db.Insert(r).Run())
	}

	s.releves = &model.FileWatcher{
		Flow: "releves", Pattern: "*.csv", Interval: time.Hour,
		RemoteAccount: *s.prod, Client: *s.cliSFTP, Rule: *s.receive,
	}
	s.factures = &model.FileWatcher{
		Flow: "factures", Pattern: "*", Interval: 30 * time.Minute, Disabled: true,
		RemoteAccount: *s.test, Client: *s.cliSFTP, Rule: *s.receive,
	}

	for _, fw := range []*model.FileWatcher{s.releves, s.factures} {
		require.NoError(t, s.db.Insert(fw).Run())
	}

	s.reg.running(s.releves)
}

func (s *suite) request(t *testing.T, perms model.PermsMask, method, target string, form url.Values,
) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")

	var req *http.Request
	if form != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, bytes.NewBufferString(form.Encode()))
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

	return s.request(t, perms, http.MethodGet, target, nil)
}

func (s *suite) post(t *testing.T, perms model.PermsMask, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	if form == nil {
		form = url.Values{}
	}

	return s.request(t, perms, http.MethodPost, target, form)
}

func (s *suite) watcher(t *testing.T, name string) *model.FileWatcher {
	t.Helper()

	var fw model.FileWatcher
	require.NoError(t, s.db.Get(&fw, "flow=?", name).Run())

	return &fw
}

func urlOf(fw *model.FileWatcher, suffix string) string {
	return "/webui/watchers/" + strconv.FormatInt(fw.ID, 10) + suffix
}

func id(m interface{ GetID() int64 }) string { return strconv.FormatInt(m.GetID(), 10) }

// valid is a form the screen accepts: a new watcher of the test account.
func (s *suite) valid() url.Values {
	return url.Values{
		fieldName: {"nouveau"}, fieldPattern: {"*.xml"}, fieldInterval: {"15"}, fieldUnit: {unitMinutes},
		fieldProtocol: {"sftp"}, accountField("sftp"): {id(s.test)}, clientField("sftp"): {id(s.cliSFTP)},
		fieldRule: {id(s.receive)}, fieldSkip: {"1"}, fieldEnabled: {"1"},
	}
}

func rows(body string) int { return strings.Count(body, "<tr>") - 1 }

// indexes asserts that the markers appear in the body in the given order.
func indexes(t *testing.T, body string, markers ...string) {
	t.Helper()

	from := 0

	for _, marker := range markers {
		pos := strings.Index(body[from:], marker)
		require.NotEqualf(t, -1, pos, "%q not found in order in the page", marker)
		from += pos + len(marker)
	}
}

func echo(key string, args ...any) string {
	parts := []string{key}
	for _, a := range args {
		parts = append(parts, fmt.Sprint(a))
	}

	return strings.Join(parts, " ")
}

func TestIntervalText(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "watchers.interval.never", intervalText(echo, 0))
	assert.Equal(t, "watchers.interval.minutes n 15", intervalText(echo, 15*time.Minute))
	assert.Equal(t, "watchers.interval.hours n 1", intervalText(echo, time.Hour))
	assert.Equal(t, "watchers.interval.hours n 1 watchers.interval.minutes n 30", intervalText(echo, 90*time.Minute))
	assert.Equal(t, "watchers.interval.days n 2 watchers.interval.hours n 3", intervalText(echo, 51*time.Hour))
	assert.Equal(t, "watchers.interval.minutes n 0", intervalText(echo, 30*time.Second), "under a minute: zero minutes")
}

// TestList: the list with the state of each service, the rights on the
// actions, the search, the filters, the sort, the fragment and the empty
// state.
func TestList(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("in reading: the rows, by name, with their state and no action", func(t *testing.T) {
		rec := s.get(t, permRead, "/webui/watchers")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Equal(t, 2, rows(body))
		indexes(t, body, "factures", "Arrêtée", "releves", "En ligne")
		assert.Contains(t, body, "part-sftp", "the partner")
		assert.Contains(t, body, "· gw-prod", "and its account")
		assert.Contains(t, body, "recv-releves", "the rule")
		assert.Contains(t, body, ">*.csv<", "the pattern")
		assert.Contains(t, body, "1 h", "the interval in hours")
		assert.Contains(t, body, "30 min", "or in minutes")
		assert.Contains(t, body, "2 surveillances", "the count")
		assert.NotContains(t, body, "/webui/watchers/new", "no creation without the write right")
		assert.NotContains(t, body, `id="act-`, "no action menu")
	})

	t.Run("in writing: the creation, start or stop by state, the run, the edition", func(t *testing.T) {
		body := s.get(t, permWrite, "/webui/watchers").Body.String()

		assert.Contains(t, body, `href="/webui/watchers/new"`)
		assert.Contains(t, body, urlOf(s.releves, "/stop"), "a running watcher stops")
		assert.NotContains(t, body, urlOf(s.releves, "/start"))
		assert.Contains(t, body, urlOf(s.factures, "/start"), "a stopped one starts")
		assert.NotContains(t, body, urlOf(s.factures, "/stop"))
		assert.Contains(t, body, urlOf(s.releves, "/fire"))
		assert.Contains(t, body, urlOf(s.releves, "/edit"))
		assert.NotContains(t, body, urlOf(s.releves, "/delete"), "no deletion without the delete right")

		body = s.get(t, permAll, "/webui/watchers").Body.String()
		assert.Contains(t, body, urlOf(s.releves, "/delete"))
	})

	t.Run("a service in error shows its reason", func(t *testing.T) {
		s.reg.failed(s.factures, "connexion refusée")
		defer func() { s.reg.states[s.factures.ID] = utils.StateOffline.String() }()

		body := s.get(t, permRead, "/webui/watchers").Body.String()
		assert.Contains(t, body, `<span title="connexion refusée">`)
		assert.Contains(t, body, ">Erreur<")
	})

	t.Run("the search on the name", func(t *testing.T) {
		body := s.get(t, permRead, "/webui/watchers?q=rel").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, "releves")
		assert.Contains(t, body, "2 surveillances · 1 affichée")

		body = s.get(t, permRead, "/webui/watchers?q=zzz").Body.String()
		assert.Contains(t, body, "Aucune surveillance ne correspond à « zzz »")
	})

	t.Run("the filters on the state and the partner", func(t *testing.T) {
		body := s.get(t, permRead, "/webui/watchers?state=running").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, "releves")

		body = s.get(t, permRead, "/webui/watchers?partner=part-sftp").Body.String()
		assert.Equal(t, 2, rows(body))

		body = s.get(t, permRead, "/webui/watchers?partner=part-http").Body.String()
		assert.Contains(t, body, "Aucune surveillance ne correspond aux filtres")
		assert.Contains(t, body, `name="partner" value="part-sftp"`, "the partners that have a watcher")
		assert.NotContains(t, body, `value="part-http"`, "not the others")
	})

	t.Run("the sort", func(t *testing.T) {
		body := s.get(t, permRead, "/webui/watchers?sort=partner&dir=desc").Body.String()
		indexes(t, body, "triées par partenaire", "gw-test", "gw-prod")
	})

	t.Run("the fragment of the live list", func(t *testing.T) {
		rec := s.get(t, permRead, "/webui/watchers?fragment=list")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "<html", "no shell around a fragment")
		assert.Contains(t, body, `<div id="watchers-list" data-live-part="list" data-live-paged>`)

		body = s.get(t, permRead, "/webui/watchers").Body.String()
		assert.Contains(t, body, `<div data-live data-live-period="0">`, "read on the user's actions only")
		assert.Contains(t, body, `data-autosubmit data-live-filter>`, "the filters apply in place")
	})

	t.Run("the notifications of the previous action", func(t *testing.T) {
		body := s.get(t, permRead, "/webui/watchers?done=fired&name=releves&count=3").Body.String()
		assert.Contains(t, body, "Passage de « releves » lancé : 3 transferts créés")

		body = s.get(t, permRead, "/webui/watchers?done=fired&name=releves&count=1").Body.String()
		assert.Contains(t, body, "1 transfert créé")

		body = s.get(t, permRead, "/webui/watchers?done=fired&name=releves&count=0").Body.String()
		assert.Contains(t, body, "aucun fichier nouveau")

		body = s.get(t, permRead, "/webui/watchers?failed=fire&name=releves&reason=dial+tcp%3A+refused").Body.String()
		assert.Contains(t, body, "Le passage de « releves » a échoué.")
		assert.Contains(t, body, "dial tcp: refused", "the message of the product")
	})

	t.Run("without any watcher", func(t *testing.T) {
		empty := newSuite(t)

		body := empty.get(t, permWrite, "/webui/watchers").Body.String()
		assert.Contains(t, body, "Aucune surveillance pour l")
		assert.Contains(t, body, `href="/webui/watchers/new"`)
	})
}

// TestNewWatcher: the form offers what can be watched; a valid form creates
// and starts the watcher unless it is disabled; a refused form comes back
// with its errors and its values.
func TestNewWatcher(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("the form offers the protocols that list, then the accounts by partner and the clients", func(t *testing.T) {
		rec := s.get(t, permWrite, "/webui/watchers/new")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<option value="sftp">SFTP</option>`)
		assert.Contains(t, body, `<option value="r66-tls">R66-TLS</option>`)
		assert.Contains(t, body, `<option value="webdav">WebDAV</option>`)
		assert.NotContains(t, body, `<option value="pesit`, "PeSIT does not list files")
		assert.NotContains(t, body, `<option value="http`, "nor HTTP")
		assert.Contains(t, body, `data-case="sftp"`)
		assert.Contains(t, body, `id="account-sftp" name="account-sftp"`)
		assert.Contains(t, body, `<optgroup label="part-sftp">`)
		assert.Contains(t, body, ">gw-prod<")
		assert.Contains(t, body, ">gw-test<")
		assert.NotContains(t, body, "part-http", "HTTP does not list files")
		assert.Contains(t, body, `id="client-sftp" name="client-sftp"`)
		assert.Contains(t, body, ">cli-sftp<")
		assert.Contains(t, body, `id="client-r66" name="client-r66"`)
		assert.Contains(t, body, ">cli-r66<", "every client whose protocol lists, in its block")
		assert.NotContains(t, body, "cli-http")
		assert.Contains(t, body, `<optgroup label="part-r66">`)
		assert.Contains(t, body, "Aucun partenaire FTP n", "a protocol without any account says it")
		assert.Contains(t, body, ">recv-releves<")
		assert.NotContains(t, body, "send-paie", "receive rules only")
		assert.Contains(t, body, `name="skip_duplicates" value="1" checked`)
		assert.Contains(t, body, `name="enabled" value="1" checked`)
		assert.Contains(t, body, `<option value="min" selected>`)
		assert.Equal(t, 2, strings.Count(body, `<details class="w-card w-card--fold" open>`), "the partner and the rule cards fold, open")
	})

	t.Run("without any partner that lists, the form says it", func(t *testing.T) {
		alone := newSuite(t)
		require.NoError(t, alone.db.Insert(&model.RemoteAgent{Name: "p", Protocol: "http", Address: types.Addr("127.0.0.3", 80)}).Run())

		body := alone.get(t, permWrite, "/webui/watchers/new").Body.String()
		assert.Contains(t, body, "Aucun partenaire ne peut être surveillé")
		assert.NotContains(t, body, `<form class="w-form"`)
	})

	t.Run("a valid form creates the watcher and starts it", func(t *testing.T) {
		rec := s.post(t, permWrite, "/webui/watchers/new", s.valid())
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), "done=created")
		assert.Contains(t, rec.Header().Get("Location"), "name=nouveau")

		fw := s.watcher(t, "nouveau")
		assert.Equal(t, "*.xml", fw.Pattern)
		assert.Equal(t, 15*time.Minute, fw.Interval)
		assert.Equal(t, s.test.ID, fw.RemoteAccountID)
		assert.Equal(t, s.cliSFTP.ID, fw.ClientID)
		assert.Equal(t, s.receive.ID, fw.RuleID)
		assert.False(t, fw.NoDuplicateCheck, "the files already transferred are skipped")
		assert.False(t, fw.Disabled)
		assert.Contains(t, s.reg.calls, "start:nouveau")
	})

	t.Run("a disabled watcher is stored stopped, in hours", func(t *testing.T) {
		form := s.valid()
		form.Set(fieldName, "nuit")
		form.Set(fieldInterval, "2")
		form.Set(fieldUnit, unitHours)
		form.Del(fieldEnabled)
		form.Del(fieldSkip)

		rec := s.post(t, permWrite, "/webui/watchers/new", form)
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		fw := s.watcher(t, "nuit")
		assert.True(t, fw.Disabled)
		assert.True(t, fw.NoDuplicateCheck)
		assert.Equal(t, 2*time.Hour, fw.Interval)
		assert.NotContains(t, s.reg.calls, "start:nuit")
	})

	t.Run("a refused form says what to correct and keeps the values", func(t *testing.T) {
		form := s.valid()
		form.Set(fieldName, "releves")
		form.Set(fieldInterval, "0")
		form.Set(fieldPattern, "[")
		form.Set(clientField("sftp"), id(s.cliR66))
		form.Set(fieldRule, id(s.send))

		rec := s.post(t, permWrite, "/webui/watchers/new", form)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Une autre surveillance porte déjà ce nom")
		assert.Contains(t, body, "Indiquez un intervalle")
		assert.Contains(t, body, "motif n")
		assert.Contains(t, body, "Le client doit être du même protocole que le partenaire")
		assert.Contains(t, body, "Choisissez une règle de réception")
		assert.Contains(t, body, `value="releves"`)
		assert.Contains(t, body, `value="["`)
		assert.Contains(t, body, `value="0"`)
		assert.Contains(t, body, `<option value="sftp" selected>`)
		assert.Contains(t, body, "La surveillance n")

		form = s.valid()
		form.Set(fieldName, "pesit")
		form.Set(fieldProtocol, "pesit-tls")
		rec = s.post(t, permWrite, "/webui/watchers/new", form)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Choisissez un protocole qui sait lister un dossier")

		form = s.valid()
		form.Set(fieldName, "mismatch")
		form.Set(fieldProtocol, "r66")
		form.Set(accountField("r66"), id(s.test))
		form.Set(clientField("r66"), id(s.cliR66))
		rec = s.post(t, permWrite, "/webui/watchers/new", form)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Le compte doit être celui d")

		_, err := s.db.Count(&model.FileWatcher{}).Where("flow=?", "releves").Run()
		require.NoError(t, err)
	})

	t.Run("an empty form", func(t *testing.T) {
		rec := s.post(t, permWrite, "/webui/watchers/new", url.Values{})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Ce champ est obligatoire")
		assert.Contains(t, body, "Choisissez un protocole")
		assert.NotContains(t, body, "Choisissez un compte distant", "the account and the client follow the protocol")

		form := url.Values{fieldProtocol: {"sftp"}}
		body = s.post(t, permWrite, "/webui/watchers/new", form).Body.String()
		assert.Contains(t, body, "Choisissez un compte distant")
		assert.Contains(t, body, "Choisissez un client")
	})
}

// TestEditWatcher: the form shows the stored values; a running watcher starts
// again on its new configuration, a stopped one stays stopped.
func TestEditWatcher(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("the form shows the values, the interval in its unit", func(t *testing.T) {
		rec := s.get(t, permWrite, urlOf(s.releves, "/edit"))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Modifier « releves »")
		assert.Contains(t, body, `value="releves"`)
		assert.Contains(t, body, `value="*.csv"`)
		assert.Contains(t, body, `value="1"`)
		assert.Contains(t, body, `<option value="h" selected>`, "one hour, in hours")
		assert.Contains(t, body, `<option value="sftp" selected>`, "the protocol of the client")
		assert.Contains(t, body, fmt.Sprintf(`<option value="%s" selected>gw-prod</option>`, id(s.prod)))
		assert.Contains(t, body, "SFTP · part-sftp · gw-prod", "the summary of the folded partner card")
		assert.Contains(t, body, `<span class="w-fold__meta">recv-releves</span>`, "the summary of the folded rule card")
		assert.Contains(t, body, `name="enabled" value="1" checked`)
		assert.Contains(t, body, "redémarre sur sa nouvelle configuration")

		body = s.get(t, permWrite, urlOf(s.factures, "/edit")).Body.String()
		assert.Contains(t, body, `value="30"`)
		assert.Contains(t, body, `<option value="min" selected>`)
		assert.NotContains(t, body, `name="enabled" value="1" checked`, "disabled")
	})

	t.Run("a running watcher stops, saves and starts again", func(t *testing.T) {
		s.reg.calls = nil
		form := s.valid()
		form.Set(fieldName, "releves")
		form.Set(fieldInterval, "90")
		form.Set(accountField("sftp"), id(s.prod))

		rec := s.post(t, permWrite, urlOf(s.releves, "/edit"), form)
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), "done=updated")

		fw := s.watcher(t, "releves")
		assert.Equal(t, 90*time.Minute, fw.Interval)
		assert.Equal(t, "*.xml", fw.Pattern)
		assert.Equal(t, []string{"stop:releves", "start:releves"}, s.reg.calls)
	})

	t.Run("a stopped watcher stays stopped", func(t *testing.T) {
		s.reg.calls = nil
		form := s.valid()
		form.Set(fieldName, "factures")
		form.Del(fieldEnabled)

		rec := s.post(t, permWrite, urlOf(s.factures, "/edit"), form)
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, []string{"stop:factures"}, s.reg.calls, "stopped for the save, not started")
		assert.True(t, s.watcher(t, "factures").Disabled)
	})

	t.Run("the name of another watcher is refused", func(t *testing.T) {
		form := s.valid()
		form.Set(fieldName, "releves")

		rec := s.post(t, permWrite, urlOf(s.factures, "/edit"), form)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Une autre surveillance porte déjà ce nom")
	})

	t.Run("a missing watcher", func(t *testing.T) {
		rec := s.get(t, permWrite, "/webui/watchers/999/edit")
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Surveillance introuvable")
	})
}

// TestDetail: the sheet of a watcher: its identity, the state of its
// service, the last files it received, the actions according to the rights.
func TestDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	for i, name := range []string{"releves-2026-01.csv", "releves-2026-02.csv"} {
		require.NoError(t, s.db.Insert(&model.Transfer{
			RuleID: s.receive.ID, RemoteAccountID: s.prod.NullableID(), ClientID: s.cliSFTP.NullableID(),
			SrcFilename: name, Filesize: int64(100 + i), Start: time.Date(2026, 1, 20+i, 9, 30, 0, 0, time.Local),
		}).Run())
	}

	// A file of another account: not one of this watcher.
	require.NoError(t, s.db.Insert(&model.Transfer{
		RuleID: s.receive.ID, RemoteAccountID: s.test.NullableID(), ClientID: s.cliSFTP.NullableID(),
		SrcFilename: "autre.csv", Start: time.Date(2026, 1, 25, 9, 30, 0, 0, time.Local),
	}).Run())

	t.Run("in reading", func(t *testing.T) {
		rec := s.get(t, permRead, urlOf(s.releves, ""))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Surveillance « releves »")
		assert.Contains(t, body, "part-sftp · gw-prod")
		assert.Contains(t, body, "cli-sftp · SFTP")
		assert.Contains(t, body, `href="/webui/rules/`+id(s.receive)+`">recv-releves</a>`, "the rule leads to its sheet")
		assert.Contains(t, body, `<code class="w-code">*.csv</code>`)
		assert.Contains(t, body, ">1 h<")
		assert.Contains(t, body, ">En ligne<")
		indexes(t, body, "releves-2026-02.csv", "21/01/2026", "09:30:00", "releves-2026-01.csv", "20/01/2026")
		assert.NotContains(t, body, "autre.csv", "the files of another account are not the watcher's")
		assert.Contains(t, body, `href="/webui/transfers?account=gw-prod&amp;direction=receive&amp;partner=part-sftp&amp;rule=recv-releves"`)
		assert.Contains(t, body, `<span class="w-fold__meta">2 fichiers</span>`)
		assert.NotContains(t, body, urlOf(s.releves, "/edit"), "no edition without the write right")
		assert.NotContains(t, body, `id="act-detail"`, "no action without a right")
	})

	t.Run("with every right: the edition, the stop, the run, the deletion", func(t *testing.T) {
		body := s.get(t, permAll, urlOf(s.releves, "")).Body.String()
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="`+urlOf(s.releves, "/edit"))
		assert.Contains(t, body, urlOf(s.releves, "/stop"))
		assert.Contains(t, body, urlOf(s.releves, "/fire"))
		assert.Contains(t, body, urlOf(s.releves, "/delete"))
		assert.Equal(t, 1, strings.Count(body, urlOf(s.releves, "/edit")), "the edition once, as the button")
	})

	t.Run("a service in error shows its reason; the files of its own account", func(t *testing.T) {
		s.reg.failed(s.factures, "connexion refusée")

		body := s.get(t, permRead, urlOf(s.factures, "")).Body.String()
		assert.Contains(t, body, ">Erreur<")
		assert.Contains(t, body, `<span class="wt-reason">connexion refusée</span>`)
		assert.Contains(t, body, "autre.csv", "the file of its own account")
		assert.Contains(t, body, `<span class="w-fold__meta">1 fichier</span>`)

		require.NoError(t, s.db.Insert(&model.FileWatcher{
			Flow: "vide", Interval: time.Hour, RemoteAccount: *s.r66Acc, Client: *s.cliR66, Rule: *s.receive,
		}).Run())
		body = s.get(t, permRead, urlOf(s.watcher(t, "vide"), "")).Body.String()
		assert.Contains(t, body, "Aucun fichier reçu pour l")
		assert.Contains(t, body, `<span class="w-fold__meta">Aucun</span>`)
	})

	t.Run("the list links to the sheet, the menu too", func(t *testing.T) {
		body := s.get(t, permRead, "/webui/watchers").Body.String()
		assert.Contains(t, body, `class="w-cell-title w-nowrap" href="`+urlOf(s.releves, "")+`"`)
		assert.Contains(t, body, `aria-label="Voir la fiche"`, "in reading, the sheet is the only action: an icon button")

		body = s.get(t, permAll, "/webui/watchers").Body.String()
		assert.Contains(t, body, `id="act-`+id(s.releves)+`"`)
		assert.Contains(t, body, "Voir la fiche</a>", "with more actions, the sheet opens the menu")
	})
}

// TestDeleteWatcher: the confirmation page names the watcher and its
// consequence; the deletion removes the watcher and its service.
func TestDeleteWatcher(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permAll, urlOf(s.factures, "/delete"))
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "Supprimer la surveillance « factures » ?")
	assert.Contains(t, body, "part-sftp · gw-test")
	assert.Contains(t, body, "Les transferts déjà créés restent")
	assert.Contains(t, body, `action="`+urlOf(s.factures, "/delete"))

	rec = s.post(t, permAll, urlOf(s.factures, "/delete"), nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=deleted")
	assert.Contains(t, s.reg.calls, "remove:factures")

	var fw model.FileWatcher
	require.True(t, database.IsNotFound(s.db.Get(&fw, "flow=?", "factures").Run()))
}

// TestServiceActions: start, stop and run once from the list, with the
// notification of each.
func TestServiceActions(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("start", func(t *testing.T) {
		rec := s.post(t, permWrite, urlOf(s.factures, "/start"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=started")
		assert.Contains(t, rec.Header().Get("Location"), "name=factures")
		assert.Equal(t, utils.StateRunning.String(), s.reg.states[s.factures.ID])
	})

	t.Run("stop", func(t *testing.T) {
		rec := s.post(t, permWrite, urlOf(s.releves, "/stop"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=stopped")
		assert.Equal(t, utils.StateOffline.String(), s.reg.states[s.releves.ID])
	})

	t.Run("a run that passes counts the transfers it created", func(t *testing.T) {
		rec := s.post(t, permWrite, urlOf(s.releves, "/fire"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=fired")
		assert.Contains(t, rec.Header().Get("Location"), "count=0")
		assert.Contains(t, s.reg.calls, "fire:releves")
	})

	t.Run("a run that fails says why", func(t *testing.T) {
		s.reg.fireErr = errors.New("dial tcp 127.0.0.1:2222: connection refused")
		defer func() { s.reg.fireErr = nil }()

		rec := s.post(t, permWrite, urlOf(s.releves, "/fire"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)

		location, err := url.Parse(rec.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "fire", location.Query().Get("failed"))
		assert.Equal(t, "releves", location.Query().Get("name"))
		assert.Equal(t, "dial tcp 127.0.0.1:2222: connection refused", location.Query().Get("reason"))
	})

	t.Run("the state of the list carried through an action", func(t *testing.T) {
		rec := s.post(t, permWrite, urlOf(s.releves, "/start")+"?ret="+url.QueryEscape("q=rel&state=running"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)

		location, err := url.Parse(rec.Header().Get("Location"))
		require.NoError(t, err)
		assert.Equal(t, "rel", location.Query().Get("q"))
		assert.Equal(t, "running", location.Query().Get("state"))
		assert.Equal(t, "started", location.Query().Get("done"))
	})
}
