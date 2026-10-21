package overrides

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermAdminRead
	permWrite = model.PermAdminRead | model.PermAdminWrite
	permAll   = model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete

	nodeID = "noeud-1"
)

// suite is a database, the configuration of a node with its own override
// file, the handlers on them and the router that names their routes as the
// web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router
	file   string
}

func newSuite(t *testing.T, cluster bool) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	s.db.Config.Admin.Host, s.db.Config.Admin.Port = "127.0.0.1", 8080

	if cluster {
		s.file = filepath.Join(t.TempDir(), nodeID+".ini")
		s.db.Config.Overrides = conf.NewOverride(s.file)
		s.db.Config.NodeID = nodeID
	}

	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/overrides").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/overrides/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/overrides/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/overrides/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/overrides/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/overrides/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/overrides/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	return s
}

// fixtures: a local server and two partners in the shared configuration, and
// three rules: the host and the port of the server, the host alone of the
// partners (the server keeps its own rule), the host of the administration.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	require.NoError(t, s.db.Insert(&model.LocalAgent{
		Name: "srv-public", Protocol: "sftp", Address: types.Addr("localhost", 2222),
	}).Run())

	for i, name := range []string{"part-a", "part-b"} {
		require.NoError(t, s.db.Insert(&model.RemoteAgent{
			Name: name, Protocol: "sftp", Address: types.Addr("localhost", uint16(6666+i)),
		}).Run())
	}

	for target, real := range map[string]string{
		"localhost:2222": "10.0.0.5:2222",
		"localhost":      "10.0.0.6",
		"127.0.0.1":      "10.0.0.7",
	} {
		require.NoError(t, s.db.Config.Overrides.AddIndirection(target, real))
	}
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

func page(action, target string) string {
	return "/webui/overrides/" + action + "?" + url.Values{targetParam: {target}}.Encode()
}

// redirected asserts a 303 to the list and returns the query it carries.
func redirected(t *testing.T, rec *httptest.ResponseRecorder) url.Values {
	t.Helper()

	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

	u, err := url.Parse(rec.Header().Get("Location"))
	require.NoError(t, err)
	assert.Equal(t, "/webui/overrides", u.Path)

	return u.Query()
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

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		rec := s.get(t, permAll, "/webui/overrides")
		require.Equal(t, http.StatusOK, rec.Code)
		indexes(t, rec.Body.String(),
			"Surcharges de configuration</h1>", "Nœud", nodeID, `href="/webui/overrides/new"`, "Ajouter une adresse",
			"ces adresses remplacent celles de la configuration commune",
			">127.0.0.1</td>", ">10.0.0.7</td>", "Hôte seul, port conservé", "Administration",
			">localhost</td>", ">10.0.0.6</td>", "Hôte seul, port conservé", "2 partenaires",
			">localhost:2222</td>", ">10.0.0.5:2222</td>", "Hôte et port",
			`href="/webui/servers/1"`, "Serveur « srv-public »",
			"l&#39;API et l&#39;interface web écouteront sur l&#39;adresse de ce nœud")
		assert.Contains(t, rec.Body.String(), "Modifier la règle")
		assert.Contains(t, rec.Body.String(), "Supprimer la règle")
	})

	t.Run("Sorted descending", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		indexes(t, s.get(t, permAll, "/webui/overrides?sort=target&dir=desc").Body.String(),
			">localhost:2222</td>", ">localhost</td>", ">127.0.0.1</td>")
	})

	t.Run("In reading only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/overrides").Body.String()
		assert.NotContains(t, body, "Ajouter une adresse")
		assert.NotContains(t, body, "Modifier la règle")
		assert.NotContains(t, body, "Supprimer la règle")
	})

	t.Run("Without rule", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)

		body := s.get(t, permAll, "/webui/overrides").Body.String()
		assert.Contains(t, body, "Aucune adresse remplacée sur ce nœud")
		assert.Contains(t, body, "Ajouter une adresse")
	})

	t.Run("On an instance started without an instance name", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, false)

		body := s.get(t, permAll, "/webui/overrides").Body.String()
		assert.Contains(t, body, "Cette instance ne fonctionne pas en grappe")
		assert.NotContains(t, body, "Ajouter une adresse")

		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, "/webui/overrides/new").Code)
		assert.Equal(t, http.StatusNotFound, s.post(t, "/webui/overrides/new", url.Values{}).Code)
		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, page("edit", "localhost")).Code)
	})

	t.Run("The notification says when it applies, and warns for the administration", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		body := s.get(t, permAll, "/webui/overrides?done=created&name=127.0.0.1").Body.String()
		indexes(t, body, "Adresse « 127.0.0.1 » remplacée", "Prise en compte : au prochain démarrage",
			"l&#39;API et l&#39;interface web écouteront")
	})
}

