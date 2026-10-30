package smtp

import (
	"bytes"
	"context"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermRulesRead
	permWrite = model.PermRulesRead | model.PermRulesWrite
	permAll   = model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete

	alertsSender  = "alertes@passerelle.example"
	reportsSender = "rapports@passerelle.example"
	secret        = "mot-de-passe-de-test"
)

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router

	alerts, reports, partial *model.SMTPCredential
	send, receive            *model.Rule
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/smtp").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/smtp/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/smtp/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/smtp/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/smtp/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/smtp/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/smtp/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/smtp/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	return s
}

// fixtures inserts three credentials: the alerts, on the submission port
// with a login and a password, that the "flux-paie" rule uses in both
// directions; the reports, on port 465 without authentication; one the REST
// API left without host. And an e-mail template for the tasks.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.alerts = &model.SMTPCredential{
		EmailAddress: alertsSender, ServerAddress: types.Addr("smtp.example.org", 587), Login: "alertes", Password: secret,
	}
	s.reports = &model.SMTPCredential{EmailAddress: reportsSender, ServerAddress: types.Addr("smtp.example.org", 465)}
	s.partial = &model.SMTPCredential{EmailAddress: "incomplet@passerelle.example", ServerAddress: types.Addr("", 25)}

	for _, c := range []*model.SMTPCredential{s.alerts, s.reports, s.partial} {
		require.NoError(t, s.db.Insert(c).Run())
	}

	require.NoError(t, s.db.Insert(&model.EmailTemplate{Name: "fin-de-transfert", Subject: "Fin", Body: "Fichier reçu."}).Run())

	s.send = &model.Rule{Name: "flux-paie", IsSend: true, Path: "/paie-send"}
	s.receive = &model.Rule{Name: "flux-paie", IsSend: false, Path: "/paie-receive"}

	for _, r := range []*model.Rule{s.send, s.receive} {
		require.NoError(t, s.db.Insert(r).Run())
		require.NoError(t, s.db.Insert(&model.Task{
			RuleID: r.ID, Chain: model.ChainPost, Type: tasks.Email,
			Args: map[string]string{"sender": alertsSender, "recipients": "exploitation@client.example", "template": "fin-de-transfert"},
		}).Run())
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

	return s.request(t, perms, http.MethodPost, target, form)
}

func (s *suite) cred(t *testing.T, sender string) *model.SMTPCredential {
	t.Helper()

	var c model.SMTPCredential
	require.NoError(t, s.db.Get(&c, "email_address=?", sender).Run())

	return &c
}

func path(c *model.SMTPCredential, suffix string) string {
	return "/webui/smtp/" + strconv.FormatInt(c.ID, 10) + suffix
}

func tasksURL(r *model.Rule) string {
	return "/webui/rules/" + strconv.FormatInt(r.ID, 10) + "/tasks"
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

func rows(body string) int { return strings.Count(body, "<tr>") - 1 }

// TestListFragment: the region of the list alone, for the live list of the kit (#766):
// the elements of the part "list", with the ids of the page, no shell; the
// page itself declares the live region and its filters form.
func TestListFragment(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permAll, "/webui/smtp?fragment=list")
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.NotContains(t, body, "<html", "no shell around a fragment")
	assert.Contains(t, body, `<span class="w-toolbar__count" id="smtp-count" data-live-part="list" aria-live="polite">`)
	assert.Contains(t, body, `<div id="smtp-list" data-live-part="list" data-live-paged>`)

	body = s.get(t, permAll, "/webui/smtp").Body.String()
	assert.Contains(t, body, `<div data-live data-live-period="0">`, "read on the user's actions only")
	assert.Contains(t, body, `data-autosubmit data-live-filter>`, "the filters apply in place")
	assert.Contains(t, body, `<template data-live-error>`)
	assert.Contains(t, body, `live.js`)
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given a user with every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, "/webui/smtp")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Identifiants SMTP · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/smtp" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/smtp/new">`)
		assert.NotContains(t, body, "w-disclose", "no filter")
		assert.Contains(t, body, `aria-live="polite">3 identifiants</span>`)

		assert.Equal(t, 3, rows(body))
		indexes(t, body, "<tbody>", ">alertes@passerelle.example<", ">incomplet@passerelle.example<",
			">rapports@passerelle.example<", "</tbody>")
		assert.Contains(t, body, `<td class="w-mono smtp-server">smtp.example.org:587</td><td><span class="w-mono">alertes</span></td>`+
			`<td class="w-nowrap">2 règles</td>`)
		assert.Contains(t, body, `<td class="w-mono smtp-server">smtp.example.org:465</td><td><span class="w-muted">Aucune</span></td>`+
			`<td class="w-nowrap">Aucune règle</td>`)
		indexes(t, body, ">alertes@passerelle.example</a>", "Voir la fiche", "Modifier l&#39;identifiant",
			"Supprimer l&#39;identifiant", ">incomplet@passerelle.example</a>")
		assert.NotContains(t, body, secret)
	})

	t.Run("Given a search and a sort", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/smtp?q=RAP").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, ">rapports@passerelle.example</a>")

		body = s.get(t, permRead, "/webui/smtp?sort=server&dir=desc").Body.String()
		indexes(t, body, ">alertes@passerelle.example<", ">rapports@passerelle.example<", ">incomplet@passerelle.example<")

		body = s.get(t, permRead, "/webui/smtp?q=zz").Body.String()
		assert.Contains(t, body, "Aucun identifiant ne commence par « zz »")
	})

	t.Run("Given a user with the read right only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/smtp").Body.String()
		assert.NotContains(t, body, "w-btn--primary", "no coloured action without the write right")
		assert.NotContains(t, body, "Modifier l&#39;identifiant")
		assert.NotContains(t, body, "Supprimer l&#39;identifiant")
		assert.Contains(t, body, "Voir la fiche")
	})

	t.Run("Given no credential at all", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permAll, "/webui/smtp").Body.String()
		assert.Contains(t, body, "Aucun identifiant SMTP pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/smtp/new"`)
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("Given the credential the rules use", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, path(s.alerts, ""))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Identifiant SMTP « alertes@passerelle.example »")
		assert.Contains(t, body, `<code class="w-code">smtp.example.org</code>`)
		assert.Contains(t, body, `<code class="w-code">587</code>`)
		assert.Contains(t, body, "STARTTLS si le serveur le propose")
		assert.Contains(t, body, `Identifiant <code class="w-code">alertes</code>`)
		assert.Contains(t, body, "Défini, jamais affiché")
		indexes(t, body, "Utilisé par", `<ul class="ruleuse-list" aria-label="Règles dont les tâches EMAIL emploient l&#39;identifiant">`,
			`<span class="ruleuse-direction">Envoi</span> <a class="w-link ruleuse-rule" href="`+tasksURL(s.send)+`">flux-paie</a>`,
			`<span class="ruleuse-direction">Réception</span> <a class="w-link ruleuse-rule" href="`+tasksURL(s.receive)+`">flux-paie</a>`)
		assert.Contains(t, body, `href="/webui/static_v2/ruleuse/ruleuse.css"`, "the cards of the rules")
		assert.NotContains(t, body, "<table", "no table in a card")
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-key="smtp-server" open>`, "the cards after the first fold (#807)")
		assert.Contains(t, body, `<details class="w-card w-card--fold" data-key="smtp-uses" open>`)
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="`+path(s.alerts, "/edit")+`"`)
		assert.NotContains(t, body, secret)
	})

	t.Run("Given the credential on port 465 without authentication", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, path(s.reports, "")).Body.String()
		assert.Contains(t, body, "TLS dès la connexion")
		assert.Contains(t, body, "Aucune authentification")
		assert.Contains(t, body, "Aucune règle ne l&#39;emploie.")
		assert.NotContains(t, body, "w-btn--primary", "no edition without the write right")
	})

	t.Run("Given a credential the REST API left without host", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, path(s.partial, "")).Body.String()
		assert.Contains(t, body, "L&#39;adresse du serveur est incomplète.")
		assert.Contains(t, body, `<span class="w-muted">—</span>`, "no host")
	})

	t.Run("Given an unknown credential", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.get(t, permRead, "/webui/smtp/999")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Identifiant introuvable")
	})
}

