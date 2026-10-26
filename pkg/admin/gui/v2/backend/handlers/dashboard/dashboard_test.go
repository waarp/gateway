package dashboard

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

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

// at is a moment of the days of the fixtures, in the time zone of the gateway.
func at(day, hour, minute int) time.Time {
	return time.Date(2026, 10, day, hour, minute, 0, 0, time.Local)
}

// sampleServices is a lister with a server in error.
func sampleServices() (core, servers, clients []internal.Service) {
	core = []internal.Service{
		{Name: "Database", State: utils.StateRunning.String()},
		{Name: "Controller", State: utils.StateRunning.String()},
	}
	servers = []internal.Service{
		{Name: "srv-sftp", State: utils.StateRunning.String()},
		{Name: "srv-r66", State: utils.StateError.String(), Reason: "bind: address already in use"},
	}
	clients = []internal.Service{{Name: "cli-sftp", State: utils.StateRunning.String()}}

	return core, servers, clients
}

type suite struct {
	db  *database.DB
	h   *Handlers
	now time.Time

	failedRecent, failedOld, failedServer *model.Transfer
}

// newSuite builds the handlers on a fresh base, the clock stopped at 16:05 on
// the 21st: the window runs from 17:00 on the 20th to the hour under way.
func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), now: at(21, 16, 5)}
	s.h = New(s.db, testhelpers.GetTestLogger(t), sampleServices)
	s.h.now = func() time.Time { return s.now }

	return s
}

// fixtures inserts a partner with its account and a client, a server with
// its account, two rules; live transfers of every state, three of them failed
// (one started half an hour ago, one three hours ago, a server one yesterday,
// before the window); history entries done in and out of the window, and one
// cancelled.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	bank := &model.RemoteAgent{Name: "banque-demo", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)}
	require.NoError(t, s.db.Insert(bank).Run())

	prod := &model.RemoteAccount{RemoteAgentID: bank.ID, Login: "prod"}
	require.NoError(t, s.db.Insert(prod).Run())

	client := &model.Client{Name: "cli-sftp", Protocol: "sftp"}
	require.NoError(t, s.db.Insert(client).Run())

	server := &model.LocalAgent{Name: "srv-sftp", Protocol: "sftp", Address: types.Addr("0.0.0.0", 2222)}
	require.NoError(t, s.db.Insert(server).Run())

	south := &model.LocalAccount{LocalAgentID: server.ID, Login: "filiale-sud"}
	require.NoError(t, s.db.Insert(south).Run())

	send := &model.Rule{Name: "releves", IsSend: true, Path: "/releves"}
	receive := &model.Rule{Name: "commandes", IsSend: false, Path: "/commandes"}
	require.NoError(t, s.db.Insert(send).Run())
	require.NoError(t, s.db.Insert(receive).Run())

	clientTransfer := func(file string, status types.TransferStatus, start time.Time) *model.Transfer {
		return &model.Transfer{
			RuleID: send.ID, ClientID: client.NullableID(), RemoteAccountID: prod.NullableID(),
			SrcFilename: file, DestFilename: file, Status: status, Start: start, Filesize: model.UnknownSize,
		}
	}

	s.failedRecent = clientTransfer("releve-20102026.pdf", types.StatusError, at(21, 15, 35))
	s.failedRecent.ErrCode, s.failedRecent.ErrDetails = types.TeConnection, "dial tcp 127.0.0.1:10022: connection refused"
	s.failedRecent.RemainingTries = 2

	s.failedOld = clientTransfer("paie-septembre.xml", types.StatusError, at(21, 13, 5))
	s.failedOld.ErrCode = types.TeBadAuthentication

	s.failedServer = &model.Transfer{
		RuleID: receive.ID, LocalAccountID: south.NullableID(), DestFilename: "commandes-0457.edi",
		Status: types.StatusError, Start: at(20, 10, 0), ErrCode: types.TeExternalOperation,
		ErrDetails: "post-task exit 2",
	}

	for _, trans := range []*model.Transfer{
		clientTransfer("paie-octobre.xml", types.StatusRunning, at(21, 15, 40)),
		clientTransfer("rapport-mensuel.pdf", types.StatusPlanned, at(21, 18, 0)),
		clientTransfer("rapport-annuel.pdf", types.StatusPlanned, at(21, 19, 0)),
		clientTransfer("archive.zip", types.StatusPaused, at(21, 9, 0)),
		clientTransfer("stocks.csv", types.StatusInterrupted, at(21, 11, 30)),
		{
			RuleID: receive.ID, LocalAccountID: south.NullableID(), DestFilename: "inventaire.csv",
			Status: types.StatusAvailable, Start: at(21, 23, 0),
		},
		s.failedRecent, s.failedOld, s.failedServer,
	} {
		require.NoError(t, s.db.Insert(trans).Run())
	}

	done := func(id int64, file string, start time.Time, size int64) *model.HistoryEntry {
		return &model.HistoryEntry{
			Identifier: model.ID(id), RemoteTransferID: "r-" + file, IsSend: true, Rule: "releves", Account: "prod",
			Agent: "banque-demo", Client: "cli-sftp", Protocol: "sftp", SrcFilename: file, DestFilename: file,
			Filesize: size, Progress: size, Start: start, Stop: start.Add(time.Minute), Status: types.StatusDone,
		}
	}

	cancelled := done(1005, "virements.xml", at(21, 13, 10), model.UnknownSize)
	cancelled.Status = types.StatusCancelled

	for _, hist := range []*model.HistoryEntry{
		done(1001, "releve-1.pdf", at(21, 15, 30), 1_000_000),
		done(1002, "releve-2.pdf", at(21, 14, 40), 2_000_000),
		done(1003, "releve-3.pdf", at(20, 17, 30), 500_000),  // the first hour of the window
		done(1004, "releve-4.pdf", at(20, 15, 0), 4_000_000), // before the window
		cancelled,
	} {
		require.NoError(t, s.db.Insert(hist).Run())
	}
}