func TestCreate(t *testing.T) {
	t.Parallel()

	t.Run("The form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)

		indexes(t, s.get(t, permWrite, "/webui/overrides/new").Body.String(),
			"Nouvelle adresse remplacée", `id="configured"`, `placeholder="lb.example.org"`,
			`id="replacement"`, `placeholder="10.0.0.5"`, "Prise en compte :")
	})

	t.Run("Saved in the file of the node", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)

		q := redirected(t, s.post(t, "/webui/overrides/new", url.Values{
			fieldConfigured: {"lb.example.org:2222"}, fieldReplacement: {"10.0.0.5:22222"},
		}))
		assert.Equal(t, doneCreated, q.Get(doneParam))
		assert.Equal(t, "lb.example.org:2222", q.Get(nameParam))
		assert.Equal(t, "10.0.0.5:22222", s.db.Config.Overrides.GetIndirection("lb.example.org:2222"))

		content, err := os.ReadFile(s.file)
		require.NoError(t, err)
		assert.Contains(t, string(content), "IndirectAddress = lb.example.org:2222 -> 10.0.0.5:22222")
	})

	t.Run("IPv6 addresses in brackets", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)

		redirected(t, s.post(t, "/webui/overrides/new", url.Values{
			fieldConfigured: {"[2001:db8::1]:443"}, fieldReplacement: {"[::1]:8443"},
		}))
		assert.Equal(t, "[::1]:8443", s.db.Config.Overrides.GetIndirection("[2001:db8::1]:443"))
	})

	t.Run("Refused by the screen", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		for _, tc := range []struct {
			name                    string
			configured, replacement string
			want                    string
		}{
			{"nothing", "", "", "Ce champ est obligatoire."},
			{"the separator of the file", "a->b", "10.0.0.9", "Indiquez un nom d&#39;hôte ou une adresse IP"},
			{"a space", "lb example.org", "10.0.0.9", "Indiquez un nom d&#39;hôte ou une adresse IP"},
			{"an IPv6 address without brackets", "2001:db8::1", "10.0.0.9", "Indiquez un nom d&#39;hôte ou une adresse IP"},
			{"port zero", "lb.example.org:0", "10.0.0.9:22", "Indiquez un port de 1 à 65535."},
			{"different forms", "lb.example.org", "10.0.0.9:22", "Les deux adresses doivent avoir la même forme"},
			{"an address already replaced", "localhost", "10.0.0.9", "Cette adresse est déjà remplacée : modifiez sa règle."},
		} {
			rec := s.post(t, "/webui/overrides/new", url.Values{
				fieldConfigured: {tc.configured}, fieldReplacement: {tc.replacement},
			})
			require.Equalf(t, http.StatusUnprocessableEntity, rec.Code, "%s", tc.name)
			assert.Containsf(t, rec.Body.String(), "La règle n&#39;a pas été enregistrée.", "%s", tc.name)
			assert.Containsf(t, rec.Body.String(), tc.want, "%s", tc.name)
		}

		assert.Equal(t, "10.0.0.6", s.db.Config.Overrides.GetIndirection("localhost"), "never overwritten")
		assert.Empty(t, s.db.Config.Overrides.GetIndirection("a->b"))

		content, err := os.ReadFile(s.file)
		require.NoError(t, err)
		assert.NotContains(t, string(content), "a->b", "the file of the node stays readable")
	})
}

func TestEdit(t *testing.T) {
	t.Parallel()

	t.Run("The form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		body := s.get(t, permWrite, page("edit", "127.0.0.1")).Body.String()
		indexes(t, body, "Modifier l&#39;adresse remplacée « 127.0.0.1 »", "<code class=\"w-code\">127.0.0.1</code>",
			"Administration", "l&#39;API et l&#39;interface web écouteront", `id="replacement"`, `value="10.0.0.7"`)
		assert.NotContains(t, body, `id="configured"`, "the configured address is the key of the rule")
	})

	t.Run("Saved", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		q := redirected(t, s.post(t, page("edit", "localhost"), url.Values{fieldReplacement: {"10.0.0.16"}}))
		assert.Equal(t, doneUpdated, q.Get(doneParam))
		assert.Equal(t, "10.0.0.16", s.db.Config.Overrides.GetIndirection("localhost"))
	})

	t.Run("Refused: a different form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)
		s.fixtures(t)

		rec := s.post(t, page("edit", "localhost"), url.Values{fieldReplacement: {"10.0.0.16:22"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Les deux adresses doivent avoir la même forme")
		assert.Equal(t, "10.0.0.6", s.db.Config.Overrides.GetIndirection("localhost"))
	})

	t.Run("A missing rule", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t, true)

		rec := s.get(t, permWrite, page("edit", "absent.example.org"))
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Règle introuvable")
	})
}

func TestDelete(t *testing.T) {
	t.Parallel()

	s := newSuite(t, true)
	s.fixtures(t)

	indexes(t, s.get(t, permAll, page("delete", "localhost")).Body.String(),
		"Supprimer la règle « localhost » ?", "l&#39;adresse configurée sera de nouveau employée telle quelle",
		"10.0.0.6", "Hôte seul, port conservé", "2 partenaires", ">Supprimer</button>")

	q := redirected(t, s.post(t, page("delete", "localhost"), url.Values{}))
	assert.Equal(t, doneDeleted, q.Get(doneParam))
	assert.Equal(t, "localhost", q.Get(nameParam))
	assert.Empty(t, s.db.Config.Overrides.GetIndirection("localhost"))

	content, err := os.ReadFile(s.file)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "localhost -> ")

	assert.Equal(t, http.StatusNotFound, s.post(t, page("delete", "localhost"), url.Values{}).Code)
}

func TestCheckAddress(t *testing.T) {
	t.Parallel()

	for addr, want := range map[string]struct {
		withPort bool
		err      string
	}{
		"lb.example.org":        {false, ""},
		"lb.example.org:2222":   {true, ""},
		"node_1.internal":       {false, ""},
		"10.0.0.5":              {false, ""},
		"10.0.0.5:22":           {true, ""},
		"[::1]":                 {false, ""},
		"[::1]:8080":            {true, ""},
		"":                      {false, "required"},
		"a->b":                  {false, "address"},
		"lb.example.org:65536":  {true, "port"},
		"lb.example.org:http":   {true, "port"},
		"-lb.example.org":       {false, "address"},
		"lb..example.org":       {false, "address"},
		"2001:db8::1":           {false, "address"},
		"[lb.example.org]:2222": {true, "address"},
	} {
		withPort, err := checkAddress(addr)
		assert.Equalf(t, want.err, err, "%q", addr)

		if want.err == "" {
			assert.Equalf(t, want.withPort, withPort, "%q", addr)
		}
	}
}
