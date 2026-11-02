package transfers

import (
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func id(trans interface{ GetID() int64 }) string { return strconv.FormatInt(trans.GetID(), 10) }

// TestSetSelection checks the row selection of the list (#792): to whoever
// may write, the bar under the toolbar with the state of the list as hidden
// fields and the three gestures, the column of checkboxes, each named after
// its transfer, the total of the filter on the count; nothing of it to a
// reader.
func TestSetSelection(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permWrite, "/webui/transfers?full=1&state=error&rule=releves")
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	indexes(t, body, `id="transfers-pager-top"`, `<form class="w-selection" id="sel-transfers" method="post" action="/webui/transfers/resume"`,
		`<input type="hidden" name="rule" value="releves">`, `<input type="hidden" name="state" value="error">`,
		`<input type="checkbox" name="all" value="1">`, `Sélectionner les 1 transferts du filtre`,
		`formaction="/webui/transfers/resume">Reprendre les transferts</button>`,
		`formaction="/webui/transfers/retry" formmethod="get">Relancer les transferts</button>`,
		`formaction="/webui/transfers/cancel" formmethod="get">Annuler les transferts</button>`,
		`id="transfers-table"`)
	assert.Contains(t, body, `data-select-total="sel-transfers" data-total="1">1 transfert</span>`, "the total of the filter")
	assert.Contains(t, body, `<table class="w-table w-table--one-line tf-table w-table--select">`)
	assert.Contains(t, body, `<col class="tf-col-select">`)
	assert.Contains(t, body, `data-select-page="sel-transfers"`, "the header checkbox")
	assert.Contains(t, body, `<input type="checkbox" name="ids" value="`+id(s.failed)+`" form="sel-transfers" data-select-row> `+
		`<span class="visually-hidden">Sélectionner le transfert n° `+id(s.failed)+`</span>`)
	assert.NotContains(t, body, `value="`+id(s.running)+`" form="sel-transfers"`, "the rows of the filter alone")
	assert.Equal(t, 1, countOf(body, "w-btn--primary"), "the bar adds no coloured action")

	rec = s.get(t, permRead, "/webui/transfers?full=1")
	require.Equal(t, http.StatusOK, rec.Code)
	body = rec.Body.String()
	assert.NotContains(t, body, "w-selection", "no bar to a reader")
	assert.NotContains(t, body, "w-table--select")
	assert.NotContains(t, body, "data-select-row")
	assert.Contains(t, body, `data-select-total="sel-transfers" data-total="6">6 transferts</span>`)
}

func countOf(body, marker string) int {
	n := 0

	for i := 0; ; n++ {
		j := indexFrom(body, marker, i)
		if j < 0 {
			return n
		}

		i = j + len(marker)
	}
}

func indexFrom(body, marker string, from int) int {
	for i := from; i+len(marker) <= len(body); i++ {
		if body[i:i+len(marker)] == marker {
			return i
		}
	}

	return -1
}

