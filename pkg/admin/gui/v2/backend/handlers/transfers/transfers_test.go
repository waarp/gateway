package transfers

import (
	"bytes"
	"context"
	"database/sql"
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

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermTransfersRead
	permWrite = model.PermTransfersRead | model.PermTransfersWrite
)

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router

	bank    *model.RemoteAgent
	prod    *model.RemoteAccount
	client  *model.Client
	server  *model.LocalAgent
	south   *model.LocalAccount
	send    *model.Rule
	receive *model.Rule

	running, failed, planned, available *model.Transfer
	done, cancelled                     *model.HistoryEntry
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/transfers").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/transfers/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/transfers/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/transfers/preregister").Methods(http.MethodGet).HandlerFunc(h.PreregisterForm)
	r.Path("/transfers/preregister").Methods(http.MethodPost).HandlerFunc(h.Preregister)
	r.Path("/transfers/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/transfers/{id:[0-9]+}/pause").Methods(http.MethodPost).HandlerFunc(h.Pause)
	r.Path("/transfers/{id:[0-9]+}/resume").Methods(http.MethodPost).HandlerFunc(h.Resume)
	r.Path("/transfers/{id:[0-9]+}/cancel").Methods(http.MethodGet).HandlerFunc(h.CancelForm)
	r.Path("/transfers/{id:[0-9]+}/cancel").Methods(http.MethodPost).HandlerFunc(h.Cancel)
	r.Path("/transfers/{id:[0-9]+}/retry").Methods(http.MethodGet).HandlerFunc(h.RetryForm)
	r.Path("/transfers/{id:[0-9]+}/retry").Methods(http.MethodPost).HandlerFunc(h.Retry)

	return s
}

// at is a moment of the day of the fixtures, in the time zone of the gateway.
func at(hour, minute, second int) time.Time {
	return time.Date(2026, 10, 21, hour, minute, second, 0, time.Local)
}

// fixtures inserts a partner with its account and a client, a server with
// its account, a sending and a receiving rule; four transfers (running,
// failed, planned, and a server one available) and two history entries (done
// and cancelled).
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.bank = &model.RemoteAgent{Name: "banque-demo", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)}
	require.NoError(t, s.db.Insert(s.bank).Run())

	s.prod = &model.RemoteAccount{RemoteAgentID: s.bank.ID, Login: "prod"}
	require.NoError(t, s.db.Insert(s.prod).Run())

	s.client = &model.Client{Name: "cli-sftp", Protocol: "sftp"}
	require.NoError(t, s.db.Insert(s.client).Run())

	s.server = &model.LocalAgent{Name: "srv-sftp", Protocol: "sftp", Address: types.Addr("0.0.0.0", 2222)}
	require.NoError(t, s.db.Insert(s.server).Run())

	s.south = &model.LocalAccount{LocalAgentID: s.server.ID, Login: "filiale-sud"}
	require.NoError(t, s.db.Insert(s.south).Run())

	s.send = &model.Rule{Name: "releves", IsSend: true, Path: "/releves"}
	s.receive = &model.Rule{Name: "commandes", IsSend: false, Path: "/commandes"}
	require.NoError(t, s.db.Insert(s.send).Run())
	require.NoError(t, s.db.Insert(s.receive).Run())

	client := func(file string, status types.TransferStatus, start time.Time) *model.Transfer {
		return &model.Transfer{
			RuleID: s.send.ID, ClientID: s.client.NullableID(), RemoteAccountID: s.prod.NullableID(),
			SrcFilename: file, DestFilename: file, Status: status, Start: start, Filesize: model.UnknownSize,
		}
	}

	s.running = client("paie-octobre.xml", types.StatusRunning, at(10, 40, 58))
	s.running.Filesize, s.running.Progress, s.running.Step = 1228800, 786432, types.StepData

	s.failed = client("releve-20102026.pdf", types.StatusError, at(10, 38, 2))
	s.failed.ErrCode, s.failed.ErrDetails = types.TeForbidden, "failed to make transfer request: FICHIER DEJA RECU"
	s.failed.Step, s.failed.RemainingTries = types.StepSetup, 2
	s.failed.TransferInfo = map[string]any{model.FollowID: 2112951385528143872, "lot": "2026-10-20-B"}

	s.planned = client("rapport-mensuel.pdf", types.StatusPlanned, at(18, 0, 0))

	s.available = &model.Transfer{
		RuleID: s.receive.ID, LocalAccountID: s.south.NullableID(), DestFilename: "inventaire.csv",
		Status: types.StatusAvailable, Start: at(23, 0, 0),
	}

	for _, trans := range []*model.Transfer{s.running, s.failed, s.planned, s.available} {
		require.NoError(t, s.db.Insert(trans).Run())
	}

	// The insertion plans the first attempt at the start; the controller
	// moves it after a failure.
	s.failed.NextRetry = at(10, 43, 2)
	require.NoError(t, s.db.Update(s.failed).Cols("next_retry").Run())

	s.done = &model.HistoryEntry{
		Identifier: model.ID(1001), RemoteTransferID: "r-1001", IsSend: false, IsServer: true, Rule: "commandes", Account: "filiale-sud",
		Agent: "srv-sftp", Protocol: "sftp", SrcFilename: "commandes-0457.edi", DestFilename: "commandes-0457.edi",
		Filesize: 49152, Progress: 49152, Start: at(10, 36, 44), Stop: at(10, 36, 45), Status: types.StatusDone,
	}
	s.cancelled = &model.HistoryEntry{
		Identifier: model.ID(1002), RemoteTransferID: "r-1002", IsSend: true, Rule: "releves", Account: "prod", Agent: "banque-demo",
		Client: "cli-sftp", Protocol: "sftp", SrcFilename: "virements.xml", DestFilename: "virements.xml",
		Filesize: model.UnknownSize, Start: at(10, 12, 40), Stop: at(10, 13, 2), Status: types.StatusCancelled,
	}

	for _, hist := range []*model.HistoryEntry{s.done, s.cancelled} {
		require.NoError(t, s.db.Insert(hist).Run())
	}
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

