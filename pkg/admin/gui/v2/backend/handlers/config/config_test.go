package config

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
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
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermAdminRead
	permWrite = model.PermAdminRead | model.PermAdminWrite
)

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path(pagePath).Methods(http.MethodGet).HandlerFunc(h.Page)
	r.Path(exportPath).Methods(http.MethodGet).HandlerFunc(h.Export)
	r.Path(importPath).Methods(http.MethodPost).HandlerFunc(h.Import)

	return s
}

// fixtures: two rules, so that the export has something to carry.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	for _, rule := range []*model.Rule{
		{Name: "releves", IsSend: true, Path: "releves"},
		{Name: "commandes", IsSend: false, Path: "commandes"},
	} {
		require.NoError(t, s.db.Insert(rule).Run())
	}
}

func (s *suite) request(t *testing.T, perms model.PermsMask, method, target string, body *bytes.Buffer,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")

	var req *http.Request
	if body != nil {
		req = httptest.NewRequestWithContext(ctx, method, target, body)
		req.Header.Set("Content-Type", contentType)
	} else {
		req = httptest.NewRequestWithContext(ctx, method, target, nil)
	}

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

func (s *suite) get(t *testing.T, perms model.PermsMask, target string) *httptest.ResponseRecorder {
	t.Helper()

	return s.request(t, perms, http.MethodGet, target, nil, "")
}

// upload posts the import form as multipart: the fields, then the file of
// the given name and content (none when the name is empty).
func (s *suite) upload(t *testing.T, fields url.Values, name string, content []byte) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	for key, values := range fields {
		for _, v := range values {
			require.NoError(t, w.WriteField(key, v))
		}
	}

	if name != "" {
		part, err := w.CreateFormFile(fieldFile, name)
		require.NoError(t, err)
		_, err = part.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())

	return s.request(t, permWrite, http.MethodPost, "/webui"+importPath, body, w.FormDataContentType())
}

// redirected asserts a 303 to the page and returns the query it carries.
func redirected(t *testing.T, rec *httptest.ResponseRecorder) url.Values {
	t.Helper()

	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

	u, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/webui/config", u.Path)

	return u.Query()
}

func (s *suite) rules(t *testing.T) []string {
	t.Helper()

	var rules model.Rules
	require.NoError(t, s.db.Select(&rules).OrderBy("name", true).Run())

	names := make([]string, 0, len(rules))
	for _, r := range rules {
		names = append(names, r.Name)
	}

	return names
}

func allTargets() url.Values {
	v := url.Values{}
	for _, c := range categories {
		v.Add(fieldTarget, c)
	}

	return v
}

func TestPage(t *testing.T) {
	t.Parallel()

	t.Run("Given a user who may write", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.get(t, permWrite, "/webui/config")
		require.Equal(t, http.StatusOK, rec.Code)
		body := rec.Body.String()

		assert.Contains(t, body, "<title>Import et export de la configuration · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<h1 class="w-page-header__title">Import et export de la configuration</h1>`)
		assert.Contains(t, body, `<li><span>Administration</span></li><li><span aria-current="page">Import et export de la configuration</span></li>`)
		assert.Contains(t, body, `href="/webui/config" aria-current="page"`, "the entry of the sidebar")

		// The export card: every category checked, JSON, the secrets said,
		// an outline button.
		assert.Contains(t, body, `<form class="w-form cfg-form" method="get" action="/webui/config/export" id="cfg-export">`)
		assert.Equal(t, 2*len(categories), strings.Count(body, `name="target" value="`), "every category on both cards")
		assert.Len(t, regexp.MustCompile(`id="cfg-(export|import)-[a-z]+" name="target" value="[a-z]+" checked>`).FindAllString(body, -1),
			2*len(categories), "all checked on both cards")
		assert.Contains(t, body, `<input type="checkbox" id="cfg-export-servers" name="target" value="servers" checked> <span>Serveurs locaux, avec leurs comptes et leur authentification</span>`)
		assert.Contains(t, body, `<input type="radio" name="format" value="json" checked>`)
		assert.Contains(t, body, `<input type="radio" name="format" value="yaml">`)
		assert.Contains(t, body, `<strong>Le fichier contient les secrets en clair.</strong>`)
		assert.Contains(t, body, `<button class="w-btn w-btn--outline-dark" type="submit" id="cfg-export-go">Exporter la configuration</button>`)

		// The import card: the file, the options, the simulation first, one
		// primary action.
		assert.Contains(t, body, `<form class="w-form cfg-form" method="post" action="/webui/config/import" enctype="multipart/form-data" id="cfg-import" data-noguard>`, "not guarded against leaving: the export is a submission of the other form")
		assert.Contains(t, body, `<input type="file" id="cfg-file" name="file" accept=".json,.yaml,.yml,application/json,application/yaml" required aria-describedby="cfg-file-h">`)
		assert.Contains(t, body, `L&#39;import est tout ou rien : la première erreur annule l&#39;ensemble.`)
		assert.Contains(t, body, `<input type="checkbox" name="restart" value="1"> <span>Redémarrer les services concernés (serveurs locaux, clients, surveillances)</span>`)
		assert.Contains(t, body, `<input type="checkbox" name="reset" value="1" id="cfg-reset"> <span>Vider ces catégories avant l&#39;import</span>`)
		assert.Contains(t, body, `<input type="checkbox" name="reset_confirm" value="1" id="cfg-reset-confirm" aria-describedby="cfg-confirm-h"> <span>Je confirme la suppression</span>`)
		assert.Contains(t, body, `name="action" value="simulate" id="cfg-simulate">Simuler l&#39;import</button>`)
		assert.Contains(t, body, `<button class="w-btn w-btn--primary cfg-go" type="submit" name="action" value="import" id="cfg-import-go">Importer</button>`)
		assert.Contains(t, body, `<button class="w-btn w-btn--danger cfg-go-reset" type="submit" name="action" value="import" id="cfg-import-reset-go">Vider et importer</button>`)
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"), "one primary action on the page")
		assert.NotContains(t, body, "w-toast--")
	})

	t.Run("Given a user who may only read", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permRead, "/webui/config").Body.String()

		assert.Contains(t, body, `id="cfg-export">`, "the export is for the reader")
		assert.NotContains(t, body, `id="cfg-import">`, "no import form")
		assert.Contains(t, body, `<p class="w-empty__title">L&#39;import demande le droit d&#39;écriture sur l&#39;administration.</p>`)
	})

	t.Run("Given the results of the previous action", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permWrite, "/webui/config?done=simulated&categories=R%C3%A8gles+de+transfert").Body.String()
		assert.Contains(t, body, `<strong>Le fichier est valide : l&#39;import passerait (Règles de transfert).</strong>`)
		assert.NotContains(t, body, "redémarrés")

		body = s.get(t, permWrite, "/webui/config?done=imported&categories=Utilisateurs&restarted=1").Body.String()
		assert.Contains(t, body, `<strong>Configuration importée (Utilisateurs).</strong>`)
		assert.Contains(t, body, `Les services concernés ont été redémarrés.`)

		body = s.get(t, permWrite, "/webui/config?done=bogus").Body.String()
		assert.NotContains(t, body, "w-toast--")
	})
}

func TestExport(t *testing.T) {
	t.Parallel()

	t.Run("Given categories and a format", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permRead, "/webui/config/export?target=rules&target=users&target=bogus&format=yaml")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/yaml; charset=utf-8", rec.Header().Get("Content-Type"))
		assert.Regexp(t, regexp.MustCompile(`^attachment; filename="waarp-gateway-gw-test-\d{8}-\d{6}\.yaml"$`),
			rec.Header().Get("Content-Disposition"))
		body := rec.Body.String()
		assert.True(t, strings.HasPrefix(body, "# Created on "), "the header of the product: %q", body[:40])
		assert.Contains(t, body, "rules:\n")
		assert.Contains(t, body, "- name: commandes\n")
		assert.Contains(t, body, "users:\n")
		assert.NotContains(t, body, "locals:", "an unchosen category")
		assert.NotContains(t, body, "bogus")

		rec = s.get(t, permRead, "/webui/config/export?target=rules")
		require.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"), "JSON by default")
		assert.Regexp(t, `\.json"$`, rec.Header().Get("Content-Disposition"))

		var data map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &data))
		assert.Len(t, data["rules"], 2)
		assert.NotContains(t, data, "users")
	})

	t.Run("Given no category", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		for _, target := range []string{"", "?target=bogus&format=yaml"} {
			rec := s.get(t, permRead, "/webui/config/export"+target)
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
			body := rec.Body.String()
			assert.Contains(t, body, `<span class="w-field__error" id="cfg-export-cat-h">`)
			assert.Contains(t, body, `Choisissez au moins une catégorie`)
			assert.Contains(t, body, `<h1 class="w-page-header__title">Import et export de la configuration</h1>`, "the page comes back")
		}

		body := s.get(t, permRead, "/webui/config/export?target=bogus&format=yaml").Body.String()
		assert.Contains(t, body, `<input type="radio" name="format" value="yaml" checked>`, "the format posted stays")
		assert.NotContains(t, body, ` checked> <span>Serveurs locaux`, "the categories posted stay")
	})
}

