package rules

import (
	"bytes"
	"context"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermRulesRead
	permWrite = model.PermRulesRead | model.PermRulesWrite
	permAll   = model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete
)

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router

	partner *model.RemoteAgent
	account *model.RemoteAccount
	client  *model.Client
	server  *model.LocalAgent
	local   *model.LocalAccount

	send, receive, orders *model.Rule
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/rules").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/rules/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/rules/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/rules/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/rules/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/rules/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/rules/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/rules/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)
	r.Path("/tasks").Methods(http.MethodGet).HandlerFunc(FormerTasksPage)
	r.Path("/rules/{id:[0-9]+}/tasks").Methods(http.MethodGet).HandlerFunc(h.TasksTab)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/new").Methods(http.MethodGet).HandlerFunc(h.TaskNewForm)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/new").Methods(http.MethodPost).HandlerFunc(h.TaskCreate)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/{rank:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.TaskEditForm)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/{rank:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.TaskUpdate)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/{rank:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.TaskDeleteForm)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/{rank:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.TaskDelete)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/{rank:[0-9]+}/up").Methods(http.MethodPost).HandlerFunc(h.TaskUp)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/order").Methods(http.MethodPost).HandlerFunc(h.TaskOrder)
	r.Path("/rules/{id:[0-9]+}/tasks/{chain}/{rank:[0-9]+}/down").Methods(http.MethodPost).HandlerFunc(h.TaskDown)
	r.Path("/rules/{id:[0-9]+}/access").Methods(http.MethodGet).HandlerFunc(h.AccessTab)
	r.Path("/rules/{id:[0-9]+}/access/new").Methods(http.MethodGet).HandlerFunc(h.AccessNewForm)
	r.Path("/rules/{id:[0-9]+}/access/new").Methods(http.MethodPost).HandlerFunc(h.AccessCreate)
	r.Path("/rules/{id:[0-9]+}/access/open").Methods(http.MethodGet).HandlerFunc(h.AccessOpenForm)
	r.Path("/rules/{id:[0-9]+}/access/open").Methods(http.MethodPost).HandlerFunc(h.AccessOpen)
	r.Path("/rules/{id:[0-9]+}/access/{kind}/{target:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.AccessDeleteForm)
	r.Path("/rules/{id:[0-9]+}/access/{kind}/{target:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.AccessDelete)

	return s
}

// fixtures inserts a partner with its account and a client, a server with
// its account, and three rules: "releves" in both directions, the sending one with a task before
// the transfer, one on error and an access for the partner; "commandes",
// receiving, under the path "depot/commandes", with a comment.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.partner = &model.RemoteAgent{Name: "banque-demo", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)}
	require.NoError(t, s.db.Insert(s.partner).Run())

	s.account = &model.RemoteAccount{RemoteAgentID: s.partner.ID, Login: "prod"}
	require.NoError(t, s.db.Insert(s.account).Run())

	s.client = &model.Client{Name: "cli-sftp", Protocol: "sftp"}
	require.NoError(t, s.db.Insert(s.client).Run())

	s.server = &model.LocalAgent{Name: "srv-sftp", Protocol: "sftp", Address: types.Addr("0.0.0.0", 2222)}
	require.NoError(t, s.db.Insert(s.server).Run())

	s.local = &model.LocalAccount{LocalAgentID: s.server.ID, Login: "filiale-sud"}
	require.NoError(t, s.db.Insert(s.local).Run())

	s.send = &model.Rule{Name: "releves", IsSend: true, Path: "releves-envoi"}
	s.receive = &model.Rule{Name: "releves", IsSend: false, Path: "releves-reception"}
	s.orders = &model.Rule{Name: "commandes", IsSend: false, Path: "depot/commandes", Comment: "Commandes des magasins"}

	for _, r := range []*model.Rule{s.send, s.receive, s.orders} {
		require.NoError(t, s.db.Insert(r).Run())
	}

	for _, task := range []*model.Task{
		{RuleID: s.send.ID, Chain: model.ChainPre, Rank: 0, Type: tasks.Delete},
		{RuleID: s.send.ID, Chain: model.ChainError, Rank: 0, Type: tasks.Delete},
	} {
		require.NoError(t, s.db.Insert(task).Run())
	}

	require.NoError(t, s.db.Insert(&model.RuleAccess{RuleID: s.send.ID, RemoteAgentID: s.partner.NullableID()}).Run())
}