func pathOf(id int64, suffix string) string {
	return "/webui/transfers/" + strconv.FormatInt(id, 10) + suffix
}

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

func rows(body string) int {
	start, end := strings.Index(body, "<tbody>"), strings.Index(body, "</tbody>")
	if start < 0 || end < 0 {
		return 0
	}

	return strings.Count(body[start:end], "<tr>")
}

func statusOfTransfer(t *testing.T, db *database.DB, id int64) types.TransferStatus {
	t.Helper()

	view := &model.NormalizedTransferView{}
	require.NoError(t, db.Get(view, "id=?", id).Run())

	return view.Status
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given a user who may act on the transfers", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, "/webui/transfers")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Suivi des transferts · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/transfers" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"), "one coloured action")
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/transfers/new">`)
		assert.Contains(t, body, `href="/webui/transfers/preregister"`)
		assert.Contains(t, body, "transfers.js", "the list refreshes itself")
		assert.Contains(t, body, `data-refresh-seconds="5"`)

		// The row of the states: every state then each one, with its count.
		count := func(label string, n int) string {
			return label + `</span> <span class="tf-count">` + strconv.Itoa(n) + `</span>`
		}
		indexes(t, body, `id="transfers-states"`, `aria-current="true"><span>`+count("Tous", 6),
			"<span>"+count("En cours", 1), `<span class="w-status w-status--ko">`+count("En erreur", 1),
			"<span>"+count("Planifiés", 1), "<span>"+count("En pause", 0),
			"<span>"+count("Interrompus", 0), "<span>"+count("Attendus", 1),
			"<span>"+count("Terminés", 1), "<span>"+count("Annulés", 1), "</nav>")
		assert.Contains(t, body, `id="transfers-count" aria-live="polite">6 transferts</span>`)
		assert.Contains(t, body, `title="Transfert pré-enregistré : la passerelle attend que le partenaire se connecte`,
			"each state says what it means")

		// The last transfers first (the highest numbers); the number, then the file with its protocol under it.
		assert.Equal(t, 6, rows(body))
		indexes(t, body, "<tbody>", "<bdi>virements.xml</bdi></a>", "<bdi>commandes-0457.edi</bdi></a>", "<bdi>inventaire.csv</bdi></a>",
			"<bdi>rapport-mensuel.pdf</bdi></a>", "<bdi>releve-20102026.pdf</bdi></a>", "<bdi>paie-octobre.xml</bdi></a>", "</tbody>")
		assert.Contains(t, body, `class="w-num" aria-sort="descending"><a href="/webui/transfers?sort=id&amp;dir=asc">N°</a>`)
		// One line per transfer, in the order of the columns of the historic screen: the direction as an icon, the
		// number, the state (strong), the file, the rule, the partner, the start, the end, the progress, the size;
		// what does not fit goes to the tooltips; a failed transfer says "Failed" alone.
		indexes(t, body, `<td class="tf-dir" title="Envoi depuis le client vers le serveur">`,
			`<td class="w-num tf-num">`+strconv.FormatInt(s.failed.ID, 10)+`</td>`,
			`title="Accès refusé : failed to make transfer request: FICHIER DEJA RECU"`,
			`<span class="w-status w-status--ko w-status--strong">En erreur</span>`,
			`title="releve-20102026.pdf"><bdi>releve-20102026.pdf</bdi></a>`,
			`title="releves · banque-demo · compte prod · SFTP">releves</td>`,
			`<td class="tf-x" title="banque-demo · compte prod · SFTP">banque-demo</td>`,
			`title="21/10/2026 à 10:38:02">21/10/2026 10:38:02</span>`,
			`<span class="tf-next" title="Prochaine tentative le 21/10/2026 à 10:43:02">`, "<span>21/10/2026 10:43:02</span>")
		assert.NotContains(t, body, "tf-cause", "no part of the error under the state")
		assert.NotContains(t, body, "tf-sub", "no second line in the list")
		assert.NotContains(t, body, `title="prod">prod</td>`, "the account in the tooltip of the partner only")
		indexes(t, body, "<bdi>paie-octobre.xml</bdi></a>", `<span class="tf-gauge"><progress class="tf-progress" max="100" value="64"`,
			`<span class="tf-gauge__pct" aria-hidden="true">64 %</span>`, `<td class="w-num" title="1 228 800 octets">1,2 Mo</td>`)
		indexes(t, body, "Attendu</span>", "<bdi>inventaire.csv</bdi></a>", "srv-sftp",
			`title="Attendu jusqu&#39;au 21/10/2026 à 23:00:00">21/10/2026 23:00:00</span>`)
		indexes(t, body, "Planifié</span>", "<bdi>rapport-mensuel.pdf</bdi></a>",
			`title="21/10/2026 à 18:00:00">21/10/2026 18:00:00</span>`, "Taille inconnue", "</tr>")
		assert.Contains(t, body, `title="Réception sur le serveur depuis le client"`, "the direction and the role in the tooltip")
		assert.Contains(t, body, `<use href="#cloud-download"/>`, "a cloud when the gateway is the server")
		assert.NotContains(t, body, ">0 octet<", "no size before the transfer starts")

		// The partner and the account chosen in lists, as the rule and the direction: the partners and the servers
		// in two groups, the accounts under their partner or server.
		indexes(t, body, `<select class="w-select" id="f-partner" name="partner">`, "Tous les partenaires et serveurs",
			`<optgroup label="Partenaires distants">`, `<option value="banque-demo">banque-demo</option>`,
			`<optgroup label="Serveurs locaux">`, `<option value="srv-sftp">srv-sftp</option>`, "</select>",
			`<select class="w-select" id="f-account" name="account">`, "Tous les comptes",
			`<optgroup label="banque-demo">`, `<option value="prod">prod</option>`,
			`<optgroup label="srv-sftp">`, `<option value="filiale-sud">filiale-sud</option>`, "</select>")

		// A partner chosen: its accounts alone.
		rec = s.get(t, permWrite, "/webui/transfers?partner=srv-sftp&account=filiale-sud")
		chosen := rec.Body.String()
		indexes(t, chosen, `<option value="srv-sftp" selected>srv-sftp</option>`, `<select class="w-select" id="f-account"`,
			`<optgroup label="srv-sftp">`, `<option value="filiale-sud" selected>filiale-sud</option>`, "</select>")
		assert.NotContains(t, chosen, `<optgroup label="banque-demo">`, "the accounts of the chosen partner alone")

		// A partner of the address that the lists do not hold shows alone, chosen.
		rec = s.get(t, permWrite, "/webui/transfers?partner=partenaire-supprime")
		assert.Contains(t, rec.Body.String(), `<option value="partenaire-supprime" selected>partenaire-supprime</option>`)

		// The secondary action of the header in a "⋯" menu, as on the other pages.
		indexes(t, body, `popovertarget="act-page"`, `<div class="w-actions" id="act-page" popover>`,
			`href="/webui/transfers/preregister"`)

		// The actions the state allows, in the menu.
		indexes(t, body, "<bdi>virements.xml</bdi></a>", "Voir le transfert", "Relancer", "<bdi>commandes-0457.edi</bdi></a>")
		indexes(t, body, "<bdi>releve-20102026.pdf</bdi></a>", "Voir le transfert", "Reprendre", "Relancer", "Annuler le transfert",
			"<bdi>paie-octobre.xml</bdi></a>", "Voir le transfert", "Mettre en pause", "Annuler le transfert", "</tbody>")
	})

	t.Run("Given a user who may only read", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/transfers").Body.String()
		assert.NotContains(t, body, "/webui/transfers/new", "no addition without the write right")
		assert.NotContains(t, body, "w-btn--primary")
		assert.NotContains(t, body, "Mettre en pause")
		assert.NotContains(t, body, "Annuler le transfert")
		assert.Contains(t, body, "Voir le transfert")
	})

	t.Run("Given a state", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/transfers?state=error").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, "<bdi>releve-20102026.pdf</bdi></a>")
		assert.Contains(t, body, `href="/webui/transfers?state=error" title="Le transfert a échoué ; il peut être retenté, repris ou relancé" aria-current="true"><span class="w-status w-status--ko">En erreur</span>`)
		assert.Contains(t, body, `<input type="hidden" name="state" value="error">`, "a search keeps the state")
		assert.Contains(t, body, `aria-live="polite">1 transfert</span>`)

		body = s.get(t, permRead, "/webui/transfers?state=paused").Body.String()
		assert.Contains(t, body, "Aucun transfert « En pause »")
		assert.Contains(t, body, `href="/webui/transfers?reset=1"`, "the way out shows every state")
	})

	t.Run("Given a search", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/transfers?q=paie").Body.String()
		assert.Equal(t, 1, rows(body), "by a part of the file name")
		assert.Contains(t, body, "<bdi>paie-octobre.xml</bdi></a>")
		assert.Contains(t, body, `<span>Tous</span> <span class="tf-count">1</span>`, "the states count what the search finds")
		assert.Contains(t, body, `<span>En erreur</span> <span class="tf-count">0</span>`,
			"no error shape without a failed transfer")

		body = s.get(t, permRead, "/webui/transfers?q=2112951385528143872").Body.String()
		assert.Equal(t, 1, rows(body), "by the follow ID")
		assert.Contains(t, body, "<bdi>releve-20102026.pdf</bdi></a>")

		body = s.get(t, permRead, "/webui/transfers?q=absent").Body.String()
		assert.Contains(t, body, "Aucun transfert ne correspond à « absent »")
	})

	t.Run("Given filters", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/transfers?rule=commandes").Body.String()
		assert.Equal(t, 2, rows(body))
		assert.Contains(t, body, "Filtres · 1")
		assert.Contains(t, body, `<details class="w-disclose" open>`)

		body = s.get(t, permRead, "/webui/transfers?direction=receive&partner=srv-sftp").Body.String()
		assert.Equal(t, 2, rows(body))
		assert.Contains(t, body, "Filtres · 2")

		body = s.get(t, permRead, "/webui/transfers?account=prod&protocol=sftp").Body.String()
		assert.Equal(t, 4, rows(body))
		assert.Contains(t, body, `<option value="sftp" selected>SFTP</option>`)
		assert.Contains(t, body, `<option value="r66">R66</option>`, "the families of the protocols")
		assert.NotContains(t, body, `value="r66-tls"`, "a TLS variant goes with its family")

		body = s.get(t, permRead, "/webui/transfers?protocol=r66-tls").Body.String()
		assert.Contains(t, body, `<option value="r66" selected>R66</option>`, "a variant reads as its family")

		body = s.get(t, permRead, "/webui/transfers?from=2026-10-21T10:36&to=2026-10-21T10:39").Body.String()
		assert.Equal(t, 2, rows(body), "the period of the start, its last minute included")

		body = s.get(t, permRead, "/webui/transfers?direction=sideways&from=hier").Body.String()
		assert.Equal(t, 6, rows(body), "an invalid filter is dropped")
		assert.NotContains(t, body, "Filtres · ")
	})

	t.Run("Given a sort", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/transfers?sort=file&dir=asc").Body.String()
		indexes(t, body, "<tbody>", "<bdi>commandes-0457.edi</bdi></a>", "<bdi>inventaire.csv</bdi></a>", "<bdi>paie-octobre.xml</bdi></a>",
			"<bdi>rapport-mensuel.pdf</bdi></a>", "<bdi>releve-20102026.pdf</bdi></a>", "<bdi>virements.xml</bdi></a>")

		body = s.get(t, permRead, "/webui/transfers?sort=state&dir=asc").Body.String()
		indexes(t, body, "<tbody>", "<bdi>paie-octobre.xml</bdi></a>", "<bdi>releve-20102026.pdf</bdi></a>", "<bdi>rapport-mensuel.pdf</bdi></a>",
			"<bdi>inventaire.csv</bdi></a>", "<bdi>commandes-0457.edi</bdi></a>", "<bdi>virements.xml</bdi></a>")

		body = s.get(t, permRead, "/webui/transfers?sort=start&dir=asc").Body.String()
		indexes(t, body, "<tbody>", "<bdi>virements.xml</bdi></a>", "<bdi>commandes-0457.edi</bdi></a>", "<bdi>rapport-mensuel.pdf</bdi></a>")
		assert.Contains(t, body, `aria-sort="ascending"`)

		body = s.get(t, permRead, "/webui/transfers?sort=id&dir=asc").Body.String()
		indexes(t, body, "<tbody>", "<bdi>paie-octobre.xml</bdi></a>", "<bdi>releve-20102026.pdf</bdi></a>", "<bdi>virements.xml</bdi></a>")

		body = s.get(t, permRead, "/webui/transfers?sort=start&dir=desc").Body.String()
		indexes(t, body, "<tbody>", "<bdi>inventaire.csv</bdi></a>", "<bdi>rapport-mensuel.pdf</bdi></a>", "<bdi>paie-octobre.xml</bdi></a>")
	})

	t.Run("Given the refresh", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permRead, "/webui/transfers?state=error&fragment=list")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "<html", "the fragment alone")
		indexes(t, body, `<div id="transfers-fragment"`, `id="transfers-states"`, `id="transfers-count"`,
			`id="transfers-table"`, "<bdi>releve-20102026.pdf</bdi></a>")
		assert.Contains(t, rec.Header().Get("Set-Cookie"), "webui_transfers_list=state%3Derror",
			"the state the script loads is the one to find again when coming back")

		// Back to every state by the script: the state kept for the session is
		// cleared, so that the sidebar leads to every state (Driss, 2026-10-22).
		rec = s.get(t, permRead, "/webui/transfers?reset=1&fragment=list")
		assert.Contains(t, rec.Header().Get("Set-Cookie"), "webui_transfers_list=;")
		assert.Contains(t, rec.Header().Get("Set-Cookie"), "Max-Age=0")

		// The automatic refresh of the bare list says nothing about the state.
		rec = s.get(t, permRead, "/webui/transfers?fragment=list")
		assert.Empty(t, rec.Header().Values("Set-Cookie"))
	})

	t.Run("Given no transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permWrite, "/webui/transfers").Body.String()
		assert.Contains(t, body, "Aucun transfert pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/transfers/new"`)
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("Given a failed client transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, pathOf(s.failed.ID, "?ret=state%3Derror"))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "¿")
		id := strconv.FormatInt(s.failed.ID, 10)
		indexes(t, body, `<h1 class="w-page-header__title tf-title">releve-20102026.pdf</h1>`, "En erreur",
			"Transfert n° "+id, `href="/webui/transfers?state=error"`, "Retour à la liste",
			`action="/webui/transfers/`+id+`/resume?ret=state%3Derror&amp;from=sheet"`, "Reprendre",
			"Relancer", "Annuler le transfert")
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		indexes(t, body, `tf-error-card`, "<strong>Accès refusé</strong>", "TeForbidden",
			"failed to make transfer request: FICHIER DEJA RECU", "Préparation du transfert")
		indexes(t, body, "Règle", "releves", "Envoi · client", "Partenaire", "banque-demo", "Compte", "prod",
			"Client", "cli-sftp", "Protocole", "SFTP", "Identifiant de suivi", "2112951385528143872")
		indexes(t, body, "Tentatives restantes", "2", "Prochaine tentative", "21/10/2026 à 10:43:02")
		indexes(t, body, "Informations de transfert", "Identifiant de suivi", "__followID__", "lot", "2026-10-20-B")
	})

	t.Run("Given a done server transfer, read only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, pathOf(s.done.ID, "")).Body.String()
		indexes(t, body, ">commandes-0457.edi</h1>", "Terminé", "Réception · serveur", "Serveur", "srv-sftp",
			"Début", "21/10/2026 à 10:36:44", "Fin", "21/10/2026 à 10:36:45", "100 %", "48 Ko")
		assert.NotContains(t, body, "Tentatives restantes", "no retry for a server transfer")
		assert.NotContains(t, body, "tf-error-card")
		assert.NotContains(t, body, "Relancer", "no action in reading")
		assert.Contains(t, body, "Aucune information de transfert.")
		assert.NotContains(t, body, "Chemin prévu", "the paths of a transfer over are its own")
	})

	t.Run("Given a planned transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, pathOf(s.planned.ID, "")).Body.String()
		indexes(t, body, "Fichier", "Nom à la source", "rapport-mensuel.pdf",
			"Chemin prévu sur la passerelle", "rapport-mensuel.pdf",
			"Chemin prévu chez le partenaire", "rapport-mensuel.pdf", "Déroulement")
	})

	t.Run("Given an unknown transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.get(t, permRead, pathOf(4242, ""))
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Transfert introuvable")
	})
}

func TestActions(t *testing.T) {
	t.Parallel()

	t.Run("Pause", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, pathOf(s.planned.ID, "/pause?ret=state%3Dplanned"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=paused&name=rapport-mensuel.pdf&state=planned", rec.Header().Get("Location"))
		assert.Equal(t, types.StatusPaused, statusOfTransfer(t, s.db, s.planned.ID))

		rec = s.post(t, permWrite, pathOf(s.running.ID, "/pause?from=sheet"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, pathOf(s.running.ID, "?done=paused&name=paie-octobre.xml"), rec.Header().Get("Location"),
			"from the sheet, back to the sheet")
		assert.Equal(t, types.StatusPaused, statusOfTransfer(t, s.db, s.running.ID),
			"without its pipeline on a single node, as the REST API")

		rec = s.post(t, permWrite, pathOf(s.done.ID, "/pause"), nil)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "« commandes-0457.edi » ne peut pas être mis en pause")
		assert.Contains(t, rec.Body.String(), "Statut du transfert : Terminé.")
	})

	t.Run("Pause on another node", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.db.Config.NodeID = "gw-2"

		rec := s.post(t, permWrite, pathOf(s.running.ID, "/pause"), nil)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "autre nœud de la grappe")
		assert.Equal(t, types.StatusRunning, statusOfTransfer(t, s.db, s.running.ID))
	})

	t.Run("Resume", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, pathOf(s.failed.ID, "/resume"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=resumed&name=releve-20102026.pdf", rec.Header().Get("Location"))
		assert.Equal(t, types.StatusPlanned, statusOfTransfer(t, s.db, s.failed.ID))

		rec = s.post(t, permWrite, pathOf(s.available.ID, "/resume"), nil)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un transfert serveur reprend à l&#39;initiative du partenaire")

		rec = s.post(t, permWrite, pathOf(s.planned.ID, "/resume"), nil)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "Seul un transfert en pause, interrompu ou en erreur reprend.")
		assert.Contains(t, rec.Body.String(), "Statut du transfert : Planifié.")
	})

	t.Run("Cancel", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, pathOf(s.failed.ID, "/cancel?from=sheet"))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		indexes(t, body, "Annuler le transfert « releve-20102026.pdf » ?", "passe à l&#39;historique",
			"releve-20102026.pdf", "En erreur", "releves", "banque-demo",
			`<a class="w-btn w-btn--outline-dark" href="`+pathOf(s.failed.ID, "")+`">Annuler</a>`,
			`class="w-btn w-btn--danger"`, "Annuler le transfert</button>")

		rec = s.post(t, permWrite, pathOf(s.failed.ID, "/cancel?from=sheet"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, pathOf(s.failed.ID, "?done=cancelled&name=releve-20102026.pdf"), rec.Header().Get("Location"))
		assert.Equal(t, types.StatusCancelled, statusOfTransfer(t, s.db, s.failed.ID), "to the history")

		rec = s.get(t, permWrite, pathOf(s.cancelled.ID, "/cancel"))
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un transfert terminé ou annulé ne s&#39;annule plus.")
	})

	t.Run("Retry", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, pathOf(s.cancelled.ID, "/retry"))
		require.Equal(t, http.StatusOK, rec.Code)
		indexes(t, rec.Body.String(), "Relancer « virements.xml »", "Le Transfert n° 1002 reste tel quel",
			`type="datetime-local"`, "vide : maintenant", ">Relancer</button>")

		rec = s.post(t, permWrite, pathOf(s.cancelled.ID, "/retry"), url.Values{fieldDate: {"demain"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez une date et une heure.")

		rec = s.post(t, permWrite, pathOf(s.cancelled.ID, "/retry"), url.Values{fieldDate: {"2026-12-01T06:30"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		var created model.Transfers
		require.NoError(t, s.db.Select(&created).Where("src_filename=?", "virements.xml").Run())
		require.Len(t, created, 1)
		assert.Equal(t, types.StatusPlanned, created[0].Status)
		assert.True(t, created[0].Start.Equal(time.Date(2026, 12, 1, 6, 30, 0, 0, time.Local)))
		assert.Equal(t, pathOf(created[0].ID, "?done=retried&name="+strconv.FormatInt(created[0].ID, 10)),
			rec.Header().Get("Location"), "the sheet of the new transfer follows")

		rec = s.post(t, permWrite, pathOf(s.running.ID, "/retry"), nil)
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "ne peut pas être relancé")

		rec = s.get(t, permWrite, pathOf(s.done.ID, "/retry"))
		require.Equal(t, http.StatusConflict, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un transfert serveur se relance à l&#39;initiative du partenaire.")
	})

	t.Run("Retry of a transfer whose rule is gone", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		require.NoError(t, s.db.Insert(&model.HistoryEntry{
			Identifier: model.ID(1003), RemoteTransferID: "r-1003", IsSend: true, Rule: "disparue", Account: "prod", Agent: "banque-demo",
			Client: "cli-sftp", Protocol: "sftp", SrcFilename: "x.txt", DestFilename: "x.txt",
			Start: at(9, 0, 0), Stop: at(9, 0, 1), Status: types.StatusDone,
		}).Run())

		rec := s.post(t, permWrite, pathOf(1003, "/retry"), nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "n&#39;existe plus")
	})
}

func TestForms(t *testing.T) {
	t.Parallel()

	t.Run("Add a client transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, "/webui/transfers/new?ret=state%3Dplanned")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "¿")
		indexes(t, body, "Nouveau transfert", `<optgroup label="Envoi">`, ">releves</option>",
			`<optgroup label="Réception">`, ">commandes</option>", `<optgroup label="banque-demo">`, ">prod</option>",
			"Le client du protocole du partenaire", "cli-sftp · SFTP", `type="datetime-local"`)

		assert.Equal(t, 1, strings.Count(body, `name="info_name"`)-1, "one empty row, besides the template")

		rec = s.post(t, permWrite, "/webui/transfers/new?ret=state%3Dplanned",
			url.Values{fieldInfoName: {"", "lot"}, fieldInfoValue: {"sans nom", "B"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body = rec.Body.String()
		assert.Contains(t, body, "Le transfert n&#39;a pas été créé")
		assert.Contains(t, body, "Indiquez une valeur.")
		assert.Contains(t, body, "Choisissez une valeur dans la liste.")
		assert.Contains(t, body, "Indiquez le nom de l&#39;information 1.")
		indexes(t, body, `id="info-name-1"`, `value=""`, `aria-invalid="true"`, `value="sans nom"`,
			`id="info-name-2"`, `value="lot"`, `value="B"`, `id="info-name-3"`, `<template id="tf-info-row">`)

		rec = s.post(t, permWrite, "/webui/transfers/new",
			url.Values{fieldInfoName: {"lot", "lot"}, fieldInfoValue: {"A", "B"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "L&#39;information « lot » est indiquée deux fois.")

		// One more row without JavaScript: the form comes back as typed.
		rec = s.post(t, permWrite, "/webui/transfers/new", url.Values{
			fieldFile: {"x.pdf"}, fieldInfoName: {"lot"}, fieldInfoValue: {"B"}, fieldInfoAdd: {"1"},
		})
		require.Equal(t, http.StatusOK, rec.Code)
		body = rec.Body.String()
		assert.NotContains(t, body, "n&#39;a pas été créé", "nothing checked")
		indexes(t, body, `value="x.pdf"`, `id="info-name-1"`, `value="lot"`, `value="B"`, `id="info-name-2"`, `value=""`)
		n, err := s.db.Count(&model.Transfer{}).Where("src_filename=?", "x.pdf").Run()
		require.NoError(t, err)
		assert.Zero(t, n, "nothing saved")

		rec = s.post(t, permWrite, "/webui/transfers/new?ret=state%3Dplanned", url.Values{
			fieldFile: {"sortie/releve.pdf"}, fieldRule: {strconv.FormatInt(s.send.ID, 10)},
			fieldAccount: {strconv.FormatInt(s.prod.ID, 10)},
			fieldInfoName: {"lot", "", " flux "}, fieldInfoValue: {"B", "", "paie"},
			fieldTries: {"3"}, fieldDelay: {"60"}, fieldFactor: {"1,5"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		var created model.Transfer
		require.NoError(t, s.db.Get(&created, "src_filename=?", "sortie/releve.pdf").Run())
		assert.Equal(t, "/webui/transfers?done=added&name="+strconv.FormatInt(created.ID, 10)+"&state=planned",
			rec.Header().Get("Location"))
		assert.Equal(t, "releve.pdf", created.DestFilename, "the same name at the destination")
		assert.Equal(t, s.client.ID, created.ClientID.Int64, "the client of the protocol")
		assert.Equal(t, int8(3), created.RemainingTries)
		assert.Equal(t, int32(60), created.NextRetryDelay)
		assert.InDelta(t, 1.5, created.RetryIncrementFactor, 0.001)

		view := &model.NormalizedTransferView{}
		require.NoError(t, s.db.Get(view, "id=?", created.ID).Eager().Run())
		assert.Equal(t, "B", view.TransferInfo["lot"])
		assert.Equal(t, "paie", view.TransferInfo["flux"])
	})

	t.Run("Preregister a server transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, "/webui/transfers/preregister").Body.String()
		indexes(t, body, "Pré-enregistrer un transfert", "au statut « Attendu »", `<optgroup label="srv-sftp">`,
			">filiale-sud</option>", "Date limite")
		assert.NotContains(t, body, "Tentatives en cas d&#39;erreur", "no retry for a server transfer")

		rec := s.post(t, permWrite, "/webui/transfers/preregister", url.Values{
			fieldFile: {"inventaire-novembre.csv"}, fieldRule: {strconv.FormatInt(s.receive.ID, 10)},
			fieldAccount: {strconv.FormatInt(s.south.ID, 10)}, fieldStart: {""},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "the due date is required")

		rec = s.post(t, permWrite, "/webui/transfers/preregister", url.Values{
			fieldFile: {"inventaire-novembre.csv"}, fieldRule: {strconv.FormatInt(s.receive.ID, 10)},
			fieldAccount: {strconv.FormatInt(s.south.ID, 10)}, fieldStart: {"2020-01-01T06:00"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez une date limite à venir.", "as the REST API")

		rec = s.post(t, permWrite, "/webui/transfers/preregister", url.Values{
			fieldFile: {"inventaire-novembre.csv"}, fieldRule: {strconv.FormatInt(s.receive.ID, 10)},
			fieldAccount: {strconv.FormatInt(s.south.ID, 10)}, fieldStart: {"2026-11-02T06:00"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		var created model.Transfer
		require.NoError(t, s.db.Get(&created, "dest_filename=?", "inventaire-novembre.csv").Run())
		assert.Equal(t, types.StatusAvailable, created.Status)
		assert.Equal(t, sql.NullInt64{Int64: s.south.ID, Valid: true}, created.LocalAccountID)
	})
}

func TestMoment(t *testing.T) {
	t.Parallel()

	catalog := common.NewShell().Catalog()
	translate := catalog.Translator("fr")

	assert.Equal(t, moment{Text: "21/10/2026 10:38:02", Title: "21/10/2026 à 10:38:02"}, momentOf(translate, at(10, 38, 2)),
		"the date and the time to the second, on every line")
	assert.Equal(t, "31/12/2025 08:00:00", momentOf(translate, time.Date(2025, 12, 31, 8, 0, 0, 0, time.Local)).Text)
	assert.Equal(t, "2026-10-21 10:38:02", momentOf(catalog.Translator("en"), at(10, 38, 2)).Text)
	assert.Equal(t, moment{}, momentOf(translate, time.Time{}))
}

func TestDestination(t *testing.T) {
	t.Parallel()

	send := &model.NormalizedTransferView{
		IsSend: true, LocalPath: "/gw/out/a.csv", RemotePath: "in/a.csv", SrcFilename: "a.csv", DestFilename: "a.csv",
	}
	assert.Equal(t, "in/a.csv", destination(send), "a send: its path at the partner")

	receive := &model.NormalizedTransferView{
		LocalPath: "/gw/in/b.csv", RemotePath: "out/b.csv", SrcFilename: "b.csv", DestFilename: "b.csv",
	}
	assert.Equal(t, "/gw/in/b.csv", destination(receive), "a receive: its path on the gateway")

	planned := &model.NormalizedTransferView{IsSend: true, SrcFilename: "c.csv", DestFilename: "c-dest.csv"}
	assert.Equal(t, "c-dest.csv", destination(planned), "the name at the destination until the path is known")
	assert.Equal(t, "d.csv", destination(&model.NormalizedTransferView{SrcFilename: "d.csv"}))
}

// TestCatalogue checks that every state, every error code and every step of
// the product has its text in the catalogue: their keys are built at run
// time, the catalogue test cannot see them.
func TestCatalogue(t *testing.T) {
	t.Parallel()

	has := common.NewShell().Catalog().Has

	for _, spec := range stateSpecs() {
		assert.Truef(t, has(stateKeyPrefix+spec.key), "state %s", spec.key)
		assert.Truef(t, has(statesKeyPrefix+spec.key), "states %s", spec.key)
	}

	for code := types.TeOk + 1; code.IsValid(); code++ {
		assert.Truef(t, has(errorKeyPrefix+code.String()), "error code %s", code)
	}

	for step := types.StepSetup; step.IsValid(); step++ {
		assert.Truef(t, has(stepKeyPrefix+step.String()), "step %s", step)
	}

	for _, key := range sortKeys {
		assert.Truef(t, has(sortKeyPrefix+key), "sort %s", key)
	}

	for _, done := range doneKinds {
		assert.Truef(t, has(toastKeyPrefix+done), "notification %s", done)
	}
}
