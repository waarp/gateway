package transfers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// The keys as the gateway writes them: the partner option or a SETINFO
// pre-task set the wait, the pipeline its start, the PeSIT module the
// reception, the SENDMESSAGE task the one sent.
func (s *suite) acknowledgements(t *testing.T) {
	t.Helper()

	s.running.TransferInfo = map[string]any{
		model.InfoAckExpected: true, model.InfoAckWaitTimeout: "5m", model.InfoAckWaitSince: at(10, 42, 13),
	}
	s.planned.TransferInfo = map[string]any{model.InfoAckExpected: true}
	s.failed.TransferInfo[model.InfoAckExpected] = true
	s.failed.TransferInfo[model.InfoAckWaitSince] = at(10, 38, 30)

	for _, trans := range []*model.Transfer{s.running, s.planned, s.failed} {
		require.NoError(t, trans.UpdateInfo(s.db))
	}

	s.done.TransferInfo = map[string]any{
		model.InfoAckReceived: true, model.InfoAckReceivedOn: at(10, 36, 50),
		model.InfoAckSent: true, model.InfoAckSentOn: at(10, 36, 55).Format(time.RFC3339),
		model.InfoAckSentTo: "banque-demo", model.InfoAckSentAs: "prod",
	}
	s.cancelled.TransferInfo = map[string]any{
		model.InfoAckSent: true, model.InfoAckSentOn: at(10, 13, 2).Format(time.RFC3339),
		model.InfoAckSentTo: "banque-demo", model.InfoAckSentAs: "prod",
	}

	for _, hist := range []*model.HistoryEntry{s.done, s.cancelled} {
		require.NoError(t, hist.UpdateInfo(s.db))
	}
}

