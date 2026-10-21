package emailtemplates

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"slices"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermRulesRead
	permWrite = model.PermRulesRead | model.PermRulesWrite
	permAll   = model.PermRulesRead | model.PermRulesWrite | model.PermRulesDelete

	sender = "alertes@passerelle.example"
)

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router

	alert, report, other *model.EmailTemplate
	send, receive        *model.Rule
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/email-templates").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/email-templates/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/email-templates/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/email-templates/preview").Methods(http.MethodPost).HandlerFunc(h.Preview)
	r.Path("/email-templates/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/email-templates/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/email-templates/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/email-templates/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/email-templates/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	return s
}

// fixtures inserts three templates: the alert, in HTML, that the "flux-paie"
// rule uses in both directions; the report, in text, with two attachments;
// one the REST API left with another MIME type. And the SMTP credential the
// EMAIL tasks name.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	require.NoError(t, s.db.Insert(&model.SMTPCredential{
		EmailAddress: sender, ServerAddress: types.Addr("smtp.example.org", 587),
	}).Run())

	s.alert = &model.EmailTemplate{
		Name: "alerte-erreur", Subject: "Transfert #TRANSFERID# en erreur", MIMEType: "text/html",
		Body: "<p>Le fichier <b>#TRUEFILENAME#</b> de la règle #RULE# : #ERRORMSG#</p>",
	}
	s.report = &model.EmailTemplate{
		Name: "rapport", Subject: "Rapport du #DATE#", MIMEType: "text/plain", Body: "Ci-joint le rapport.",
		Attachments: []string{"#TRUEFULLPATH#", "/var/rapports/synthese.pdf"},
	}
	s.other = &model.EmailTemplate{Name: "ancien", Subject: "Image", MIMEType: "image/png", Body: "Corps"}

	for _, tpl := range []*model.EmailTemplate{s.alert, s.report, s.other} {
		require.NoError(t, s.db.Insert(tpl).Run())
	}

	s.send = &model.Rule{Name: "flux-paie", IsSend: true, Path: "/paie-send"}
	s.receive = &model.Rule{Name: "flux-paie", IsSend: false, Path: "/paie-receive"}

	for _, r := range []*model.Rule{s.send, s.receive} {
		require.NoError(t, s.db.Insert(r).Run())
		require.NoError(t, s.db.Insert(&model.Task{
			RuleID: r.ID, Chain: model.ChainError, Type: tasks.Email,
			Args: map[string]string{"sender": sender, "recipients": "exploitation@client.example", "template": "alerte-erreur"},
		}).Run())
	}
}

// history inserts a finished transfer of the sending rule, in error.
func (s *suite) history(t *testing.T) *model.HistoryEntry {
	t.Helper()

	local := "/data/out/paie-octobre.xml"
	if runtime.GOOS == "windows" {
		local = "C:" + local
	}

	h := &model.HistoryEntry{
		Identifier: model.ID(1187), RemoteTransferID: "88412", IsSend: true, Rule: "flux-paie", Account: "gw",
		Agent: "cabinet-paie", Client: "cli-sftp", Protocol: "sftp", SrcFilename: "paie-octobre.xml",
		LocalPath: local, RemotePath: "/in/paie-octobre.xml",
		Start: time.Date(2026, 10, 20, 17, 40, 2, 0, time.Local), Stop: time.Date(2026, 10, 20, 17, 40, 31, 0, time.Local),
		Status: types.StatusCancelled, ErrCode: types.TeConnection, ErrDetails: "connexion refusée par le partenaire",
	}
	require.NoError(t, s.db.Insert(h).Run())

	return h
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

func (s *suite) stored(t *testing.T, name string) *model.EmailTemplate {
	t.Helper()

	var tpl model.EmailTemplate
	require.NoError(t, s.db.Get(&tpl, "name=?", name).Run())

	return &tpl
}

func pagePathOf(tpl *model.EmailTemplate, suffix string) string {
	return "/webui/email-templates/" + strconv.FormatInt(tpl.ID, 10) + suffix
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

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given a user with every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, "/webui/email-templates")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Modèles d&#39;e-mail · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/email-templates" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/email-templates/new">`)
		assert.Contains(t, body, `aria-live="polite">3 modèles</span>`)
		assert.NotContains(t, body, "editor.js", "the list does not load the editor")

		assert.Equal(t, 3, rows(body))
		indexes(t, body, "<tbody>", ">alerte-erreur<", ">ancien<", ">rapport<", "</tbody>")
		indexes(t, body, ">alerte-erreur</a>", "Transfert #TRANSFERID# en erreur", "<td>HTML</td>",
			`<td class="w-nowrap">Aucune</td>`, `<td class="w-nowrap">2 règles</td>`)
		indexes(t, body, ">ancien</a>", "<td>image/png</td>", ">rapport</a>", "<td>Texte</td>",
			`<td class="w-nowrap">2 pièces jointes</td>`, `<td class="w-nowrap">Aucune règle</td>`)
		indexes(t, body, ">alerte-erreur</a>", "Voir la fiche", "Modifier le modèle", "Supprimer le modèle", ">ancien</a>")
	})

	t.Run("Given a search, a sort, and a user who may only read", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/email-templates?q=RAP").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, ">rapport<")
		assert.Contains(t, body, "3 modèles · 1 affiché")
		assert.NotContains(t, body, "/webui/email-templates/new", "no creation without the write right")
		assert.NotContains(t, body, "Modifier le modèle")

		body = s.get(t, permRead, "/webui/email-templates?sort=subject&dir=desc").Body.String()
		indexes(t, body, "<tbody>", ">alerte-erreur<", ">rapport<", ">ancien<", "</tbody>")

		body = s.get(t, permRead, "/webui/email-templates?q=zz").Body.String()
		assert.Contains(t, body, "Aucun modèle ne commence par « zz »")
	})

	t.Run("Given no template", func(t *testing.T) {
		t.Parallel()

		body := newSuite(t).get(t, permAll, "/webui/email-templates").Body.String()
		assert.Contains(t, body, "Aucun modèle d&#39;e-mail pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/email-templates/new"`)
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("Given a template the rules use, and no transfer of theirs", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, pagePathOf(s.alert, ""))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "¿")
		assert.Contains(t, body, "<h1 class=\"w-page-header__title\">Modèle d&#39;e-mail « alerte-erreur »</h1>")
		indexes(t, body, "Message", "Transfert #TRANSFERID# en erreur", "HTML", "Aperçu", "Pièces jointes",
			"Utilisé par", "flux-paie", "Envoi", "flux-paie", "Réception")
		assert.Contains(t, body, "Valeurs d&#39;exemple : aucun transfert n&#39;a encore employé ce modèle.")
		assert.Contains(t, body, "<dd>Transfert 1234 en erreur</dd>", "the subject with the example values")
		assert.Contains(t, body, `sandbox="" srcdoc="`, "the body in a frame that runs nothing")
		assert.Contains(t, body, "factures-2026-10.csv", "the example file")
		assert.Contains(t, body, `<form class="et-preview__controls" method="get" action="/webui/email-templates/`)
		assert.NotContains(t, body, `class="et-mark"`, "the values are not highlighted by default")
	})

	t.Run("Given a recent transfer of a rule that uses it, chosen and highlighted", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)
		h := s.history(t)

		body := s.get(t, permRead, pagePathOf(s.alert, "")).Body.String()
		assert.Contains(t, body, `<option value="1187">Transfert 1187 · flux-paie · 20/10/2026 17:40</option>`)
		assert.Contains(t, body, "Valeurs d&#39;exemple ; choisissez l&#39;un des derniers transferts")

		body = s.get(t, permRead, pagePathOf(s.alert, "?values="+strconv.FormatInt(h.ID, 10)+"&mark=1")).Body.String()
		assert.Contains(t, body, `<option value="1187" selected>`)
		assert.Contains(t, body, "Valeurs du transfert choisi, telles que la tâche EMAIL les lit.")
		assert.Contains(t, body, `<dd>Transfert <mark class="et-mark">1187</mark> en erreur</dd>`)
		assert.Contains(t, body, "&lt;mark&gt;paie-octobre.xml&lt;/mark&gt;", "the file of the transfer, highlighted in the body")
		assert.Contains(t, body, "connexion refusée par le partenaire")
		assert.Contains(t, body, `name="mark" value="1" id="et-mark" checked`)
	})

	t.Run("Given a text template with attachments", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, pagePathOf(s.report, "")).Body.String()
		indexes(t, body, "Pièces jointes", "#TRUEFULLPATH#", "/var/rapports/synthese.pdf", "Utilisé par", "Aucune règle ne l&#39;emploie.")
		assert.Contains(t, body, "factures-2026-10.csv</span>", "the attachment path substituted in the preview")
		assert.Contains(t, body, `&lt;div class=&#34;plain&#34;&gt;Ci-joint le rapport.&lt;/div&gt;`)
	})

	t.Run("Given a body whose HTML holds a script", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		tpl := &model.EmailTemplate{Name: "piege", Subject: "x", MIMEType: "text/html", Body: `<p>a</p><script>alert(1)</script>`}
		require.NoError(t, s.db.Insert(tpl).Run())

		body := s.get(t, permRead, pagePathOf(tpl, "?mark=1")).Body.String()
		assert.NotContains(t, body, "<script>alert", "the body only lives in the srcdoc attribute, escaped")
		assert.Contains(t, body, "&lt;script&gt;alert(1)&lt;/script&gt;")
		assert.Contains(t, body, `sandbox=""`)
	})

	t.Run("Given a template that does not exist", func(t *testing.T) {
		t.Parallel()

		rec := newSuite(t).get(t, permRead, "/webui/email-templates/999")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Modèle introuvable")
	})
}