// TestSetResume checks the grouped resumption: the transfers of the filters
// resume at once when the page said their count, minus the exceptions or the
// rows checked alone; a changed or unknown count asks on a page; what the
// model refuses is counted; an identifier out of the filters is ignored.
func TestSetResume(t *testing.T) {
	t.Parallel()

	t.Run("the whole filter, at once", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/transfers/resume",
			url.Values{"state": {"error"}, "all": {"1"}, "expected": {"1"}, "sort": {"id"}, "dir": {"asc"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?dir=asc&done=set_resumed&name=1&sort=id&state=error", rec.Header().Get("Location"),
			"back to the list as it was, with the result")
		assert.Equal(t, types.StatusPlanned, statusOfTransfer(t, s.db, s.failed.ID))
		assert.Equal(t, types.StatusPlanned, statusOfTransfer(t, s.db, s.planned.ID), "a transfer out of the filter is intact")
		assert.Equal(t, types.StatusRunning, statusOfTransfer(t, s.db, s.running.ID))
	})

	t.Run("the whole filter minus the exceptions", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		other := &model.Transfer{
			RuleID: s.send.ID, ClientID: s.client.NullableID(), RemoteAccountID: s.prod.NullableID(),
			SrcFilename: "releve-2.pdf", Status: types.StatusError, Start: at(10, 39, 0), Filesize: model.UnknownSize,
		}
		require.NoError(t, s.db.Insert(other).Run())

		rec := s.post(t, permWrite, "/webui/transfers/resume",
			url.Values{"state": {"error"}, "all": {"1"}, "except": {id(other), "7777"}, "expected": {"1"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=set_resumed&name=1&state=error", rec.Header().Get("Location"))
		assert.Equal(t, types.StatusPlanned, statusOfTransfer(t, s.db, s.failed.ID))
		assert.Equal(t, types.StatusError, statusOfTransfer(t, s.db, other.ID), "excepted")
	})

	t.Run("the rows checked, those the model refuses counted", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/transfers/resume",
			url.Values{"ids": {id(s.failed), id(s.planned), id(s.done), "abc"}, "expected": {"3"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=set_resumed&failed=2&name=1&reason=state%3A2", rec.Header().Get("Location"))
		assert.Equal(t, types.StatusPlanned, statusOfTransfer(t, s.db, s.failed.ID))
		assert.Equal(t, types.StatusDone, statusOfTransfer(t, s.db, s.done.ID))

		rec = s.get(t, permWrite, "/webui/transfers?full=1&done=set_resumed&failed=2&name=1&reason=state:2")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "1 transfert repris")
		assert.Contains(t, rec.Body.String(), "2 non repris : Transferts dont le statut ne permet pas ce geste : 2")
	})

	t.Run("a row checked out of the filter is ignored", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/transfers/resume",
			url.Values{"state": {"error"}, "ids": {id(s.planned)}, "expected": {"1"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=set_empty&state=error", rec.Header().Get("Location"), "nothing selected")
		assert.Equal(t, types.StatusPlanned, statusOfTransfer(t, s.db, s.planned.ID))

		rec = s.get(t, permWrite, "/webui/transfers?full=1&done=set_empty")
		assert.Contains(t, rec.Body.String(), "Aucun transfert sélectionné")

		rec = s.post(t, permWrite, "/webui/transfers/resume", url.Values{"state": {"error"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=set_empty&state=error", rec.Header().Get("Location"), "an empty selection")
	})

	t.Run("a changed count asks", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/transfers/resume",
			url.Values{"state": {"error"}, "all": {"1"}, "expected": {"4"}, "except": {id(s.running)}})
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Reprendre 1 transfert ? · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "L&#39;ensemble a changé depuis l&#39;affichage")
		assert.Contains(t, body, "Le nombre est passé de 4 à 1.")
		indexes(t, body, `<a class="w-btn w-btn--outline-dark" href="/webui/transfers?state=error">`,
			`<form class="w-form" method="post" action="/webui/transfers/resume"`,
			`<input type="hidden" name="state" value="error">`, `<input type="hidden" name="all" value="1">`,
			`<input type="hidden" name="expected" value="1">`,
			"<dt>Statut</dt>", "<dd>En erreur</dd>", "<dt>Nombre</dt>", "<dd>1 transfert</dd>",
			"<dt>Seront repris</dt>", `<dd class="w-num">1</dd>`, `Reprendre les transferts</button>`)
		assert.NotContains(t, body, "<dt>Sauf</dt>", "an exception out of the filter is not named")
		assert.Equal(t, types.StatusError, statusOfTransfer(t, s.db, s.failed.ID), "nothing resumed")

		rec = s.post(t, permWrite, "/webui/transfers/resume", url.Values{"state": {"error"}, "all": {"1"}})
		require.Equal(t, http.StatusOK, rec.Code, "without the count (no JavaScript), the page asks")
		assert.NotContains(t, rec.Body.String(), "a changé")
	})

	t.Run("the page says what the gesture leaves aside", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/transfers/resume", url.Values{"all": {"1"}, "q": {"*"}})
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Reprendre 6 transferts ? · gw-test · Waarp Gateway</title>")
		indexes(t, body, "<dt>Statut</dt>", "<dd>Tous</dd>", "<dt>Recherche</dt>", "<dd>*</dd>", "<dt>Nombre</dt>", "<dd>6 transferts</dd>",
			"<dt>Seront repris</dt>", `<dd class="w-num">1</dd>`,
			"<dt>Transferts serveur, à l&#39;initiative du partenaire</dt>", `<dd class="w-num">1</dd>`,
			"<dt>Transferts dont le statut ne permet pas ce geste</dt>", `<dd class="w-num">4</dd>`)
	})

	t.Run("GET is not an action", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.get(t, permWrite, "/webui/transfers/resume?all=1")
		assert.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, rec.Code)
	})
}