// serve runs the page handler as a user with the given rights.
func (s *suite) serve(t *testing.T, perms model.PermsMask, lang, target string) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, lang)

	rec := httptest.NewRecorder()
	s.h.Page(rec, httptest.NewRequestWithContext(ctx, http.MethodGet, target, nil))

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

// part returns the markup of an element of the body, by its id: the element
// and what it contains, its nested elements of the same name counted.
func part(t *testing.T, body, id string) string {
	t.Helper()

	at := strings.Index(body, ` id="`+id+`"`)
	require.NotEqualf(t, -1, at, "part %q not found", id)

	open := strings.LastIndex(body[:at], "<")
	name := body[open+1 : open+1+strings.IndexAny(body[open+1:], " >")]
	depth := 0

	for i := open; i < len(body); i++ {
		switch {
		case strings.HasPrefix(body[i:], "<"+name+" ") || strings.HasPrefix(body[i:], "<"+name+">"):
			depth++
		case strings.HasPrefix(body[i:], "</"+name+">"):
			depth--
			if depth == 0 {
				return body[open : i+len(name)+3]
			}
		}
	}

	t.Fatalf("part %q is not closed", id)

	return ""
}

func TestWindowStart(t *testing.T) {
	t.Parallel()

	assert.Equal(t, at(20, 17, 0), windowStart(at(21, 16, 5)), "23 whole hours before the hour under way")
	assert.Equal(t, at(20, 1, 0), windowStart(at(21, 0, 59)))
}