func (s *suite) request(t *testing.T, perms model.PermsMask, method, target string, body *bytes.Buffer,
) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")

	var req *http.Request
	if body != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, body)
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

	return s.request(t, perms, http.MethodPost, target, bytes.NewBufferString(form.Encode()))
}

func path(rule *model.Rule, suffix string) string {
	return "/webui/rules/" + strconv.FormatInt(rule.ID, 10) + suffix
}

// read reads a rule again from the database.
func (s *suite) read(t *testing.T, rule *model.Rule) *model.Rule {
	t.Helper()

	var got model.Rule
	require.NoError(t, s.db.Get(&got, "id=?", rule.ID).Run())

	return &got
}

func (s *suite) count(t *testing.T, bean database.Table, where string, args ...any) uint64 {
	t.Helper()

	n, err := s.db.Count(bean).Where(where, args...).Run()
	require.NoError(t, err)

	return n
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given no rule", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.get(t, permWrite, "/webui/rules?reset=1")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Aucune règle de transfert pour l&#39;instant")
		assert.Contains(t, rec.Body.String(), `href="/webui/rules/new`, "the empty state offers the creation")
	})

	t.Run("Given rules", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permRead, "/webui/rules?reset=1")
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Equal(t, 3, strings.Count(body, `class="w-cell-title w-mono w-nowrap"`), "one row per rule and direction")
		assert.Contains(t, body, `href="/webui/rules/`+strconv.FormatInt(s.send.ID, 10))
		assert.Contains(t, body, ">1 · 0 · 1<", "the tasks of each chain: before, after, on error")
		assert.Contains(t, body, "Réservée : 1", "the sending rule is kept for the partner")
		assert.Contains(t, body, "Ouverte à tous")
		assert.Contains(t, body, "Commandes des magasins")
		assert.Contains(t, body, "depot/commandes")
		assert.NotContains(t, body, "Ajouter une règle", "no creation without the write right")
		assert.NotContains(t, body, "Supprimer la règle", "no deletion without the delete right")

		body = s.get(t, permAll, "/webui/rules?reset=1").Body.String()
		assert.Contains(t, body, "Ajouter une règle")
		assert.Contains(t, body, "Supprimer la règle")
		assert.Contains(t, body, "/webui/rules/"+strconv.FormatInt(s.send.ID, 10)+"/tasks", "the menu leads to the Tasks tab")
		assert.Contains(t, body, "/access", "and to the Access tab")
	})

	t.Run("Given a search on the path and a filter on the direction", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/rules?q=DEPOT").Body.String()
		assert.Equal(t, 1, strings.Count(body, `class="w-cell-title w-mono w-nowrap"`), "a part of the path, regardless of case")
		assert.Contains(t, body, ">commandes<")

		body = s.get(t, permRead, "/webui/rules?reset=1&direction=send").Body.String()
		assert.Equal(t, 1, strings.Count(body, `class="w-cell-title w-mono w-nowrap"`), "the sending rule alone")
		assert.Contains(t, body, "Réservée : 1")

		body = s.get(t, permRead, "/webui/rules?q=absente").Body.String()
		assert.Contains(t, body, "Aucune règle ne correspond à « absente »")
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given the sheet of a rule", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permRead, path(s.send, ""))
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		id := strconv.FormatInt(s.send.ID, 10)
		assert.Contains(t, body, "Règle d&#39;envoi « releves »", "the title says the direction")
		assert.Contains(t, body, `href="/webui/rules/`+id+`" aria-current="page">Détails</a>`)
		assert.Contains(t, body, `href="/webui/rules/`+id+`/tasks">Tâches <span class="w-tabs__count">2</span>`)
		assert.Contains(t, body, `href="/webui/rules/`+id+`/access">Accès <span class="w-tabs__count">1</span>`)
		assert.Contains(t, body, "releves-envoi")
		assert.Contains(t, body, "Non défini : celui du serveur, sinon celui de la passerelle.", "an empty directory says what is used")
		assert.NotContains(t, body, "Modifier la règle", "no edition without the write right")
	})

	t.Run("Given a rule that does not exist", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permRead, "/webui/rules/9999")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Règle introuvable")
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given a valid form", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, "/webui/rules/new", url.Values{
			fieldName: {"factures"}, fieldDirection: {dirReceive}, fieldComment: {"Factures fournisseurs"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		var rule model.Rule
		require.NoError(t, s.db.Get(&rule, "name=? AND is_send=?", "factures", false).Run())
		assert.Equal(t, "factures", rule.Path, "an empty path is the name of the rule")
		assert.Equal(t, "Factures fournisseurs", rule.Comment)
		assert.True(t, strings.HasPrefix(rec.Header().Get("Location"), path(&rule, "")), "the sheet of the new rule")
	})

	t.Run("Given the name of a rule of the other direction", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, "/webui/rules/new", url.Values{
			fieldName: {"commandes"}, fieldDirection: {dirSend}, fieldPath: {"commandes-envoi"},
		})
		assert.Equal(t, http.StatusSeeOther, rec.Code, "a name serves both directions")
	})

	for _, tc := range []struct {
		name, message string
		form          url.Values
	}{
		{"no name", "Ce champ est obligatoire.", url.Values{fieldDirection: {dirSend}}},
		{"a slash in the name", "Le nom ne peut pas contenir « / »", url.Values{fieldName: {"a/b"}, fieldDirection: {dirSend}}},
		{"no direction", "Choisissez un sens.", url.Values{fieldName: {"sans-sens"}}},
		{
			"the name of a rule of the same direction", "Une autre règle de réception porte déjà ce nom.",
			url.Values{fieldName: {"releves"}, fieldDirection: {dirReceive}, fieldPath: {"autre"}},
		},
		{
			"a path under the one of another rule", "Ce chemin se trouve sous celui d&#39;une autre règle de réception",
			url.Values{fieldName: {"sous"}, fieldDirection: {dirReceive}, fieldPath: {"depot/commandes/sous"}},
		},
		{
			"a name too long", "100 caractères au plus.",
			url.Values{fieldName: {strings.Repeat("r", 101)}, fieldDirection: {dirSend}},
		},
	} {
		t.Run("Given "+tc.name, func(t *testing.T) {
			t.Parallel()

			// The other cases run at the same time: count the rules of this
			// name only.
			name := tc.form.Get(fieldName)
			before := s.count(t, &model.Rule{}, "name=?", name)
			rec := s.post(t, permWrite, "/webui/rules/new", tc.form)
			assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.message)
			assert.Contains(t, rec.Body.String(), "La règle n&#39;a pas été enregistrée.")
			assert.Equal(t, before, s.count(t, &model.Rule{}, "name=?", name), "nothing saved")
		})
	}
}