func TestImport(t *testing.T) {
	t.Parallel()

	newRule := []byte(`{"rules": [{"name": "imported", "isSend": true, "path": "imported"}]}`)

	t.Run("Given a simulation of the export of the instance", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		export := s.get(t, permRead, "/webui/config/export?"+allTargets().Encode()).Body.Bytes()
		fields := allTargets()
		fields.Set(fieldAction, actionSimulate)
		fields.Set(fieldRestart, "1")

		query := redirected(t, s.upload(t, fields, "export.json", export))
		assert.Equal(t, doneSimulated, query.Get(doneParam))
		assert.Equal(t, "Serveurs locaux, avec leurs comptes et leur authentification, Partenaires distants, avec leurs "+
			"comptes et leur authentification, Clients locaux, Règles de transfert, Utilisateurs, Instances cloud, SNMP, "+
			"Autorités d'authentification, Clés cryptographiques, E-mail, identifiants SMTP et modèles, "+
			"Surveillances de partenaires", query.Get(categoriesParam))
		assert.Empty(t, query.Get(restartedParam), "no restart on a simulation")
		assert.Equal(t, []string{"commandes", "releves"}, s.rules(t), "nothing changed")
	})

	t.Run("Given an import of a new rule", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		fields := url.Values{fieldTarget: {"rules"}, fieldAction: {actionImport}, fieldRestart: {"1"}}
		query := redirected(t, s.upload(t, fields, "conf.JSON", newRule))
		assert.Equal(t, doneImported, query.Get(doneParam))
		assert.Equal(t, "Règles de transfert", query.Get(categoriesParam))
		assert.Equal(t, "1", query.Get(restartedParam))
		assert.Equal(t, []string{"commandes", "imported", "releves"}, s.rules(t))
	})

	t.Run("Given an import that empties the category first", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		fields := url.Values{fieldTarget: {"rules"}, fieldAction: {actionImport}, fieldReset: {"1"}}
		rec := s.upload(t, fields, "conf.yaml", []byte("rules:\n- name: imported\n  isSend: true\n  path: imported\n"))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "without the confirmation")
		body := rec.Body.String()
		assert.Contains(t, body, `<span class="w-field__error" id="cfg-confirm-h">`)
		assert.Contains(t, body, `Confirmez la suppression pour vider les catégories`)
		assert.Contains(t, body, `name="reset" value="1" id="cfg-reset" checked>`, "the options posted stay")
		assert.Equal(t, []string{"commandes", "releves"}, s.rules(t), "nothing changed")

		fields.Set(fieldResetConfirm, "1")
		query := redirected(t, s.upload(t, fields, "conf.yaml", []byte("rules:\n- name: imported\n  isSend: true\n  path: imported\n")))
		assert.Equal(t, doneImported, query.Get(doneParam))
		assert.Equal(t, []string{"imported"}, s.rules(t), "the rules of the instance are gone")
	})

	t.Run("Given a file the product refuses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		fields := url.Values{fieldTarget: {"rules"}, fieldAction: {actionSimulate}}
		rec := s.upload(t, fields, "conf.json", []byte(`{"rules": [{"name": "", "isSend": true, "path": "x"}]}`))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, `<strong>L&#39;import a été refusé.</strong>`)
		assert.Contains(t, body, `<pre class="cfg-reason"><code>`)
		assert.Contains(t, body, `the rule&#39;s name cannot be empty`, "the message of the product, as is")
		assert.Equal(t, []string{"commandes", "releves"}, s.rules(t))
	})

	t.Run("Given a file that cannot be read", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		fields := url.Values{fieldTarget: {"rules"}, fieldAction: {actionSimulate}}

		rec := s.upload(t, fields, "conf.json", []byte(`{"rules": [`))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "never a 500")
		body := rec.Body.String()
		assert.Contains(t, body, `Le fichier n&#39;est pas un JSON ou un YAML valide.`)
		assert.Contains(t, body, `failed to deserialize file`)

		rec = s.upload(t, fields, "conf.txt", []byte(`{}`))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), `Choisissez un fichier .json, .yaml ou .yml`)

		rec = s.upload(t, fields, "", nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), `Choisissez le fichier à importer`)

		rec = s.request(t, permWrite, http.MethodPost, "/webui/config/import",
			bytes.NewBufferString(fields.Encode()), "application/x-www-form-urlencoded")
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "not even a multipart form")
		assert.Contains(t, rec.Body.String(), `Choisissez le fichier à importer`)
	})

	t.Run("Given no category", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.upload(t, url.Values{fieldAction: {actionSimulate}}, "conf.json", newRule)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, `<span class="w-field__error" id="cfg-import-cat-h">`)
		assert.Contains(t, body, `Choisissez au moins une catégorie`)
		assert.NotContains(t, body, `id="cfg-import-rules" name="target" value="rules" checked>`, "the categories posted stay")
	})
}

func TestFileName(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.October, 24, 18, 30, 5, 0, time.FixedZone("CEST", 2*3600))
	assert.Equal(t, "waarp-gateway-gw-prod-20261024-183005.json", fileName("gw-prod", formatJSON, now))
	assert.Equal(t, "waarp-gateway-Passerelle-de-recette-20261024-183005.yaml", fileName("Passerelle de recette !", formatYAML, now))
	assert.Equal(t, "waarp-gateway-gateway-20261024-183005.json", fileName("///", formatJSON, now))
}