func TestPage(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	read, write := model.PermTransfersRead, model.PermTransfersRead|model.PermTransfersWrite

	t.Run("Given a user who may act on the transfers, in French", func(t *testing.T) {
		rec := s.serve(t, write, "fr", "/webui/dashboard")
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Contains(t, body, "<title>Tableau de bord · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<h1 class="w-page-header__title">Tableau de bord</h1>`)
		assert.Contains(t, body, `<li><a href="/webui/home">Accueil</a></li><li><span>Supervision</span></li><li><span aria-current="page">Tableau de bord</span></li>`)
		assert.Contains(t, body, `id="tdb-live" role="status" aria-live="polite"`)
		assert.Contains(t, body, `>Actualisé à 16:05:00</span>`)
		assert.Contains(t, body, `id="tdb-refresh" href="/webui/dashboard">Actualiser</a>`)
		assert.Contains(t, body, `data-refresh-seconds="30" data-refreshed-at="16:05:00"`)
		assert.NotContains(t, body, "w-btn--primary", "no action of its own yet")
		assert.NotContains(t, body, "¿", "no missing key")

		// The services of the node: the tiles of the service status, as links.
		services := part(t, body, "tdb-services")
		assert.Contains(t, body, `<h2 class="w-section-title">Services du nœud gw-test</h2>`)
		assert.Equal(t, 3, strings.Count(services, `class="w-card w-tile`), "three groups")
		assert.NotContains(t, services, `role="tab"`, "links, not tabs")
		assert.Contains(t, services, `href="/webui/services?group=servers"`)
		assert.Contains(t, services, `w-tile--ko`, "a server in error")
		assert.Contains(t, services, `>1 en erreur</span>`)

		assert.Contains(t, body, `<h2 class="w-section-title">Transferts</h2>`)
	})

	t.Run("Given the tiles of the transfers", func(t *testing.T) {
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		tiles := part(t, body, "tdb-transfers")

		running := regexp.MustCompile(`(?s)<a class="w-card w-tile w-tile--info" href="/webui/transfers\?state=running">.*?</a>`).FindString(tiles)
		require.NotEmpty(t, running)
		assert.Contains(t, running, `<span class="w-tile__label">Transferts en cours</span>`)
		assert.Contains(t, running, `<span aria-hidden="true">1</span><span class="visually-hidden">1 transfert en cours</span>`)
		indexes(t, running, ">2 planifiés<", ">1 en pause<", ">1 interrompu<", ">1 attendu<")

		failed := regexp.MustCompile(`(?s)<a class="w-card w-tile w-tile--ko" href="/webui/transfers\?state=error">.*?</a>`).FindString(tiles)
		require.NotEmpty(t, failed)
		assert.Contains(t, failed, `<span aria-hidden="true">3</span><span class="visually-hidden">3 transferts en erreur</span>`)
		assert.Contains(t, failed, `<span class="w-status w-status--ko">1 depuis une heure</span>`)

		done := regexp.MustCompile(`(?s)<a class="w-card w-tile w-tile--ok" href="/webui/transfers\?from=2026-10-20T17%3A00&amp;state=done">.*?</a>`).FindString(tiles)
		require.NotEmpty(t, done, "the list of the done transfers is bounded by the window")
		assert.Contains(t, done, `<span class="w-tile__label">Terminés sur 24 h</span>`)
		assert.Contains(t, done, `<span aria-hidden="true">3</span>`, "the transfer of yesterday 15:00 is out, the one of 17:30 is in")
		assert.Contains(t, done, `<span class="w-status w-status--off">1 annulé</span>`)
		assert.Contains(t, done, `<span class="w-muted">3,3 Mo transférés</span>`)
	})

	t.Run("Given the activity of the window", func(t *testing.T) {
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()

		activity := part(t, body, "tdb-activity")
		assert.Contains(t, activity, `<svg class="w-chart w-chart--wide" viewBox="0 0 600 220" role="img"`)
		assert.Contains(t, activity, `<svg class="w-chart w-chart--narrow"`, "24 slots: a narrow rendering too")
		assert.Contains(t, activity, `<title id="tdb-activity-chart-t">Transferts par heure sur les 24 dernières heures : terminés et encore en erreur</title>`)
		assert.Contains(t, activity, "15:00 · 1 terminés", "the transfer done at 15:30 today")
		assert.Contains(t, activity, "15:00 · 1 en erreur", "the transfer failed at 15:35 today, hatched")
		assert.Contains(t, activity, "13:00 · 1 en erreur", "the transfer failed at 13:05")
		assert.Contains(t, activity, "17:00 · 1 terminés", "the transfer done yesterday at 17:30, first hour of the window")
		assert.NotContains(t, activity, "10:00 · 1 en erreur", "the server transfer failed yesterday morning is out of the window")
		assert.Contains(t, activity, `>Terminés</span>`)
		assert.Contains(t, activity, `>Encore en erreur (hachuré, chiffre au-dessus)</span>`)

		assert.Contains(t, body, `<details class="w-fold tdb-values" data-key="tdb-values">`)
		assert.NotContains(t, body, `<details class="w-fold tdb-values" data-key="tdb-values" open`, "closed by default")
		assert.Contains(t, body, `<h3 class="w-fold__title">Valeurs par heure</h3><span class="w-fold__meta" id="tdb-values-meta" data-tdb-part>3 terminés · 2 en erreur</span>`)

		values := part(t, body, "tdb-values")
		assert.Equal(t, 24, strings.Count(values, "<tr><td"), "one row per hour")
		indexes(t, values, `<td class="w-mono">17:00</td><td class="w-num">1</td><td class="w-num">0</td>`,
			`<td class="w-mono">13:00</td><td class="w-num">0</td><td class="w-num">1</td>`,
			`<td class="w-mono">15:00</td><td class="w-num">1</td><td class="w-num">1</td>`,
			`<td class="w-mono">16:00</td><td class="w-num">0</td><td class="w-num">0</td>`)
	})

	t.Run("Given the latest errors", func(t *testing.T) {
		body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
		errors := part(t, body, "tdb-errors")

		assert.Equal(t, 3, strings.Count(errors, "<tr><td"), "the three failed transfers")
		indexes(t, errors, "releve-20102026.pdf", "paie-septembre.xml", "commandes-0457.edi")
		assert.Contains(t, errors, `<span class="w-mono">15:35:00</span>`)
		assert.Contains(t, errors, `<div class="w-cell-stack"><span class="w-mono">10:00:00</span><span class="w-cell-sub">20/10/2026</span></div>`,
			"the date of a transfer not started today")
		assert.Contains(t, errors, `<a class="w-link" href="/webui/transfers/`+itoa(s.failedRecent.ID)+`">releve-20102026.pdf</a></span><span class="w-cell-sub">releves · banque-demo</span>`)
		assert.Contains(t, errors, `<span class="w-cell-sub">commandes · srv-sftp</span>`, "a server transfer names its server")
		assert.Contains(t, errors, `<span class="w-cell-sub tdb-detail" title="dial tcp 127.0.0.1:10022: connection refused">dial tcp 127.0.0.1:10022: connection refused</span>`)

		recent := regexp.MustCompile(`(?s)<div class="w-actions" id="tdb-act-` + itoa(s.failedRecent.ID) + `" popover>.*?</div>`).FindString(errors)
		require.NotEmpty(t, recent)
		indexes(t, recent, `href="/webui/transfers/`+itoa(s.failedRecent.ID)+`"`,
			`<form method="post" action="/webui/transfers/`+itoa(s.failedRecent.ID)+`/resume?from=dashboard">`,
			`href="/webui/transfers/`+itoa(s.failedRecent.ID)+`/retry?from=dashboard"`,
			`<hr>`, `href="/webui/transfers/`+itoa(s.failedRecent.ID)+`/cancel?from=dashboard"`)

		server := regexp.MustCompile(`(?s)<div class="w-actions" id="tdb-act-` + itoa(s.failedServer.ID) + `" popover>.*?</div>`).FindString(errors)
		require.NotEmpty(t, server)
		assert.NotContains(t, server, "/resume", "a server transfer resumes on the initiative of the partner")
		assert.NotContains(t, server, "/retry")
		assert.Contains(t, server, "/cancel?from=dashboard")

		assert.Contains(t, errors, `<p class="tdb-foot"><a class="w-link" href="/webui/transfers?state=error">Voir les 3 transferts en erreur</a></p>`)
	})

	t.Run("Given a user who reads the transfers only", func(t *testing.T) {
		body := s.serve(t, read, "en", "/webui/dashboard").Body.String()
		errors := part(t, body, "tdb-errors")

		assert.NotContains(t, errors, "popover", "no menu without the right to write")
		assert.Equal(t, 3, strings.Count(errors, `aria-label="View the transfer"`), "the sheet alone")
		assert.Contains(t, body, "Services of node gw-test")
		assert.Contains(t, body, `<span class="w-tile__label">Done in 24 h</span>`)
		assert.Contains(t, body, `>3 done · 2 failed</span>`)
		assert.Contains(t, body, `>View the 3 failed transfers</a>`)
	})

	t.Run("Given a user without the right to read the transfers", func(t *testing.T) {
		body := s.serve(t, 0, "fr", "/webui/dashboard").Body.String()

		assert.Contains(t, body, `id="tdb-services"`)
		assert.NotContains(t, body, `id="tdb-transfers"`)
		assert.NotContains(t, body, "tdb-errors")
		assert.NotContains(t, part(t, body, "tdb-notice"), "w-banner", "no message: the sidebar does not offer the transfers either")
	})

	t.Run("Given a request for the body fragment", func(t *testing.T) {
		rec := s.serve(t, write, "fr", "/webui/dashboard?fragment=body")
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.NotContains(t, body, "<html")
		assert.True(t, strings.HasPrefix(body, `<div id="tdb-body"`), "the body alone")
		assert.Contains(t, body, `id="tdb-errors"`)
	})
}

func TestEmptyPage(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	body := s.serve(t, model.PermTransfersRead|model.PermTransfersWrite, "fr", "/webui/dashboard").Body.String()

	tiles := part(t, body, "tdb-transfers")
	assert.Contains(t, tiles, `w-tile--ok" href="/webui/transfers?state=error"`, "no failure: the tile is fine")
	assert.Contains(t, tiles, `<span class="w-status w-status--ok">Aucun transfert en erreur</span>`)
	assert.Equal(t, 3, strings.Count(tiles, `<span aria-hidden="true">0</span>`))

	activity := part(t, body, "tdb-activity")
	assert.NotContains(t, activity, "w-chart", "no chart without any transfer")
	assert.Contains(t, activity, `<p class="w-empty__title">Aucun transfert sur les 24 dernières heures</p>`)
	assert.NotContains(t, body, `data-key="tdb-values"`, "no values either")

	errors := part(t, body, "tdb-errors")
	assert.Contains(t, errors, `<p class="w-empty__title">Aucun transfert en erreur</p>`)
	assert.NotContains(t, errors, "<table")
}

func TestFiguresCache(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	figs, last := s.h.figures(s.now)
	require.NotNil(t, figs)
	assert.EqualValues(t, 3, figs.Live[statusKey(types.StatusError)])
	assert.Equal(t, s.now, last.at)
	assert.True(t, last.failing.IsZero())

	// One failure more: the figures are those of the period until it ends.
	require.NoError(t, s.db.Insert(&model.Transfer{
		RuleID: s.failedOld.RuleID, ClientID: s.failedOld.ClientID, RemoteAccountID: s.failedOld.RemoteAccountID,
		SrcFilename: "late.csv", DestFilename: "late.csv", Status: types.StatusError, Start: at(21, 16, 0), Filesize: model.UnknownSize,
	}).Run())

	figs, _ = s.h.figures(s.now.Add(20 * time.Second))
	assert.EqualValues(t, 3, figs.Live[statusKey(types.StatusError)], "read once per period")

	figs, last = s.h.figures(s.now.Add(31 * time.Second))
	assert.EqualValues(t, 4, figs.Live[statusKey(types.StatusError)], "read again once the period is over")
	assert.Equal(t, s.now.Add(31*time.Second), last.at)
}

func TestDegradedPage(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	write := model.PermTransfersRead | model.PermTransfersWrite

	first := s.serve(t, write, "fr", "/webui/dashboard").Body.String()
	assert.NotContains(t, part(t, first, "tdb-notice"), "w-banner", "the base answers: no message")

	// The base stops answering: the page keeps its figures, dated, and says
	// since when the base fails.
	s.h.read = func(time.Time) (*figures, error) { return nil, errors.New("database is closed") }

	s.now = at(21, 16, 7)
	body := s.serve(t, write, "fr", "/webui/dashboard").Body.String()

	notice := part(t, body, "tdb-notice")
	assert.Contains(t, notice, `<div class="w-banner w-banner--ko" role="alert">`)
	assert.Contains(t, notice, `<strong>La base de données ne répond plus depuis 16:07:00.</strong>Les chiffres affichés datent de 16:05:00.`)
	assert.Contains(t, notice, `href="/webui/services">`)
	assert.Contains(t, body, `<span aria-hidden="true">3</span><span class="visually-hidden">3 transferts en erreur</span>`,
		"the last figures read stay")
	assert.Contains(t, body, ">Actualisé à 16:07:00</span>", "the page says when it was rendered")

	// Later, still failing: the first failure is the one named.
	s.now = at(21, 16, 9)
	body = s.serve(t, write, "fr", "/webui/dashboard").Body.String()
	assert.Contains(t, body, "La base de données ne répond plus depuis 16:07:00.")

	// A base that never answered: the message alone, without figures.
	fresh := newSuite(t)
	fresh.h.read = func(time.Time) (*figures, error) { return nil, errors.New("database is closed") }
	body = fresh.serve(t, write, "fr", "/webui/dashboard").Body.String()
	assert.Contains(t, body, "Aucun chiffre n&#39;a encore pu être lu.")
	assert.Contains(t, body, `id="tdb-services"`, "the services, read in memory, stay")
	assert.NotContains(t, body, `id="tdb-transfers"`)
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }
