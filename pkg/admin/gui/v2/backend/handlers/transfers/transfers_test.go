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
		indexes(t, body, `id="transfers-states"`, `aria-current="true"><span>Tous</span> <strong>6</strong>`,
			"<span>En cours</span> <strong>1</strong>", "<span>En erreur</span> <strong>1</strong>",
			"<span>Planifiés</span> <strong>1</strong>", "<span>En pause</span> <strong>0</strong>",
			"<span>Interrompus</span> <strong>0</strong>", "<span>Disponibles</span> <strong>1</strong>",
			"<span>Terminés</span> <strong>1</strong>", "<span>Annulés</span> <strong>1</strong>", "</nav>")
		assert.Contains(t, body, `id="transfers-count" aria-live="polite">6 transferts</span>`)

		// The most recent first; the file first, the number and the protocol under it.
		assert.Equal(t, 6, rows(body))
		indexes(t, body, "<tbody>", ">inventaire.csv</a>", ">rapport-mensuel.pdf</a>", ">paie-octobre.xml</a>",
			">releve-20102026.pdf</a>", ">commandes-0457.edi</a>", ">virements.xml</a>", "</tbody>")
		indexes(t, body, ">releve-20102026.pdf</a><span class=\"tf-sub\">n° "+strconv.FormatInt(s.failed.ID, 10)+" · SFTP</span>",
			"En erreur", `<span class="tf-sub tf-cause">Accès refusé</span>`,
			`<span class="w-mono">releves</span><span class="tf-sub">Envoi · client</span>`,
			`banque-demo<span class="tf-sub w-mono">prod</span>`, "10:38:02", "10:43:02", "prochaine tentative")
		indexes(t, body, ">paie-octobre.xml</a>", `<progress class="tf-progress" max="100" value="64"`, "64 %", "1,2 Mo")
		indexes(t, body, ">inventaire.csv</a>", "Disponible", "Réception · serveur", "srv-sftp", "jusqu&#39;au 21/10/2026")
		indexes(t, body, ">rapport-mensuel.pdf</a>", "Planifié", "18:00:00", "21/10/2026", "</tr>")
		assert.NotContains(t, body, "0 octet", "no size before the transfer starts")

		// The actions the state allows, in the menu.
		indexes(t, body, ">paie-octobre.xml</a>", "Voir le transfert", "Mettre en pause", "Annuler le transfert",
			">releve-20102026.pdf</a>")
		indexes(t, body, ">releve-20102026.pdf</a>", "Voir le transfert", "Reprendre", "Relancer", "Annuler le transfert",
			">commandes-0457.edi</a>")
		indexes(t, body, ">commandes-0457.edi</a>", "Voir le transfert", ">virements.xml</a>", "Relancer", "</tbody>")
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
		assert.Contains(t, body, ">releve-20102026.pdf</a>")
		assert.Contains(t, body, `href="/webui/transfers?state=error" aria-current="true"><span>En erreur</span>`)
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
		assert.Contains(t, body, ">paie-octobre.xml</a>")
		assert.Contains(t, body, "<span>Tous</span> <strong>1</strong>", "the states count what the search finds")

		body = s.get(t, permRead, "/webui/transfers?q=2112951385528143872").Body.String()
		assert.Equal(t, 1, rows(body), "by the follow ID")
		assert.Contains(t, body, ">releve-20102026.pdf</a>")

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
		indexes(t, body, "<tbody>", ">commandes-0457.edi</a>", ">inventaire.csv</a>", ">paie-octobre.xml</a>",
			">rapport-mensuel.pdf</a>", ">releve-20102026.pdf</a>", ">virements.xml</a>")

		body = s.get(t, permRead, "/webui/transfers?sort=state&dir=asc").Body.String()
		indexes(t, body, "<tbody>", ">paie-octobre.xml</a>", ">releve-20102026.pdf</a>", ">rapport-mensuel.pdf</a>",
			">inventaire.csv</a>", ">commandes-0457.edi</a>", ">virements.xml</a>")

		body = s.get(t, permRead, "/webui/transfers?sort=start&dir=asc").Body.String()
		indexes(t, body, "<tbody>", ">virements.xml</a>", ">commandes-0457.edi</a>", ">rapport-mensuel.pdf</a>")
		assert.Contains(t, body, `aria-sort="ascending"`)
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
			`id="transfers-table"`, ">releve-20102026.pdf</a>")
		assert.Empty(t, rec.Header().Values("Set-Cookie"), "the refresh does not keep the state")
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
		assert.Contains(t, rec.Body.String(), "État du transfert : Terminé.")
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
		assert.Contains(t, rec.Body.String(), "État du transfert : Planifié.")
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

		rec = s.post(t, permWrite, "/webui/transfers/new?ret=state%3Dplanned", url.Values{fieldInfo: {"sans deux-points"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body = rec.Body.String()
		assert.Contains(t, body, "Le transfert n&#39;a pas été créé")
		assert.Contains(t, body, "Indiquez une valeur.")
		assert.Contains(t, body, "Choisissez une valeur dans la liste.")
		assert.Contains(t, body, "Ligne à corriger : « sans deux-points »")

		rec = s.post(t, permWrite, "/webui/transfers/new?ret=state%3Dplanned", url.Values{
			fieldFile: {"sortie/releve.pdf"}, fieldRule: {strconv.FormatInt(s.send.ID, 10)},
			fieldAccount: {strconv.FormatInt(s.prod.ID, 10)}, fieldInfo: {"lot: B\n\nflux : paie"},
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
		indexes(t, body, "Pré-enregistrer un transfert", "à l&#39;état « Disponible »", `<optgroup label="srv-sftp">`,
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