func TestForm(t *testing.T) {
	t.Parallel()

	t.Run("Given the creation form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.get(t, permWrite, "/webui/email-templates/new")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.NotContains(t, body, "¿")
		assert.Contains(t, body, `editor.css`)
		assert.Contains(t, body, `editor.js`)
		assert.Contains(t, body, `/static_v2/email_templates/email-templates.js`)
		assert.Contains(t, body, `<script id="email-template-vars" type="application/json">`)
		assert.Contains(t, body, `"name":"TRUEFILENAME","label":"Nom du fichier"`)
		assert.Contains(t, body, `"name":"STARTTIMESTAMP(YYYY-MM-DD_HHmmss)"`)
		assert.Contains(t, body, `"name":"TI_clé","label":"Information attachée au transfert`)
		assert.NotContains(t, body, `"name":"ARCHPATH"`, "a marker the tasks do not implement is never offered")
		assert.NotContains(t, body, `"name":"ERRORSTRCODE"`)
		assert.NotContains(t, body, `"code":"pm"`)
		assert.Contains(t, body, `class="w-rte" data-rte=`)
		assert.Contains(t, body, `<option value="text/html" selected>HTML</option>`)
		assert.Contains(t, body, `data-preview="/webui/email-templates/preview"`)
		assert.Contains(t, body, `id="subject" name="subject" value="" autocomplete="off" data-vars="email-template-vars"`)
	})

	t.Run("Given a template the rules use", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, pagePathOf(s.alert, "/edit")).Body.String()
		assert.Contains(t, body, "Employé par 2 règles : le nom se conserve")
		assert.NotContains(t, body, `id="name"`)
		assert.Contains(t, body, `<input type="hidden" name="template_id"`)

		rec := s.post(t, permWrite, pagePathOf(s.alert, "/edit"), url.Values{
			"name": {"renomme"}, "format": {"text/html"}, "subject": {"Erreur #TRANSFERID#"},
			"body_html": {"<p>Le fichier #TRUEFILENAME#.</p>"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)

		saved := s.stored(t, "alerte-erreur")
		assert.Equal(t, "Erreur #TRANSFERID#", saved.Subject, "the template is saved, its name kept")
		assert.Equal(t, "<p>Le fichier #TRUEFILENAME#.</p>", saved.Body)
	})

	t.Run("Given a new template", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/email-templates/new", url.Values{
			"name": {"fin"}, "format": {"text/plain"}, "subject": {"Fin #TIMESTAMP(DD/MM/YYYY à HH:mm)#"},
			"body_text": {"Le fichier #TI_lot# est arrivé."}, "body_html": {"<p>ignoré</p>"},
			"attachments": {" #TRUEFULLPATH# \n\n/var/rapport.pdf\n"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Contains(t, rec.Header().Get("Location"), "done=created")

		saved := s.stored(t, "fin")
		assert.Equal(t, "text/plain", saved.MIMEType)
		assert.Equal(t, "Le fichier #TI_lot# est arrivé.", saved.Body, "the body of the format chosen")
		assert.Equal(t, []string{"#TRUEFULLPATH#", "/var/rapport.pdf"}, []string(saved.Attachments))
	})

	t.Run("Given a refused form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/email-templates/new", url.Values{
			"name": {"rapport"}, "format": {"text/html"}, "subject": {"Transfert #FOO# #ARCHPATH#"},
			"body_html": {"<p><br></p>"}, "attachments": {"#date#"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Le modèle n&#39;a pas été enregistré.")
		assert.Contains(t, body, "Variables que les tâches ne connaissent pas : #FOO#, #ARCHPATH#.")
		assert.Contains(t, body, "Variables que les tâches ne connaissent pas : #date#.")
		assert.Contains(t, body, "Ce champ est obligatoire.", "an HTML body without text")
		assert.Contains(t, body, `value="Transfert #FOO# #ARCHPATH#"`, "the values are kept")

		rec = s.post(t, permWrite, "/webui/email-templates/new", url.Values{
			"name": {"rapport"}, "format": {"text/plain"}, "subject": {"x"}, "body_text": {"y"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre modèle porte déjà ce nom.")

		rec = s.post(t, permWrite, "/webui/email-templates/new", url.Values{
			"name": {"image"}, "format": {"image/png"}, "subject": {"x"}, "body_text": {"y"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Choisissez le format Texte ou HTML.")
	})

	t.Run("Given a template with another MIME type", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, pagePathOf(s.other, "/edit")).Body.String()
		assert.Contains(t, body, `<option value="image/png" selected>image/png</option>`)
		assert.Contains(t, body, `data-case="text/plain image/png"`)

		rec := s.post(t, permWrite, pagePathOf(s.other, "/edit"), url.Values{
			"name": {"ancien"}, "format": {"image/png"}, "subject": {"Image"}, "body_text": {"Corps modifié"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, "the type in place stays accepted")
		assert.Equal(t, "image/png", s.stored(t, "ancien").MIMEType)
	})
}

func TestPreview(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.post(t, permRead, "/webui/email-templates/preview", url.Values{
		"format": {"text/html"}, "subject": {"Transfert #TRANSFERID#"}, "body_html": {"<p>#TRUEFILENAME#</p>"}, "mark": {"1"},
	})
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.NotContains(t, body, "<html", "a fragment, not a page")
	assert.Contains(t, body, `<dd>Transfert <mark class="et-mark">1234</mark></dd>`)

	body = s.post(t, permRead, "/webui/email-templates/preview", url.Values{
		"format": {"text/plain"}, "subject": {"#ARCHPATH#"}, "body_text": {"x"},
	}).Body.String()
	assert.Contains(t, body, "La tâche EMAIL échouerait avec ce modèle.")
	assert.Contains(t, body, "#ARCHPATH#")

	h := s.history(t)
	body = s.post(t, permRead, "/webui/email-templates/preview", url.Values{
		"template_id": {strconv.FormatInt(s.alert.ID, 10)}, "values": {strconv.FormatInt(h.ID, 10)},
		"format": {"text/plain"}, "subject": {"Transfert #TRANSFERID#"}, "body_text": {"x"},
	}).Body.String()
	assert.Contains(t, body, "<dd>Transfert 1187</dd>", "the values of a transfer of the rules that use the template")
}

func TestDelete(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	rec := s.get(t, permAll, pagePathOf(s.alert, "/delete"))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "Ce modèle ne peut pas être supprimé.")
	assert.NotContains(t, rec.Body.String(), "w-btn--danger")

	rec = s.post(t, permAll, pagePathOf(s.alert, "/delete"), url.Values{})
	assert.Equal(t, http.StatusConflict, rec.Code)
	s.stored(t, "alerte-erreur")

	rec = s.get(t, permAll, pagePathOf(s.report, "/delete"))
	assert.Contains(t, rec.Body.String(), "Aucune règle ne l&#39;emploie. Il sera supprimé définitivement.")

	rec = s.post(t, permAll, pagePathOf(s.report, "/delete"), url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=deleted")
	assert.True(t, database.IsNotFound(s.db.Get(&model.EmailTemplate{}, "name=?", "rapport").Run()))
}

// The variables the list offers are those the tasks implement (pkg/tasks),
// each with a label in French and in English.
func TestVariablesFollowTheTasks(t *testing.T) {
	t.Parallel()

	// Implemented but not offered: a duplicate of #ERRORMSG# and a deprecated marker.
	notOffered := []string{"ERRORSTRCODE", "FULLTRANSFERID"}

	var grouped []string
	for _, g := range varGroups {
		grouped = append(grouped, g.names...)
	}

	implemented := append(tasks.Markers(), tasks.FormatMarkers()...)
	for _, name := range implemented {
		assert.Truef(t, slices.Contains(grouped, name) || slices.Contains(notOffered, name),
			"the marker %q the tasks implement must be in a group, or set aside", name)
	}

	for _, name := range append(slices.Clone(grouped), notOffered...) {
		assert.Truef(t, name == transferInfoName || slices.Contains(implemented, name),
			"%q is not a marker the tasks implement", name)
	}

	catalog := common.NewShell().Catalog()
	now := time.Date(2026, 10, 21, 9, 12, 5, 0, time.Local)

	for _, lang := range []string{"fr", "en"} {
		translate := catalog.Translator(lang)
		list := varList(translate, now)

		for _, g := range list.Groups {
			assert.NotContains(t, g.Label, "¿")

			for _, v := range g.Vars {
				assert.NotContainsf(t, v.Label, "¿", "%s: the label of %s", lang, v.Name)

				for _, tok := range v.Tokens {
					assert.NotContainsf(t, tok.Label, "¿", "%s: the label of the element %s", lang, tok.Code)
					assert.NotEmpty(t, tok.Example)
				}
			}
		}
	}

	for _, code := range tasks.TimestampTokens() {
		assert.Containsf(t, tokenKeys, code, "the element %q of a date format needs a label", code)
	}

	assert.NotEqual(t, "AM", strings.ToUpper(tasks.FormatTimestamp("(pm)", now)),
		"pm is set aside while the tasks render it wrong: offer it again once they do")
}