func TestNewCredential(t *testing.T) {
	t.Parallel()

	t.Run("Given the empty form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permWrite, "/webui/smtp/new").Body.String()
		assert.Contains(t, body, `id="sender" name="sender"`)
		assert.Contains(t, body, `id="port" name="port" value="587"`, "the submission port proposed")
		assert.Contains(t, body, `id="password" name="password" type="password"`)
		assert.Contains(t, body, "465 : TLS dès la connexion ; 587 ou 25 : STARTTLS si le serveur le propose.")
		assert.NotRegexp(t, `<(input|select|textarea)[^>]* id="(server|user|key)"`, body,
			"no field id collides with an icon of the kit")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a credential with authentication", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/smtp/new", url.Values{
			"sender": {" " + alertsSender + " "}, "host": {"smtp.example.org"}, "port": {"587"},
			"login": {"alertes"}, "password": {secret},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), "done=created")

		c := s.cred(t, alertsSender)
		assert.Equal(t, "smtp.example.org:587", c.ServerAddress.String())
		assert.Equal(t, "alertes", c.Login)
		assert.Equal(t, secret, c.Password)

		rec = s.post(t, permWrite, "/webui/smtp/new",
			url.Values{"sender": {reportsSender}, "host": {"::1"}, "port": {"465"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "[::1]:465", s.cred(t, reportsSender).ServerAddress.String(), "an IPv6 host in brackets")
	})

	t.Run("Given forms the screen or the model refuses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		refused := func(form url.Values, want string) string {
			t.Helper()

			rec := s.post(t, permWrite, "/webui/smtp/new", form)
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

			body := rec.Body.String()
			assert.Contains(t, body, "L&#39;identifiant n&#39;a pas été enregistré.")
			assert.Contains(t, body, want)

			return body
		}

		valid := func(change map[string]string) url.Values {
			form := url.Values{"sender": {"nouveau@passerelle.example"}, "host": {"smtp.example.org"}, "port": {"25"}}
			for k, v := range change {
				form.Set(k, v)
			}

			return form
		}

		refused(valid(map[string]string{"sender": ""}), "Ce champ est obligatoire.")
		refused(valid(map[string]string{"sender": "Passerelle <nouveau@passerelle.example>"}), "Indiquez une adresse e-mail seule")
		refused(valid(map[string]string{"sender": "pas-une-adresse"}), "Indiquez une adresse e-mail seule")
		refused(valid(map[string]string{"host": ""}), "Indiquez l&#39;hôte.")
		refused(valid(map[string]string{"port": "0"}), "Indiquez un port entre 1 et 65535.")
		refused(valid(map[string]string{"port": "70000"}), "Indiquez un port entre 1 et 65535.")

		body := refused(valid(map[string]string{"password": secret}), "Indiquez l&#39;identifiant, ou laissez le mot de passe vide.")
		assert.NotContains(t, body, secret, "a password typed is never rendered back")

		body = refused(valid(map[string]string{"sender": alertsSender, "login": "x", "password": "autre-secret"}),
			"Un autre identifiant utilise déjà cette adresse.")
		assert.NotContains(t, body, "autre-secret")

		count, err := s.db.Count(&model.SMTPCredential{}).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 3, count, "nothing created")
	})
}

func TestEditCredential(t *testing.T) {
	t.Parallel()

	t.Run("Given a credential no rule uses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		k := &model.SMTPCredential{
			EmailAddress: "libre@passerelle.example", ServerAddress: types.Addr("smtp.example.org", 587),
			Login: "libre", Password: secret,
		}
		require.NoError(t, s.db.Insert(k).Run())

		body := s.get(t, permWrite, path(k, "/edit")).Body.String()
		assert.Contains(t, body, `id="sender" name="sender" value="libre@passerelle.example"`)
		assert.Contains(t, body, `id="host" name="host" value="smtp.example.org"`)
		assert.Contains(t, body, `id="port" name="port" value="587"`)
		assert.Contains(t, body, "Laissez vide pour conserver le mot de passe en place.")
		assert.NotContains(t, body, secret)

		rec := s.post(t, permWrite, path(k, "/edit"), url.Values{
			"sender": {"renomme@passerelle.example"}, "host": {"relais.example.org"}, "port": {"465"}, "login": {"libre"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), path(k, "")+"?done=updated")

		c := s.cred(t, "renomme@passerelle.example")
		assert.Equal(t, "relais.example.org:465", c.ServerAddress.String())
		assert.Equal(t, secret, c.Password, "an empty password keeps the stored one")

		rec = s.post(t, permWrite, path(k, "/edit"), url.Values{
			"sender": {"renomme@passerelle.example"}, "host": {"relais.example.org"}, "port": {"465"}, "login": {""},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		c = s.cred(t, "renomme@passerelle.example")
		assert.Empty(t, c.Login)
		assert.Empty(t, c.Password, "a cleared login removes the authentication, password included")
	})

	t.Run("Given a sender with a display name the REST API stored", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		k := &model.SMTPCredential{EmailAddress: "Passerelle <nom@passerelle.example>", ServerAddress: types.Addr("smtp.example.org", 25)}
		require.NoError(t, s.db.Insert(k).Run())

		rec := s.post(t, permWrite, path(k, "/edit"), url.Values{
			"sender": {k.EmailAddress}, "host": {"smtp.example.org"}, "port": {"587"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, "the sender in place is kept as it is")
		assert.Equal(t, "smtp.example.org:587", s.cred(t, k.EmailAddress).ServerAddress.String())
	})

	t.Run("Given a credential the rules use", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, path(s.alerts, "/edit")).Body.String()
		assert.Contains(t, body, "Employé par 2 règles : l&#39;adresse se conserve")
		assert.NotContains(t, body, `name="sender"`, "the sender is shown in reading")
		assert.Contains(t, body, `name="host"`)

		rec := s.post(t, permWrite, path(s.alerts, "/edit"), url.Values{
			"sender": {"autre@passerelle.example"}, "host": {"relais.example.org"}, "port": {"587"}, "login": {"alertes"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		c := s.cred(t, alertsSender)
		assert.Equal(t, "relais.example.org:587", c.ServerAddress.String(), "the server is changed, the sender kept")
		assert.Equal(t, secret, c.Password)
	})
}

func TestDeleteCredential(t *testing.T) {
	t.Parallel()

	t.Run("Given a credential no rule uses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, path(s.reports, "/delete")).Body.String()
		assert.Contains(t, body, "Supprimer l&#39;identifiant « rapports@passerelle.example » ?")
		assert.Contains(t, body, "Aucune règle ne l&#39;emploie. Il sera supprimé définitivement.")
		assert.Contains(t, body, ">Supprimer</button>")

		rec := s.post(t, permAll, path(s.reports, "/delete"), url.Values{})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=deleted")

		count, err := s.db.Count(&model.SMTPCredential{}).Where("email_address=?", reportsSender).Run()
		require.NoError(t, err)
		assert.Zero(t, count)
	})

	t.Run("Given a credential the rules use", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, path(s.alerts, "/delete")).Body.String()
		assert.Contains(t, body, "Cet identifiant ne peut pas être supprimé.")
		assert.Contains(t, body, "Il est employé par 2 règles : changez d&#39;abord l&#39;expéditeur de leurs tâches EMAIL")
		assert.Contains(t, body, `href="`+tasksURL(s.send)+`">flux-paie</a>`)
		assert.NotContains(t, body, ">Supprimer</button>", "no deletion")
		assert.Contains(t, body, ">Fermer</a>")

		rec := s.post(t, permAll, path(s.alerts, "/delete"), url.Values{})
		assert.Equal(t, http.StatusConflict, rec.Code)
		assert.Equal(t, secret, s.cred(t, alertsSender).Password, "the credential stays")
	})
}