// The column "ACK" says the acknowledgement of each transfer in one word,
// the detail in the tooltip; the filter keeps the transfers of a case.
func TestListAck(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.acknowledgements(t)

	body := s.get(t, permRead, "/webui/transfers?full=1").Body.String()
	assert.NotContains(t, body, "¿", "every key must be in the catalogue")
	indexes(t, body, `<col class="tf-col-date"><col class="tf-col-date"><col class="tf-col-ack"><col class="tf-col-progress">`,
		`dir=asc">Fin</a></th>`, `<th scope="col">ACK</th>`, `<th scope="col">Progression</th>`)

	// Each case: the file, then the cell after the end, its tooltip, its shape and its word.
	indexes(t, body, "<bdi>paie-octobre.xml</bdi>",
		`<td class="tf-ack-cell" title="Acquittement du partenaire attendu jusqu&#39;au 21/10/2026 à 10:47:13 : `+
			`délai de 5 min, en attente depuis le 21/10/2026 à 10:42:13">`, `href="#clock"`, "<span>Attendu</span>", "</tr>")
	indexes(t, body, "<bdi>rapport-mensuel.pdf</bdi>",
		`<td class="tf-ack-cell" title="La passerelle attendra l&#39;acquittement du partenaire jusqu&#39;à 10 min `+
			`après le transfert">`, `href="#clock"`, "<span>Attendu</span>", "</tr>")
	indexes(t, body, "<bdi>releve-20102026.pdf</bdi>",
		`<td class="tf-ack-cell" title="Aucun acquittement du partenaire dans le délai de 10 min, attendu depuis le `+
			`21/10/2026 à 10:38:30">`, `href="#xmark"`, "<span>Non reçu</span>", "</tr>")
	indexes(t, body, "<bdi>commandes-0457.edi</bdi>",
		`<td class="tf-ack-cell" title="Acquittement du partenaire reçu le 21/10/2026 à 10:36:50 · Acquittement envoyé `+
			`le 21/10/2026 à 10:36:55 au partenaire banque-demo, en tant que prod">`, `href="#check"`, "<span>Reçu</span>", "</tr>")
	indexes(t, body, "<bdi>virements.xml</bdi>",
		`<td class="tf-ack-cell" title="Acquittement envoyé le 21/10/2026 à 10:13:02 au partenaire banque-demo, `+
			`en tant que prod">`, `href="#envelope"`, "<span>Envoyé</span>", "</tr>")
	indexes(t, body, "<bdi>inventaire.csv</bdi>",
		`<td class="tf-ack-cell" title="Sans acquittement"><span class="w-muted" aria-hidden="true">—</span>`+
			`<span class="visually-hidden">Sans acquittement</span></td>`, "</tr>")

	// The filter, after the protocol, applies in place like the others.
	indexes(t, body, `for="f-protocol"`, `<label class="w-field__label" for="f-ack">Acquittement</label>`,
		`<select class="w-select" id="f-ack" name="ack"><option value="">Tous les transferts</option>`,
		`<option value="none">Sans acquittement</option>`, `<option value="expected">Attendu</option>`,
		`<option value="received">Reçu</option>`, `<option value="missed">Non reçu</option>`,
		`<option value="sent">Envoyé</option></select>`, `for="f-from"`)

	for _, tc := range []struct {
		ack   string
		files []string
	}{
		{"expected", []string{"rapport-mensuel.pdf", "paie-octobre.xml"}},
		{"received", []string{"commandes-0457.edi"}},
		{"missed", []string{"releve-20102026.pdf"}},
		{"sent", []string{"commandes-0457.edi", "virements.xml"}},
		{"none", []string{"inventaire.csv"}},
	} {
		filtered := s.get(t, permRead, "/webui/transfers?full=1&ack="+tc.ack).Body.String()
		assert.Equalf(t, len(tc.files), rows(filtered), "ack=%s", tc.ack)

		for _, file := range tc.files {
			assert.Containsf(t, filtered, "<bdi>"+file+"</bdi>", "ack=%s", tc.ack)
		}

		assert.Containsf(t, filtered, `<option value="`+tc.ack+`" selected>`, "ack=%s", tc.ack)
		assert.Containsf(t, filtered, "Filtres · 1", "ack=%s counts as a filter", tc.ack)
	}

	// The name of a case reads whatever its letters, and is kept as the model
	// writes it.
	upper := s.get(t, permRead, "/webui/transfers?full=1&ack=Missed").Body.String()
	assert.Equal(t, 1, rows(upper))
	assert.Contains(t, upper, `<option value="missed" selected>`)

	// An unknown case is dropped: every transfer, no filter counted.
	loose := s.get(t, permRead, "/webui/transfers?full=1&ack=later").Body.String()
	assert.Equal(t, 6, rows(loose))
	assert.NotContains(t, loose, "Filtres · 1")

	// In English.
	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: permRead}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, "en")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/webui/transfers?full=1&ack=missed", nil)
	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	english := rec.Body.String()
	assert.Contains(t, english, `<th scope="col">ACK</th>`)
	assert.Contains(t, english, "<span>Not received</span>")
	assert.Contains(t, english, `title="No acknowledgement from the partner within 10 min, waited since `)
	assert.Contains(t, english, `<label class="w-field__label" for="f-ack">Acknowledgement</label>`)
}

// The keys of a page are read in one query, by transfer; a transfer without
// any has no entry.
func TestAckInfos(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)
	s.acknowledgements(t)

	page, err := internal.StartTransferQuery(s.db, "id", true).Run()
	require.NoError(t, err)
	require.Len(t, page, 6)

	h := New(s.db, testhelpers.GetTestLogger(t))
	infos, err := h.ackInfos(page)
	require.NoError(t, err)

	assert.Len(t, infos, 5)
	assert.Equal(t, true, infos[s.running.ID][model.InfoAckExpected])
	assert.Equal(t, "5m", infos[s.running.ID][model.InfoAckWaitTimeout])
	assert.Equal(t, true, infos[s.done.ID][model.InfoAckReceived])
	assert.Equal(t, "prod", infos[s.cancelled.ID][model.InfoAckSentAs])
	assert.NotContains(t, infos[s.failed.ID], model.FollowID, "the other keys of the transfer are not read")
	assert.Nil(t, infos[s.available.ID])

	none, err := h.ackInfos(nil)
	require.NoError(t, err)
	assert.Empty(t, none)
}
