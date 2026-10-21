package snmp

import (
	"bytes"
	"context"
	"errors"
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

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	gwsnmp "code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermAdminRead
	permWrite = model.PermAdminRead | model.PermAdminWrite
	permAll   = model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete

	// The secrets of the fixtures: never in a page.
	secretCommunity = "community-never-shown"
	secretAuth      = "auth-pass-never-shown"
	secretPriv      = "priv-pass-never-shown"
	secretAgentAuth = "agent-auth-never-shown"

	engineID = "0x80001f8880e9630000d61ff449"
)

// fakeService stands for the SNMP service of the gateway: it counts the
// reloads and the test notifications, and fails when asked to.
type fakeService struct {
	mu        sync.Mutex
	monitors  int
	agents    int
	tests     int
	testErr   error
	reloadErr error
}

func (f *fakeService) State() (utils.StateCode, string) { return utils.StateRunning, "" }

func (f *fakeService) ReloadMonitorsConf() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.monitors++

	return f.reloadErr
}

func (f *fakeService) ReloadServerConf(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.agents++

	return f.reloadErr
}

func (f *fakeService) SendTestNotification() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tests++

	return f.testErr
}

// suite is a database, the handlers on it with a fake SNMP service, and the
// router that names their routes as the web interface does.
type suite struct {
	db      *database.DB
	router  *mux.Router
	h       *Handlers
	svc     *fakeService
	v2, v3  *gwsnmp.MonitorConfig
	agentID int64
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter(), svc: &fakeService{}}
	s.h = New(s.db, testhelpers.GetTestLogger(t))
	s.h.service = func() service { return s.svc }

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/snmp").Methods(http.MethodGet).HandlerFunc(s.h.Page)
	r.Path("/snmp/test").Methods(http.MethodPost).HandlerFunc(s.h.Test)
	r.Path("/snmp/monitors/new").Methods(http.MethodGet).HandlerFunc(s.h.NewMonitorForm)
	r.Path("/snmp/monitors/new").Methods(http.MethodPost).HandlerFunc(s.h.CreateMonitor)
	r.Path("/snmp/monitors/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(s.h.Monitor)
	r.Path("/snmp/monitors/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(s.h.EditMonitorForm)
	r.Path("/snmp/monitors/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(s.h.UpdateMonitor)
	r.Path("/snmp/monitors/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(s.h.DeleteMonitorForm)
	r.Path("/snmp/monitors/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(s.h.DeleteMonitor)
	r.Path("/snmp/agent/edit").Methods(http.MethodGet).HandlerFunc(s.h.AgentForm)
	r.Path("/snmp/agent/edit").Methods(http.MethodPost).HandlerFunc(s.h.SaveAgent)
	r.Path("/snmp/agent/delete").Methods(http.MethodGet).HandlerFunc(s.h.DisableAgentForm)
	r.Path("/snmp/agent/delete").Methods(http.MethodPost).HandlerFunc(s.h.DisableAgent)

	return s
}

// fixtures inserts an SNMPv2c monitor with its community, an SNMPv3 monitor
// with authentication and encryption, and an agent with an SNMPv3 user.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.v2 = &gwsnmp.MonitorConfig{
		Name: "supervision-v2", Version: versionV2, UDPAddress: "127.0.0.1:16162", Community: secretCommunity,
	}
	s.v3 = &gwsnmp.MonitorConfig{
		Name: "alerting-v3", Version: versionV3, UDPAddress: "127.0.0.1:16163",
		SNMPv3Security: gwsnmp.V3SecurityAuthPriv, AuthEngineID: engineID, AuthUsername: "gw-user",
		AuthProtocol: gwsnmp.AuthSHA256, AuthPassphrase: secretAuth,
		PrivProtocol: gwsnmp.PrivAES, PrivPassphrase: secretPriv,
	}

	for _, m := range []*gwsnmp.MonitorConfig{s.v2, s.v3} {
		require.NoError(t, s.db.Insert(m).Run())
	}

	agent := &gwsnmp.ServerConfig{
		LocalUDPAddress: "127.0.0.1:16161", SNMPv3Username: "agent-user",
		SNMPv3AuthProtocol: gwsnmp.AuthSHA, SNMPv3AuthPassphrase: secretAgentAuth,
	}
	require.NoError(t, s.db.Insert(agent).Run())
	s.agentID = agent.ID
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

func (s *suite) post(t *testing.T, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	return s.request(t, permAll, http.MethodPost, target, bytes.NewBufferString(form.Encode()))
}

func (s *suite) monitor(t *testing.T, name string) *gwsnmp.MonitorConfig {
	t.Helper()

	var m gwsnmp.MonitorConfig
	require.NoError(t, s.db.Get(&m, "name=?", name).Run())

	return &m
}

func (s *suite) agentConf(t *testing.T) *gwsnmp.ServerConfig {
	t.Helper()

	conf, err := s.h.agent()
	require.NoError(t, err)

	return conf
}

func path(m *gwsnmp.MonitorConfig, suffix string) string {
	return "/webui/snmp/monitors/" + strconv.FormatInt(m.ID, 10) + suffix
}

// redirected asserts a 303 to the path and returns the query it carries.
func redirected(t *testing.T, rec *httptest.ResponseRecorder, wantPath string) url.Values {
	t.Helper()

	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

	u, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, wantPath, u.Path)

	return u.Query()
}

// noSecret asserts that no secret of the fixtures shows in a page.
func noSecret(t *testing.T, body string) {
	t.Helper()

	for _, secret := range []string{secretCommunity, secretAuth, secretPriv, secretAgentAuth} {
		assert.NotContains(t, body, secret)
	}
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

func v3Form(name string) url.Values {
	return url.Values{
		fieldName: {name}, fieldVersion: {versionV3}, fieldAddress: {"127.0.0.1:162"},
		fieldSecurity: {gwsnmp.V3SecurityAuthPriv}, fieldUser: {"gw"}, fieldEngine: {engineID},
		fieldAuthProtocol: {gwsnmp.AuthSHA512}, fieldAuthPass: {"new-auth-pass"},
		fieldPrivProtocol: {gwsnmp.PrivAES256}, fieldPrivPass: {"new-priv-pass"},
	}
}

func TestPage(t *testing.T) {
	t.Parallel()

	t.Run("Given every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, "/webui/snmp")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		noSecret(t, body)

		indexes(t, body,
			`<h1 class="w-page-header__title">SNMP</h1>`,
			`action="/webui/snmp/test"`, ">Tester l&#39;envoi à tous les moniteurs</button>",
			`href="/webui/snmp/monitors/new"`, "Ajouter un moniteur",
			">Alertes envoyées par la passerelle</h2>", "la passerelle envoie une alerte à chacun de ces outils",
			">alerting-v3</a>", "<td>SNMPv3</td>", "127.0.0.1:16163", "<td>Authentification et chiffrement</td>",
			">supervision-v2</a>", "<td>SNMPv2c</td>", "127.0.0.1:16162", "<td>Communauté</td>",
			">Interrogation par la supervision</h2>", "Modifier l&#39;agent", "Désactiver l&#39;agent",
			"l&#39;agent SNMP de la passerelle répond aux outils de supervision qui l&#39;interrogent",
			"En ligne", "127.0.0.1:16161", "SNMPv2c et SNMPv3", "public, la valeur par défaut",
			"agent-user", "SHA (historique), phrase secrète définie", "<dd>Aucun</dd>",
			"Désactiver l&#39;agent arrête cette écoute ; les alertes continuent de partir.")
		assert.Contains(t, body, "Modifier le moniteur")
		assert.Contains(t, body, "Supprimer le moniteur")
	})

	t.Run("Sorted by address, descending", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, "/webui/snmp?sort=address&dir=desc").Body.String()
		indexes(t, body, "127.0.0.1:16163", "127.0.0.1:16162")
		assert.Contains(t, body, "triés par adresse")
	})

	t.Run("In reading only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/snmp").Body.String()
		assert.NotContains(t, body, "Ajouter un moniteur")
		assert.NotContains(t, body, "Tester l&#39;envoi")
		assert.NotContains(t, body, "Modifier l&#39;agent")
		assert.NotContains(t, body, "Désactiver l&#39;agent arrête", "no note about an action the user cannot take")
		assert.NotContains(t, body, "Désactiver l&#39;agent")
		assert.NotContains(t, body, "Modifier le moniteur")
		assert.NotContains(t, body, "Supprimer le moniteur")
		assert.Contains(t, body, "Voir la fiche")
	})

	t.Run("Without monitor nor agent", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		body := s.get(t, permAll, "/webui/snmp").Body.String()
		assert.Contains(t, body, "Aucun moniteur SNMP pour l&#39;instant")
		assert.Contains(t, body, "La passerelle ne répond pas aux requêtes SNMP.")
		assert.Contains(t, body, "Configurer l&#39;agent")
		assert.NotContains(t, body, "Tester l&#39;envoi", "nothing to test without a monitor")
		assert.NotContains(t, body, "Désactiver l&#39;agent arrête", "no agent to disable")
	})

	t.Run("Without the SNMP service", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.h.service = func() service { return nil }

		body := s.get(t, permAll, "/webui/snmp").Body.String()
		assert.NotContains(t, body, "État du service SNMP")
	})

	t.Run("The notifications of the previous action", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		assert.Contains(t, s.get(t, permAll, "/webui/snmp?done=tested").Body.String(),
			"Notification de test envoyée aux moniteurs")

		body := s.get(t, permAll, "/webui/snmp?failed=test&reason=unable+to+marshal").Body.String()
		assert.Contains(t, body, "La notification de test n&#39;a pas pu être envoyée.")
		assert.Contains(t, body, "unable to marshal")
	})
}