func TestEdit(t *testing.T) {
	t.Parallel()

	t.Run("Given an edition that also posts the other direction", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permWrite, path(s.send, "/edit"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Le sens ne change pas", "the direction is in reading")
		assert.NotContains(t, rec.Body.String(), `type="radio"`)

		rec = s.post(t, permWrite, path(s.send, "/edit"), url.Values{
			fieldName: {"releves"}, fieldDirection: {dirReceive}, fieldPath: {"releves-envoi"},
			fieldComment: {"Relevés quotidiens"}, fieldRemote: {"depot"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		got := s.read(t, s.send)
		assert.True(t, got.IsSend, "the direction is kept")
		assert.Equal(t, "Relevés quotidiens", got.Comment)
		assert.Equal(t, "depot", got.RemoteDir)
		assert.Equal(t, uint64(2), s.count(t, &model.Task{}, "rule_id=?", s.send.ID), "the tasks are kept")
		assert.Equal(t, uint64(1), s.count(t, &model.RuleAccess{}, "rule_id=?", s.send.ID), "and the access")
	})

	t.Run("Given a rule the history names", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		require.NoError(t, s.db.Insert(&model.HistoryEntry{
			Identifier: model.ID(1001), RemoteTransferID: "r-1", IsSend: true, Rule: "releves", Account: "prod", Agent: "banque-demo",
			Client: "cli-sftp", Protocol: "sftp", SrcFilename: "a.txt", DestFilename: "a.txt", Status: types.StatusDone,
			Start: time.Now().Add(-time.Hour), Stop: time.Now(),
		}).Run())

		assert.NotContains(t, s.get(t, permWrite, path(s.send, "/edit")).Body.String(),
			"L&#39;historique désigne cette règle", "nothing is said before the validation")

		values := url.Values{fieldName: {"releves-v2"}, fieldPath: {"releves-envoi"}}
		rec := s.post(t, permWrite, path(s.send, "/edit"), values)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "a rename is asked once more")
		body := rec.Body.String()
		assert.Contains(t, body, "L&#39;historique désigne cette règle par son nom.")
		assert.Contains(t, body, "Validez de nouveau pour renommer la règle.")
		assert.Contains(t, body, `<input type="hidden" name="confirm_name" value="releves-v2">`)
		assert.Equal(t, "releves", s.read(t, s.send).Name, "nothing is saved yet")

		values.Set(fieldConfirmName, "releves-v2")
		values.Set(fieldName, "releves-v3")
		rec = s.post(t, permWrite, path(s.send, "/edit"), values)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "a name changed after the warning is asked again")
		assert.Contains(t, rec.Body.String(), `<input type="hidden" name="confirm_name" value="releves-v3">`)

		values.Set(fieldConfirmName, "releves-v3")
		require.Equal(t, http.StatusSeeOther, s.post(t, permWrite, path(s.send, "/edit"), values).Code,
			"validated again, the rule is renamed")
		assert.Equal(t, "releves-v3", s.read(t, s.send).Name)

		require.Equal(t, http.StatusSeeOther, s.post(t, permWrite, path(s.receive, "/edit"), url.Values{
			fieldName: {"releves-v2"}, fieldPath: {"releves-reception"},
		}).Code, "the rule of the other direction, which the history does not name, is renamed at once")
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("Given a rule nothing names", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, path(s.send, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ses tâches (2) et ses accès (1) seront supprimés avec elle.")
		assert.Contains(t, rec.Body.String(), ">Supprimer</button>")

		rec = s.post(t, permAll, path(s.send, "/delete"), nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=deleted")
		assert.Zero(t, s.count(t, &model.Rule{}, "id=?", s.send.ID))
		assert.Zero(t, s.count(t, &model.Task{}, "rule_id=?", s.send.ID), "its tasks go with it")
		assert.Zero(t, s.count(t, &model.RuleAccess{}, "rule_id=?", s.send.ID), "and its access")
	})

	t.Run("Given a rule a transfer still names", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		require.NoError(t, s.db.Insert(&model.Transfer{
			RuleID: s.send.ID, ClientID: s.client.NullableID(), RemoteAccountID: s.account.NullableID(),
			SrcFilename: "a.txt", DestFilename: "a.txt", Status: types.StatusError, Filesize: model.UnknownSize,
		}).Run())

		rec := s.get(t, permAll|model.PermTransfersRead, path(s.send, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "Cette règle ne peut pas être supprimée.")
		assert.Contains(t, body, "Transferts qui la désignent encore : 1")
		assert.Contains(t, body, `href="/webui/transfers?direction=send&amp;rule=releves"`, "the transfers to see")
		assert.NotContains(t, body, ">Supprimer</button>")

		rec = s.post(t, permAll, path(s.send, "/delete"), nil)
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, uint64(1), s.count(t, &model.Rule{}, "id=?", s.send.ID), "the rule stays")
	})

	t.Run("Given a rule a file watcher names", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		require.NoError(t, s.db.Insert(&model.FileWatcher{
			Flow: "surveillance-commandes", Pattern: "*.edi", RemoteAccount: *s.account, Client: *s.client, Rule: *s.orders,
		}).Run())

		body := s.get(t, permAll, path(s.orders, "/delete")).Body.String()
		assert.Contains(t, body, "Cette règle ne peut pas être supprimée.")
		assert.Contains(t, body, "Dossier surveillé « surveillance-commandes »")

		rec := s.post(t, permAll, path(s.orders, "/delete"), nil)
		assert.Equal(t, http.StatusConflict, rec.Code, "explained, not the error of the database")
		assert.Equal(t, uint64(1), s.count(t, &model.Rule{}, "id=?", s.orders.ID))
	})
}
