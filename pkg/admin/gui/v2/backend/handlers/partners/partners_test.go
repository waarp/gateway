package partners

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermPartnersRead
	permWrite = model.PermPartnersRead | model.PermPartnersWrite
	permAll   = model.PermPartnersRead | model.PermPartnersWrite | model.PermPartnersDelete
	// permClients adds the right to add a client, shown on the sheet of a
	// partner without any client of its protocol.
	permClients = permAll | model.PermServersWrite

	anyRet = "?ret=q%3Dpart"
)

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router

	sftp, r66, pesit *model.RemoteAgent
	prod, test       *model.RemoteAccount // the two accounts of the SFTP partner
	cliSFTP          *model.Client
	send, receive    *model.Rule
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/partners").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/partners/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/partners/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/partners/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/partners/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/partners/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/partners/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/partners/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	const accountPath = "/partners/{id:[0-9]+}/accounts/{aid:[0-9]+}"

	for _, base := range []string{"/partners/{id:[0-9]+}/auth", accountPath + "/auth"} {
		r.Path(base).Methods(http.MethodGet).HandlerFunc(h.Auth)
		r.Path(base + "/new").Methods(http.MethodGet).HandlerFunc(h.NewCredForm)
		r.Path(base + "/new").Methods(http.MethodPost).HandlerFunc(h.CreateCred)
		r.Path(base + "/{cid:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.CredDetail)
		r.Path(base + "/{cid:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditCredForm)
		r.Path(base + "/{cid:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.UpdateCred)
		r.Path(base + "/{cid:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteCredForm)
		r.Path(base + "/{cid:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.DeleteCred)
	}

	r.Path("/partners/{id:[0-9]+}/accounts").Methods(http.MethodGet).HandlerFunc(h.Accounts)
	r.Path("/partners/{id:[0-9]+}/accounts/new").Methods(http.MethodGet).HandlerFunc(h.NewAccountForm)
	r.Path("/partners/{id:[0-9]+}/accounts/new").Methods(http.MethodPost).HandlerFunc(h.CreateAccount)
	r.Path(accountPath).Methods(http.MethodGet).HandlerFunc(h.AccountDetail)
	r.Path(accountPath + "/edit").Methods(http.MethodGet).HandlerFunc(h.EditAccountForm)
	r.Path(accountPath + "/edit").Methods(http.MethodPost).HandlerFunc(h.UpdateAccount)
	r.Path(accountPath + "/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteAccountForm)
	r.Path(accountPath + "/delete").Methods(http.MethodPost).HandlerFunc(h.DeleteAccount)

	return s
}

// fixtures inserts three partners (an SFTP one with two accounts and a
// deprecated option kept as it is, an R66 over TLS one, a PeSIT one with
// three-state options), the clients of their protocols (one SFTP client,
// two R66 over TLS clients, no PeSIT client) and two rules, one of them
// restricted to the SFTP partner.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.sftp = &model.RemoteAgent{
		Name: "part-sftp-01", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022),
		ProtoConfig: map[string]any{"useStat": true},
	}
	s.r66 = &model.RemoteAgent{
		Name: "part-r66-tls", Protocol: "r66-tls", Address: types.Addr("127.0.0.2", 6667),
		ProtoConfig: map[string]any{"serverLogin": "waarp", "minTLSVersion": "v1.2", "serverPassword": "legacy-secret"},
	}
	s.pesit = &model.RemoteAgent{
		Name: "part-pesit-01", Protocol: "pesit", Address: types.Addr("10.20.30.40", 5000),
		ProtoConfig: map[string]any{"login": "gw", "disableRestart": true, "expectsAck": true, "ackTimeout": "10s"},
	}

	for _, agent := range []*model.RemoteAgent{s.sftp, s.r66, s.pesit} {
		require.NoError(t, s.db.Insert(agent).Run())
	}

	s.prod = &model.RemoteAccount{RemoteAgentID: s.sftp.ID, Login: "prod"}
	s.test = &model.RemoteAccount{RemoteAgentID: s.sftp.ID, Login: "test"}
	require.NoError(t, s.db.Insert(s.prod).Run())
	require.NoError(t, s.db.Insert(s.test).Run())
	require.NoError(t, s.db.Insert(&model.Credential{
		RemoteAccountID: s.prod.NullableID(), Name: "mot-de-passe", Type: auth.Password, Value: "s3cret",
	}).Run())

	s.cliSFTP = &model.Client{Name: "cli-sftp", Protocol: "sftp"}
	require.NoError(t, s.db.Insert(s.cliSFTP).Run())
	require.NoError(t, s.db.Insert(&model.Client{Name: "cli-r66-a", Protocol: "r66-tls"}).Run())
	require.NoError(t, s.db.Insert(&model.Client{Name: "cli-r66-b", Protocol: "r66-tls"}).Run())

	s.send = &model.Rule{Name: "envoi", IsSend: true, Path: "/envoi"}
	s.receive = &model.Rule{Name: "reception", IsSend: false, Path: "/reception"}
	require.NoError(t, s.db.Insert(s.send).Run())
	require.NoError(t, s.db.Insert(s.receive).Run())
	require.NoError(t, s.db.Insert(&model.RuleAccess{RuleID: s.receive.ID, RemoteAgentID: s.sftp.NullableID()}).Run())
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

func (s *suite) path(agent *model.RemoteAgent, suffix string) string {
	return partnerURL(agent.ID, suffix)
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

func (s *suite) partnerOf(t *testing.T, name string) *model.RemoteAgent {
	t.Helper()

	var agent model.RemoteAgent
	require.NoError(t, s.db.Get(&agent, "name=?", name).Run())

	return &agent
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.get(t, permWrite, "/webui/partners")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Aucun partenaire distant pour l&#39;instant")
		assert.Contains(t, rec.Body.String(), `href="/webui/partners/new">`, "the empty state leads to the creation")
	})

	t.Run("rows, rights, accounts", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permRead, "/webui/partners")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Equal(t, 3, rows(body))
		indexes(t, body, "part-pesit-01", "part-r66-tls", "part-sftp-01")
		assert.Contains(t, body, "3 partenaires")
		assert.NotContains(t, body, "Ajouter un partenaire", "no coloured action without the write right")
		assert.NotContains(t, body, "Modifier le partenaire")
		assert.NotContains(t, body, "Supprimer le partenaire")
		assert.Contains(t, body, `<td class="w-num">2</td>`, "the accounts of the SFTP partner")
		assert.Contains(t, body, "Aucun</span>", "a partner without any account carries the warning")
		assert.Contains(t, body, `<span class="w-badge w-badge--neutral">TLS</span>`)
		assert.Contains(t, body, `<td class="w-mono w-nowrap">127.0.0.1:10022</td>`)

		body = s.get(t, permAll, "/webui/partners").Body.String()
		assert.Contains(t, body, "Ajouter un partenaire")
		assert.Contains(t, body, "Modifier le partenaire")
		assert.Contains(t, body, "Supprimer le partenaire")
	})

	t.Run("sort, filter, search", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/partners?sort=accounts&dir=desc").Body.String()
		indexes(t, body, "<tbody>", "part-sftp-01", "part-pesit-01")

		body = s.get(t, permRead, "/webui/partners?protocol=pesit").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, "3 partenaires · 1 affiché")

		body = s.get(t, permRead, "/webui/partners?tls=yes").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, "part-r66-tls")

		body = s.get(t, permRead, "/webui/partners?q=part-s").Body.String()
		assert.Equal(t, 1, rows(body))

		body = s.get(t, permRead, "/webui/partners?q=zzz").Body.String()
		assert.Contains(t, body, "Aucun partenaire ne correspond à « zzz »")
		assert.Contains(t, body, "Effacer la recherche")
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("one client of the protocol, every rule", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, s.path(s.sftp, "")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Partenaire « part-sftp-01 » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="/webui/partners?q=part">Partenaires distants</a>`, "the breadcrumb keeps the list state")
		assert.Contains(t, body, "<dt>Protocole</dt><dd>SFTP</dd>")
		assert.Contains(t, body, `<dt>Adresse</dt><dd><code class="w-code">127.0.0.1:10022</code></dd>`)
		assert.Contains(t, body, "<dt>Comptes distants</dt><dd>2</dd>")
		assert.Contains(t, body, "Les transferts vers ce partenaire passent par le client « cli-sftp »")
		assert.Contains(t, body, `href="/webui/clients/`+strconv.FormatInt(s.cliSFTP.ID, 10)+`">cli-sftp</a>`)
		assert.Contains(t, body, "Toutes les règles de transfert de la passerelle.", "the restricted rule names it: every rule")
		assert.Contains(t, body, "<dt>Lire la taille du fichier avant de le recevoir</dt><dd>Activé</dd>")
		assert.Contains(t, body, "<dt>Lecture parallèle</dt><dd>Activé</dd>", "an absent flag reads as the product default")
		assert.Contains(t, body, `<span class="w-tabs__count">2</span>`, "the accounts tab counts them")
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/edit")+anyRet+`">`)
		assert.NotContains(t, body, "Supprimer le partenaire", "no deletion without the right")
	})

	t.Run("several clients, restricted rules", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.r66, "")).Body.String()
		assert.Contains(t, body, "2 clients R66 existent : chaque transfert vers ce partenaire doit nommer le sien.")
		assert.Contains(t, body, ">cli-r66-a</a>")
		assert.Contains(t, body, ">cli-r66-b</a>")
		assert.Contains(t, body, "<li>envoi (envoi)</li>", "the open rule only: the other one names the SFTP partner")
		assert.NotContains(t, body, "<li>reception")
		assert.Contains(t, body, "<dt>Identifiant présenté par le serveur</dt><dd><code class=\"w-code\">waarp</code></dd>")
		assert.Contains(t, body, "<dt>Version TLS minimale</dt><dd>TLS 1.2</dd>")
		assert.NotContains(t, body, "legacy-secret", "the deprecated password never shows")
		assert.Contains(t, body, "Supprimer le partenaire")
	})

	t.Run("no client yet, three-state options", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.pesit, "")).Body.String()
		assert.Contains(t, body, "Aucun client PeSIT : la passerelle en créera un, nommé « pesit », au premier transfert.")
		assert.NotContains(t, body, "Ajouter un client", "no client to add without the right on the servers")
		assert.Contains(t, body, "<dt>Identifiant du partenaire</dt><dd><code class=\"w-code\">gw</code></dd>")
		assert.Contains(t, body, "<dt>Reprise des transferts interrompus</dt><dd>Désactivé</dd>")
		assert.Contains(t, body, "<dt>Points de synchronisation</dt><dd>Comme le client (par défaut)</dd>")
		assert.Contains(t, body, "<dt>Attend un acquittement des transferts reçus</dt><dd>Activé</dd>")
		assert.Contains(t, body, "<dt>Délai d&#39;attente de l&#39;acquittement</dt><dd>10 s</dd>", "a duration in words")
		assert.Contains(t, body, "<dt>Comptes distants</dt><dd>Aucun</dd>")

		body = s.get(t, permClients, s.path(s.pesit, "")).Body.String()
		assert.Contains(t, body, `href="/webui/clients/new">`)
		assert.Contains(t, body, "Ajouter un client")
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permRead, "/webui/partners/999"+anyRet)
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Partenaire introuvable")
		assert.Contains(t, rec.Body.String(), `href="/webui/partners?q=part"`, "the way back keeps the list state")
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("form", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, "/webui/partners/new"+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Nouveau partenaire")
		assert.Contains(t, body, `name="protocol"`)
		assert.Contains(t, body, `name="pesit_restart"`, "the three-state select of a PeSIT partner")
		assert.Contains(t, body, "Comme le client (par défaut)")
		assert.Contains(t, body, `name="pesit_nsdu" value="1" checked`, "NSDU is on by default")
		assert.Contains(t, body, `name="as2_sign_algo"`)
		assert.Contains(t, body, `name="ftp_implicit_tls"`)
		assert.NotContains(t, body, `name="r66_login"`, "a server-only option")
		assert.NotContains(t, body, `name="ftp_enable_active"`, "a client-only option")
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, "/webui/partners/new"+anyRet, url.Values{
			"name": {"part-pesit-tls"}, "protocol": {"pesit"}, "tls": {"1"}, "host": {"10.20.30.41"}, "port": {"5443"},
			"pesit_login": {"GW01"}, "pesit_restart": {"on"}, "pesit_checkpoints": {""}, "pesit_nsdu": {"1"},
			"pesit_max_message": {"32000"}, "pesit_ack_timeout_m": {"1"}, "pesit_expects_ack": {"1"}, "min_tls": {"v1.3"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/partners?done=created&name=part-pesit-tls&q=part", rec.Header().Get("Location"))

		agent := s.partnerOf(t, "part-pesit-tls")
		assert.Equal(t, "pesit-tls", agent.Protocol)
		assert.Equal(t, "10.20.30.41:5443", agent.Address.String())
		assert.Equal(t, "GW01", agent.ProtoConfig["login"])
		assert.Equal(t, false, agent.ProtoConfig["disableRestart"], "restart on is stored as the product reads it")
		assert.Nil(t, agent.ProtoConfig["disableCheckpoints"], "the client's value: nothing stored (the model writes null)")
		assert.Equal(t, true, agent.ProtoConfig["useNSDU"])
		assert.Equal(t, "1m", agent.ProtoConfig["ackTimeout"])
		assert.Equal(t, true, agent.ProtoConfig["expectsAck"])
		assert.Equal(t, "v1.3", agent.ProtoConfig["minTLSVersion"])
	})

	t.Run("refused", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, "/webui/partners/new", url.Values{
			"name": {"part-sftp-01"}, "protocol": {"pesit"}, "host": {""}, "port": {"0"}, "pesit_ack_timeout_s": {"soon"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Le partenaire n&#39;a pas été enregistré.")
		assert.Contains(t, body, "Indiquez l&#39;hôte.")
		assert.Contains(t, body, "Indiquez un port de 1 à 65535.")
		assert.Contains(t, body, "Indiquez des nombres entiers d&#39;heures, de minutes et de secondes.")
		assert.Contains(t, body, `value="soon"`, "the posted values are kept")

		rec = s.post(t, permWrite, "/webui/partners/new", url.Values{
			"name": {"part-sftp-01"}, "protocol": {"sftp"}, "host": {"127.0.0.9"}, "port": {"22"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre partenaire porte déjà ce nom.")

		rec = s.post(t, permWrite, "/webui/partners/new", url.Values{
			"name": {"part-as2"}, "protocol": {"as2"}, "host": {"127.0.0.9"}, "port": {"80"}, "as2_handle_async_mdn": {"1"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez l&#39;adresse des accusés MDN asynchrones pour les traiter.")
	})
}

func TestEdit(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permWrite, s.path(s.r66, "/edit")+anyRet)
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "Modifier « part-r66-tls »")
	assert.Contains(t, body, `value="127.0.0.2"`)
	assert.Contains(t, body, `value="waarp"`)
	assert.Contains(t, body, "Changer de protocole rend caducs")
	assert.NotContains(t, body, "legacy-secret")

	rec = s.post(t, permWrite, s.path(s.r66, "/edit")+anyRet, url.Values{
		"name": {"part-r66-tls-2"}, "protocol": {"r66"}, "tls": {"1"}, "host": {"127.0.0.2"}, "port": {"6668"},
		"r66_server_login": {"waarp2"}, "r66_block": {"32768"}, "r66_final_hash_algo": {"SHA-512"},
	})
	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
	assert.Equal(t, s.path(s.r66, "")+"?done=updated&name=part-r66-tls-2&ret=q%3Dpart", rec.Header().Get("Location"),
		"an edited partner comes back to its sheet")

	agent := s.partnerOf(t, "part-r66-tls-2")
	assert.Equal(t, "r66-tls", agent.Protocol)
	assert.Equal(t, uint16(6668), agent.Address.Port)
	assert.Equal(t, "waarp2", agent.ProtoConfig["serverLogin"])
	assert.Equal(t, "SHA-512", agent.ProtoConfig["finalHashAlgo"])
	assert.NotEmpty(t, agent.ProtoConfig["serverPassword"], "the deprecated password is kept (the model hashes it on every write)")
	assert.Equal(t, "v1.2", agent.ProtoConfig["minTLSVersion"], "the TLS version of the form, its default")
}

func TestDelete(t *testing.T) {
	t.Parallel()

	t.Run("confirmed", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, s.path(s.sftp, "/delete")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Supprimer le partenaire « part-sftp-01 » ?")
		assert.Contains(t, body, "Ses 2 comptes distants et leurs moyens d&#39;authentification seront supprimés avec lui")
		assert.Contains(t, body, `<form class="sheet-form-actions" method="post" action="`+s.path(s.sftp, "/delete")+anyRet+`">`)

		rec = s.post(t, permAll, s.path(s.sftp, "/delete")+anyRet, nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/partners?done=deleted&name=part-sftp-01&q=part", rec.Header().Get("Location"))

		n, err := s.db.Count(&model.RemoteAgent{}).Where("id=?", s.sftp.ID).Run()
		require.NoError(t, err)
		assert.Zero(t, n)
		n, err = s.db.Count(&model.RemoteAccount{}).Where("remote_agent_id=?", s.sftp.ID).Run()
		require.NoError(t, err)
		assert.Zero(t, n, "the accounts go with the partner")
		n, err = s.db.Count(&model.Credential{}).Where("remote_account_id=?", s.prod.ID).Run()
		require.NoError(t, err)
		assert.Zero(t, n, "the means of the accounts go with it")
	})

	t.Run("refused: running transfer", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		require.NoError(t, s.db.Insert(&model.Transfer{
			RuleID: s.send.ID, ClientID: sql.NullInt64{Int64: s.cliSFTP.ID, Valid: true},
			RemoteAccountID: s.prod.NullableID(), SrcFilename: "file",
		}).Run())

		rec := s.get(t, permAll, s.path(s.sftp, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce partenaire ne peut pas être supprimé.")
		assert.Contains(t, rec.Body.String(), "Des transferts en cours utilisent un de ses comptes.")
		assert.Contains(t, rec.Body.String(), `href="/webui/transfer_monitoring"`)
		assert.NotContains(t, rec.Body.String(), `<form class="sheet-form-actions"`, "no deletion offered")

		rec = s.post(t, permAll, s.path(s.sftp, "/delete"), nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		n, err := s.db.Count(&model.RemoteAgent{}).Where("id=?", s.sftp.ID).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 1, n, "the partner stays")
	})

	t.Run("refused: folder watcher", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		require.NoError(t, s.db.Insert(&model.FileWatcher{
			Flow: "flux", RemoteAccount: *s.prod, Client: *s.cliSFTP, Rule: *s.receive,
		}).Run())

		rec := s.get(t, permAll, s.path(s.sftp, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Une surveillance de dossiers utilise un de ses comptes.")
		assert.NotContains(t, rec.Body.String(), `<form class="sheet-form-actions"`)

		rec = s.post(t, permAll, s.path(s.sftp, "/delete"), nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})
}

// The "Authentication" tab of a partner: what the gateway checks of it,
// with the kinds its protocol admits (the password of a PeSIT partner
// included), served by the shared pages of the means.
func TestAuthTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permWrite, s.path(s.pesit, "/auth")+anyRet)
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "Ce que la passerelle vérifie du partenaire")
	assert.Contains(t, body, "Ajoutez ce que la passerelle doit vérifier du partenaire")
	assert.Contains(t, body, `href="`+s.path(s.pesit, "/auth/new")+anyRet+`"`, "a password can be added")
	assert.Contains(t, body, `aria-label="Onglets de la fiche"`)

	rec = s.get(t, permWrite, s.path(s.pesit, "/auth/new"))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Moyen d&#39;authentification pour le partenaire « part-pesit-01 »")
	assert.Contains(t, rec.Body.String(), `<option value="password"`)
	assert.NotContains(t, rec.Body.String(), `<option value="trusted_tls_certificate"`, "not without TLS")

	rec = s.post(t, permWrite, s.path(s.pesit, "/auth/new")+anyRet, url.Values{
		"name": {"mdp-serveur"}, "type": {"password"}, "password": {"s3cret"},
	})
	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
	assert.Equal(t, s.path(s.pesit, "/auth")+"?done=created&kind=cred&name=mdp-serveur&ret=q%3Dpart", rec.Header().Get("Location"))

	var cred model.Credential
	require.NoError(t, s.db.Get(&cred, "name=?", "mdp-serveur").Run())
	assert.Equal(t, s.pesit.ID, cred.RemoteAgentID.Int64, "the means belongs to the partner")

	body = s.get(t, permWrite, s.path(s.pesit, "/auth")).Body.String()
	assert.Contains(t, body, "mdp-serveur")
	assert.NotContains(t, body, "s3cret", "a secret never shows")
	assert.NotContains(t, body, `href="`+s.path(s.pesit, "/auth/new")+`"`, "one password per partner: nothing left to add")

	http := &model.RemoteAgent{Name: "part-http", Protocol: "http", Address: types.Addr("127.0.0.3", 80)}
	require.NoError(t, s.db.Insert(http).Run())
	body = s.get(t, permWrite, s.path(http, "/auth")).Body.String()
	assert.Contains(t, body, "Ce protocole ne demande aucune vérification du partenaire")
}

func TestSheetHelpers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "/webui/partners/1/accounts/2/auth", accountURL(1, 2, authSuffix))
	assert.Equal(t, "/webui/partners/7/edit", partnerURL(7, editSuffix))
}

// The tab strip of the sheet stays on every page of the partner: greyed on
// the form of a new partner, live on edition and on the deletion page
// (Driss, 2026-10-20).
func TestSheetTabsOnForms(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	body := s.get(t, permAll, "/webui/partners/new").Body.String()
	assert.Contains(t, body, `<nav class="w-tabs" aria-label="Onglets de la fiche"><span aria-current="page">Détails</span>`)
	assert.Contains(t, body, `<span class="w-tabs__disabled" aria-disabled="true" title="Disponible après la création">Authentification</span>`)
	assert.Contains(t, body, `<span class="w-tabs__disabled" aria-disabled="true" title="Disponible après la création">Comptes distants</span>`)

	body = s.get(t, permAll, s.path(s.sftp, "/edit")+anyRet).Body.String()
	assert.Contains(t, body, `<a href="`+s.path(s.sftp, "")+`?ret=q%3Dpart" aria-current="page">Détails</a>`)
	assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dpart">Comptes distants <span class="w-tabs__count">2</span></a>`)
	assert.NotContains(t, body, "w-tabs__disabled")

	body = s.get(t, permAll, s.path(s.sftp, "/delete")).Body.String()
	assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/auth")+`">Authentification <span class="w-tabs__count">`)

	rec := s.post(t, permWrite, "/webui/partners/new", url.Values{"name": {""}})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), `<span class="w-tabs__disabled"`, "the refused form keeps the greyed strip")
}