func TestMonitorSheet(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("An SNMPv2c monitor", func(t *testing.T) {
		rec := s.get(t, permAll, path(s.v2, ""))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		noSecret(t, body)
		indexes(t, body, "Moniteur « supervision-v2 »", "Retour à SNMP", "Modifier le moniteur",
			"SNMPv2c", "127.0.0.1:16162", "Communauté", "<dd>Définie</dd>")
		assert.NotContains(t, body, "Niveau de sécurité")
	})

	t.Run("An SNMPv3 monitor", func(t *testing.T) {
		body := s.get(t, permAll, path(s.v3, "")).Body.String()
		noSecret(t, body)
		indexes(t, body, "Niveau de sécurité", "Authentification et chiffrement", "gw-user", engineID,
			"SHA-256, phrase secrète définie", "AES, phrase secrète définie")
	})

	t.Run("In reading only", func(t *testing.T) {
		body := s.get(t, permRead, path(s.v3, "")).Body.String()
		assert.NotContains(t, body, "Modifier le moniteur")
		assert.NotContains(t, body, "Supprimer le moniteur")
	})

	t.Run("A missing monitor", func(t *testing.T) {
		rec := s.get(t, permAll, "/webui/snmp/monitors/999")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Moniteur introuvable")
	})
}

func TestNewMonitor(t *testing.T) {
	t.Parallel()

	t.Run("The form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		body := s.get(t, permWrite, "/webui/snmp/monitors/new").Body.String()
		indexes(t, body, "Nouveau moniteur SNMP", `data-switch="version"`,
			`data-case="SNMPv2" data-of="version"`, `data-case="SNMPv3" data-of="version"`,
			`data-switch="security"`, `data-case="authNoPriv authPriv" data-of="security"`,
			"MD5 (historique)", "SHA (historique)", "SHA-512", `data-case="authPriv" data-of="security"`,
			"DES (historique)", "AES-256C", "Contexte SNMPv3")

		// The ids of the fields never take one of the icons of the kit sprite,
		// inlined in the page ("user"): the label would name the icon.
		assert.Contains(t, body, `for="username"`)
		assert.Equal(t, 1, strings.Count(body, `id="username"`))
	})

	t.Run("An SNMPv2c monitor with the default community", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, "/webui/snmp/monitors/new", url.Values{
			fieldName: {"supervision"}, fieldVersion: {versionV2}, fieldAddress: {"127.0.0.1:162"},
		})
		q := redirected(t, rec, "/webui/snmp")
		assert.Equal(t, doneCreated, q.Get(doneParam))
		assert.Equal(t, "supervision", q.Get(nameParam))

		m := s.monitor(t, "supervision")
		assert.Equal(t, versionV2, m.Version)
		assert.Equal(t, defaultCommunity, m.Community, "the product writes its default community")
		assert.Equal(t, 1, s.svc.monitors, "the SNMP service reads the monitors again")
	})

	t.Run("An SNMPv3 monitor with authentication and encryption", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		redirected(t, s.post(t, "/webui/snmp/monitors/new", v3Form("alerting")), "/webui/snmp")

		m := s.monitor(t, "alerting")
		assert.Equal(t, gwsnmp.V3SecurityAuthPriv, m.SNMPv3Security)
		assert.Equal(t, "gw", m.AuthUsername)
		assert.Equal(t, engineID, m.AuthEngineID)
		assert.Equal(t, gwsnmp.AuthSHA512, m.AuthProtocol)
		assert.Equal(t, "new-auth-pass", m.AuthPassphrase)
		assert.Equal(t, gwsnmp.PrivAES256, m.PrivProtocol)
		assert.Equal(t, "new-priv-pass", m.PrivPassphrase)
	})

	t.Run("Refused by the screen", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		for _, tc := range []struct {
			name  string
			form  url.Values
			wants []string
		}{
			{"nothing", url.Values{}, []string{"Ce champ est obligatoire.", "Choisissez une version."}},
			{"an address without port", url.Values{
				fieldName: {"m"}, fieldVersion: {versionV2}, fieldAddress: {"127.0.0.1"},
			}, []string{"Indiquez une adresse sous la forme hôte:port, avec un port de 1 à 65535."}},
			{"an SNMPv3 monitor without user", url.Values{
				fieldName: {"m"}, fieldVersion: {versionV3}, fieldAddress: {"127.0.0.1:162"},
				fieldSecurity: {gwsnmp.V3SecurityNoAuthNoPriv},
			}, []string{"Indiquez l&#39;utilisateur SNMPv3."}},
			{"an engine ID that is not a number", func() url.Values {
				f := v3Form("m")
				f.Set(fieldEngine, "80001f8880e9630000d61ff449")

				return f
			}(), []string{"Indiquez un identifiant en hexadécimal préfixé de 0x, ou un nombre décimal."}},
			{"no passphrase", func() url.Values {
				f := v3Form("m")
				f.Del(fieldAuthPass)
				f.Del(fieldPrivProtocol)

				return f
			}(), []string{"Ce champ est obligatoire.", "Choisissez un protocole de la liste."}},
		} {
			rec := s.post(t, "/webui/snmp/monitors/new", tc.form)
			require.Equalf(t, http.StatusUnprocessableEntity, rec.Code, "%s", tc.name)

			body := rec.Body.String()
			assert.Containsf(t, body, "La configuration n&#39;a pas été enregistrée.", "%s", tc.name)

			for _, want := range tc.wants {
				assert.Containsf(t, body, want, "%s", tc.name)
			}

			assert.NotContainsf(t, body, "new-priv-pass", "%s: a typed secret is never rendered back", tc.name)
		}

		count, err := s.db.Count(&gwsnmp.MonitorConfig{}).Run()
		require.NoError(t, err)
		assert.Zero(t, count)
		assert.Zero(t, s.svc.monitors)
	})

	t.Run("Refused by the model: a name taken", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, "/webui/snmp/monitors/new", v3Form("alerting-v3"))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre moniteur porte déjà ce nom.")
	})
}