// TestSetCancel checks the grouped cancellation: its confirmation page names
// the set, the count and the running transfers; the transfers go to the
// history, cancelled; a transfer running on another node is left to it.
func TestSetCancel(t *testing.T) {
	t.Parallel()

	t.Run("the page, then the cancellation", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, "/webui/transfers/cancel?all=1&partner=banque-demo&from=2026-10-21T10%3A00")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Annuler 4 transferts ? · gw-test · Waarp Gateway</title>",
			"the three live transfers of the partner since 10:00 and its cancelled one")
		indexes(t, body, `<form class="w-form" method="post" action="/webui/transfers/cancel"`,
			`<input type="hidden" name="from" value="2026-10-21T10:00">`, `<input type="hidden" name="partner" value="banque-demo">`,
			"<dt>Statut</dt>", "<dd>Tous</dd>", "<dt>Partenaire ou serveur</dt>", "<dd>banque-demo</dd>",
			"<dt>Période de début</dt>", "<dd>à partir du 21/10/2026 à 10:00:00</dd>", "<dd>4 transferts</dd>",
			"<dt>Seront annulés</dt>", `<dd class="w-num">3</dd>`, "<dt>Dont en cours, interrompus</dt>", `<dd class="w-num">1</dd>`,
			"<dt>Transferts dont le statut ne permet pas ce geste</dt>", `<dd class="w-num">1</dd>`,
			`class="w-btn w-btn--danger"`, "Annuler les transferts</button>")
		assert.NotContains(t, body, "w-btn--primary", "a destructive page")

		rec = s.post(t, permWrite, "/webui/transfers/cancel", url.Values{"all": {"1"}, "partner": {"banque-demo"}, "from": {"2026-10-21T10:00"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=set_cancelled&failed=1&from=2026-10-21T10%3A00&name=3&partner=banque-demo&reason=state%3A1",
			rec.Header().Get("Location"))

		for _, trans := range []*model.Transfer{s.running, s.failed, s.planned} {
			assert.Equal(t, types.StatusCancelled, statusOfTransfer(t, s.db, trans.ID))

			var count uint64
			count, err := s.db.Count(&model.Transfer{}).Where("id=?", trans.ID).Run()
			require.NoError(t, err)
			assert.Zero(t, count, "moved to the history")
		}

		assert.Equal(t, types.StatusAvailable, statusOfTransfer(t, s.db, s.available.ID), "the server transfer is out of the filter")

		var entry model.HistoryEntry
		require.NoError(t, s.db.Get(&entry, "id=?", s.failed.ID).Eager().Run())
		assert.Equal(t, "2026-10-20-B", entry.TransferInfo["lot"], "the information follows")
	})

	t.Run("a transfer running on another node", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.db.Config.NodeID = "gw-2"

		rec := s.post(t, permWrite, "/webui/transfers/cancel", url.Values{"all": {"1"}})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/transfers?done=set_cancelled&failed=3&name=3&reason=state%3A2%2Celsewhere%3A1",
			rec.Header().Get("Location"))
		assert.Equal(t, types.StatusRunning, statusOfTransfer(t, s.db, s.running.ID))
		assert.Equal(t, types.StatusCancelled, statusOfTransfer(t, s.db, s.available.ID))

		rec = s.get(t, permWrite, "/webui/transfers?full=1&done=set_cancelled&failed=3&name=3&reason=state:2,elsewhere:1")
		assert.Contains(t, rec.Body.String(), "3 transferts annulés")
		assert.Contains(t, rec.Body.String(), "3 non annulés : Transferts dont le statut ne permet pas ce geste : 2 · "+
			"Transferts en cours sur un autre nœud de la grappe : 1")
	})
}

// TestSetRetry checks the grouped retry: its page, with the date; a new
// transfer for each transfer of the set, at the date; an invalid date is
// refused on the page.
func TestSetRetry(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permWrite, "/webui/transfers/retry?ids="+id(s.done)+"&ids="+id(s.failed)+"&ids="+id(s.running))
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "<title>Relancer 3 transferts · gw-test · Waarp Gateway</title>")
	// The identifiers of the set, by identifier.
	indexes(t, body, `<input type="hidden" name="ids" value="`+id(s.running)+`">`, `<input type="hidden" name="ids" value="`+id(s.failed)+`">`,
		`<input type="hidden" name="ids" value="`+id(s.done)+`">`, `<input type="hidden" name="expected" value="3">`,
		"<dt>Seront relancés</dt>", `<dd class="w-num">1</dd>`,
		"<dt>Transferts serveur, à l&#39;initiative du partenaire</dt>", `<dd class="w-num">1</dd>`,
		"<dt>Transferts dont le statut ne permet pas ce geste</dt>", `<dd class="w-num">1</dd>`,
		`name="date"`, "Relancer les transferts</button>")

	before, err := s.db.Count(&model.Transfer{}).Where("status=?", types.StatusPlanned).Run()
	require.NoError(t, err)

	rec = s.post(t, permWrite, "/webui/transfers/retry",
		url.Values{"ids": {id(s.done), id(s.failed), id(s.running)}, "date": {"2026-10-22T09:00"}})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/webui/transfers?done=set_retried&failed=2&name=1&reason=server%3A1%2Cstate%3A1", rec.Header().Get("Location"),
		"the failed client transfer alone: the done one is a server transfer, the running one has not stopped")

	var planned model.Transfers
	require.NoError(t, s.db.Select(&planned).Where("status=?", types.StatusPlanned).OrderBy("id", true).Run())
	require.Len(t, planned, int(before)+1) //nolint:gosec // a count
	assert.Equal(t, at(9, 0, 0).AddDate(0, 0, 1), planned[len(planned)-1].Start.Local(), "at the date")
	assert.Equal(t, "releve-20102026.pdf", planned[len(planned)-1].SrcFilename, "the retry of the failed one")
	assert.Equal(t, types.StatusError, statusOfTransfer(t, s.db, s.failed.ID), "the transfer of origin stays")

	rec = s.post(t, permWrite, "/webui/transfers/retry", url.Values{"ids": {id(s.failed)}, "date": {"soon"}})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "is-invalid")
}

// TestSetParseIDs checks the reading of the identifiers a form sends.
func TestSetParseIDs(t *testing.T) {
	t.Parallel()

	assert.Equal(t, []int64{12, 7}, parseIDs([]string{"12", "x", "", "-3", "0", "7"}))
	assert.Empty(t, parseIDs(nil))
}