func TestEditMonitor(t *testing.T) {
	t.Parallel()

	t.Run("The form keeps the secrets out", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, path(s.v3, "/edit")).Body.String()
		noSecret(t, body)
		indexes(t, body, "Modifier « alerting-v3 »", `value="alerting-v3"`, `value="SNMPv3" selected`,
			`value="authPriv" selected`, `value="gw-user"`, `value="`+engineID+`"`,
			`value="SHA-256" selected`, "Laisser vide conserve la valeur en place.", `value="AES" selected`)
	})

	t.Run("Empty passphrases keep the stored ones", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		form := v3Form("alerting-v3")
		form.Del(fieldAuthPass)
		form.Del(fieldPrivPass)
		form.Set(fieldAddress, "127.0.0.1:1162")

		q := redirected(t, s.post(t, path(s.v3, "/edit"), form), path(s.v3, ""))
		assert.Equal(t, doneUpdated, q.Get(doneParam))

		m := s.monitor(t, "alerting-v3")
		assert.Equal(t, "127.0.0.1:1162", m.UDPAddress)
		assert.Equal(t, secretAuth, m.AuthPassphrase)
		assert.Equal(t, secretPriv, m.PrivPassphrase)
		assert.Equal(t, 1, s.svc.monitors)
	})

	t.Run("A lower level drops what it no longer uses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		form := v3Form("alerting-v3")
		form.Set(fieldSecurity, gwsnmp.V3SecurityNoAuthNoPriv)

		redirected(t, s.post(t, path(s.v3, "/edit"), form), path(s.v3, ""))

		m := s.monitor(t, "alerting-v3")
		assert.Equal(t, gwsnmp.V3SecurityNoAuthNoPriv, m.SNMPv3Security)
		assert.Empty(t, m.AuthPassphrase)
		assert.Empty(t, m.PrivPassphrase)
	})

	t.Run("An SNMPv2c monitor keeps its community and its spelling of the version", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		m := &gwsnmp.MonitorConfig{
			Name: "legacy", Version: versionV2c, UDPAddress: "127.0.0.1:162", Community: secretCommunity,
		}
		require.NoError(t, s.db.Insert(m).Run())

		body := s.get(t, permWrite, path(m, "/edit")).Body.String()
		noSecret(t, body)
		assert.Contains(t, body, `value="SNMPv2" selected`)

		redirected(t, s.post(t, path(m, "/edit"), url.Values{
			fieldName: {"legacy"}, fieldVersion: {versionV2}, fieldAddress: {"127.0.0.1:163"},
		}), path(m, ""))

		m = s.monitor(t, "legacy")
		assert.Equal(t, versionV2c, m.Version)
		assert.Equal(t, secretCommunity, m.Community)
		assert.Equal(t, "127.0.0.1:163", m.UDPAddress)
	})

	t.Run("The SNMP service fails to reload", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		s.svc.reloadErr = errors.New("reload refused")

		form := v3Form("alerting-v3")
		q := redirected(t, s.post(t, path(s.v3, "/edit"), form), path(s.v3, ""))
		assert.Equal(t, failedReload, q.Get(failedParam))
		assert.Equal(t, "reload refused", q.Get(reasonParam))
	})
}

func TestDeleteMonitor(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	body := s.get(t, permAll, path(s.v2, "/delete")).Body.String()
	indexes(t, body, "Supprimer le moniteur « supervision-v2 » ?", "Retour au moniteur",
		"Ce moniteur ne recevra plus les notifications de la passerelle.", ">Supprimer</button>")

	q := redirected(t, s.post(t, path(s.v2, "/delete"), url.Values{}), "/webui/snmp")
	assert.Equal(t, doneDeleted, q.Get(doneParam))
	assert.Equal(t, "supervision-v2", q.Get(nameParam))

	err := s.db.Get(&gwsnmp.MonitorConfig{}, "name=?", "supervision-v2").Run()
	assert.True(t, database.IsNotFound(err))
	assert.Equal(t, 1, s.svc.monitors)
}

func TestAgent(t *testing.T) {
	t.Parallel()

	t.Run("Configured for the first time", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		body := s.get(t, permWrite, "/webui/snmp/agent/edit").Body.String()
		indexes(t, body, "Configurer l&#39;agent SNMP", "Adresse d&#39;écoute", "Accepter seulement SNMPv3",
			"L&#39;enregistrement redémarre le service SNMP de la passerelle.")

		q := redirected(t, s.post(t, "/webui/snmp/agent/edit", url.Values{
			fieldAddress: {"127.0.0.1:16161"}, fieldUser: {"agent"},
			fieldAuthProtocol: {gwsnmp.AuthSHA256}, fieldAuthPass: {"agent-pass"},
		}), "/webui/snmp")
		assert.Equal(t, doneAgentSaved, q.Get(doneParam))

		conf := s.agentConf(t)
		require.NotNil(t, conf)
		assert.Equal(t, "127.0.0.1:16161", conf.LocalUDPAddress)
		assert.Equal(t, defaultCommunity, conf.Community)
		assert.Equal(t, "agent-pass", conf.SNMPv3AuthPassphrase)
		assert.Equal(t, 1, s.svc.agents, "the SNMP service restarts on the new configuration")
	})

	t.Run("Refused by the screen", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		for _, tc := range []struct {
			name string
			form url.Values
			want string
		}{
			{"no address", url.Values{}, "Ce champ est obligatoire."},
			{"encryption without authentication", url.Values{
				fieldAddress: {"127.0.0.1:161"}, fieldUser: {"u"},
				fieldPrivProtocol: {gwsnmp.PrivAES}, fieldPrivPass: {"p"},
			}, "Choisissez d&#39;abord un protocole d&#39;authentification"},
			{"SNMPv3 only without user", url.Values{
				fieldAddress: {"127.0.0.1:161"}, fieldV3Only: {"1"},
			}, "Indiquez l&#39;utilisateur SNMPv3."},
			{"a protocol without passphrase", url.Values{
				fieldAddress: {"127.0.0.1:161"}, fieldUser: {"u"}, fieldAuthProtocol: {gwsnmp.AuthSHA},
			}, "Ce champ est obligatoire."},
		} {
			rec := s.post(t, "/webui/snmp/agent/edit", tc.form)
			require.Equalf(t, http.StatusUnprocessableEntity, rec.Code, "%s", tc.name)
			assert.Containsf(t, rec.Body.String(), tc.want, "%s", tc.name)
		}

		assert.Nil(t, s.agentConf(t))
		assert.Zero(t, s.svc.agents)
	})

	t.Run("Edited: empty secrets keep the stored ones", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, "/webui/snmp/agent/edit").Body.String()
		noSecret(t, body)
		indexes(t, body, "Modifier l&#39;agent SNMP", `value="127.0.0.1:16161"`, `value="agent-user"`,
			`value="SHA" selected`)

		redirected(t, s.post(t, "/webui/snmp/agent/edit", url.Values{
			fieldAddress: {"127.0.0.1:16170"}, fieldUser: {"agent-user"}, fieldAuthProtocol: {gwsnmp.AuthSHA},
			fieldV3Only: {"1"},
		}), "/webui/snmp")

		conf := s.agentConf(t)
		assert.Equal(t, "127.0.0.1:16170", conf.LocalUDPAddress)
		assert.True(t, conf.SNMPv3Only)
		assert.Equal(t, secretAgentAuth, conf.SNMPv3AuthPassphrase)
	})

	t.Run("Its configuration deleted", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, "/webui/snmp/agent/delete").Body.String()
		indexes(t, body, "Désactiver l&#39;agent SNMP ?",
			"La passerelle ne répondra plus aux requêtes SNMP ; les notifications continuent.",
			"127.0.0.1:16161", ">Désactiver</button>")

		q := redirected(t, s.post(t, "/webui/snmp/agent/delete", url.Values{}), "/webui/snmp")
		assert.Equal(t, doneAgentDisabled, q.Get(doneParam))
		assert.Nil(t, s.agentConf(t))
		assert.Equal(t, 1, s.svc.agents)

		redirected(t, s.get(t, permWrite, "/webui/snmp/agent/delete"), "/webui/snmp")
	})
}

func TestTestNotification(t *testing.T) {
	t.Parallel()

	t.Run("Sent", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		q := redirected(t, s.post(t, "/webui/snmp/test", url.Values{}), "/webui/snmp")
		assert.Equal(t, doneTested, q.Get(doneParam))
		assert.Equal(t, 1, s.svc.tests)
	})

	t.Run("Failed: the cause is shown", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.svc.testErr = errors.New("failed to send SNMP trap: unable to marshal OID")

		q := redirected(t, s.post(t, "/webui/snmp/test", url.Values{}), "/webui/snmp")
		assert.Equal(t, failedTest, q.Get(failedParam))
		assert.Equal(t, "failed to send SNMP trap: unable to marshal OID", q.Get(reasonParam))
	})

	t.Run("Without the SNMP service", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.h.service = func() service { return nil }

		q := redirected(t, s.post(t, "/webui/snmp/test", url.Values{}), "/webui/snmp")
		assert.Equal(t, failedTest, q.Get(failedParam))
		assert.Equal(t, "Le service SNMP de la passerelle ne fonctionne pas.", q.Get(reasonParam))
	})
}
