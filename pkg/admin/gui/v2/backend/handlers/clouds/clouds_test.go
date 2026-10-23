package clouds

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/rclone/rclone/fs/object"
	"github.com/rclone/rclone/vfs"
	"github.com/rclone/rclone/vfs/vfscommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	_ "code.waarp.fr/apps/gateway/gateway/pkg/fs/backends" // the description of the storages and the library registry
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermAdminRead
	permWrite = model.PermAdminRead | model.PermAdminWrite
	permAll   = model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete

	s3Secret  = "s3-s3cret-value"
	odSecret  = "app-secret-value"
	odToken   = "tok-123-oauth"
	kmsKey    = "arn:aws:kms:eu-west-3:000000000000:key/demo" // an option the storage library masks
	failOpt   = "description"                                 // an "other option" the library knows: the dummy storage refuses its value failValue
	failValue = "refuse-me"
	failCause = "dummy storage refused the configuration"
)

var errDummy = errors.New(failCause)

//nolint:gochecknoglobals // the registry of the product is global: the dummies are registered once
var registerDummies sync.Once

// knownStorageKeys lists every key of the table, canonical keys and aliases,
// sorted.
func knownStorageKeys() []string {
	var keys []string

	for i := range storages {
		keys = append(keys, storages[i].Key)
		keys = append(keys, storages[i].Aliases...)
	}

	slices.Sort(keys)

	return keys
}

// dummies registers, under every key of the table, a storage that never
// reaches the network: it refuses a configuration carrying the "fail"
// option and, as the OneDrive storage does, writes a token in the options.
func dummies() {
	registerDummies.Do(func() {
		for _, key := range knownStorageKeys() {
			fs.Register(key, func(_, _, _ string, opts map[string]string) (fs.FS, error) {
				if opts[failOpt] == failValue {
					return nil, errDummy
				}

				if key == "onedrive" || key == "sharepoint" {
					opts[optToken] = odToken
				}

				return &fs.VFS{VFS: vfs.New(context.Background(), object.MemoryFs, &vfscommon.Options{})}, nil
			})
		}
	})
}

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router

	s3, azfiles, gcs, onedrive *model.CloudInstance
}

func newSuite(t *testing.T) *suite {
	t.Helper()
	dummies()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/clouds").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/clouds/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/clouds/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/clouds/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/clouds/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/clouds/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/clouds/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/clouds/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	return s
}

// fixtures inserts four instances, one per credential family: an S3 bucket
// with a key, a secret and an option the form does not name; an Azure Files
// share recorded under the alias of its type; a Google bucket authenticated
// by the environment; a OneDrive drive with its token. A rule and a server
// name the S3 instance in their paths.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.s3 = &model.CloudInstance{
		Name: "archive-s3", Type: "s3", Key: "AKIAEXAMPLE", Secret: s3Secret,
		Options: map[string]string{
			"bucket": "archive", "region": "eu-west-3", "chunk_size": "8MiB", "acl": "private", "sse_kms_key_id": kmsKey,
		},
	}
	s.azfiles = &model.CloudInstance{
		Name: "share-az", Type: "azurefiles", Key: "acct", Secret: "acct-key",
		Options: map[string]string{"share_name": "docs"},
	}
	s.gcs = &model.CloudInstance{
		Name: "gcs-env", Type: "google",
		Options: map[string]string{"bucket": "b1", "project_number": "42", "env_auth": "true", "bucket_policy_only": "true"},
	}
	s.onedrive = &model.CloudInstance{
		Name: "od-drive", Type: "onedrive", Key: "app-id", Secret: odSecret,
		Options: map[string]string{"tenant": "t-1", "drive_id": "d-1", "drive_type": "personal"},
	}

	for _, cloud := range []*model.CloudInstance{s.s3, s.azfiles, s.gcs, s.onedrive} {
		require.NoError(t, s.db.Insert(cloud).Run())
	}

	require.NoError(t, s.db.Insert(&model.Rule{Name: "pull-archive", IsSend: false, Path: "pull", LocalDir: "archive-s3:in"}).Run())
	require.NoError(t, s.db.Insert(&model.LocalAgent{
		Name: "srv-cloud", Protocol: "sftp", Address: types.Addr("127.0.0.1", 2222),
		RootDir: "archive-s3:root", TmpReceiveDir: t.TempDir(), // the temporary folder must stay local
	}).Run())
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

func (s *suite) cloud(t *testing.T, name string) *model.CloudInstance {
	t.Helper()

	cloud, err := internal.GetCloud(s.db, name)
	require.NoError(t, err)

	return cloud
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

// accepted returns the list of the options the storage library accepts for
// a storage, as the form renders it under the "other options" zone.
func accepted(t *testing.T, body, key string) string {
	t.Helper()

	start := strings.Index(body, `<select class="w-select" id="accepted-`+key+`"`)
	require.NotEqualf(t, -1, start, "no list of accepted options for %s", key)

	end := strings.Index(body[start:], "</select>")
	require.NotEqual(t, -1, end)

	return body[start : start+end]
}

// foldCard returns the opening tag of the foldable card of the form titled
// title, and the summary in its header.
func foldCard(t *testing.T, body, title string) (string, string) {
	t.Helper()

	head := `<span class="w-card__title">` + title + `</span><span class="w-fold__meta">`
	at := strings.Index(body, head)
	require.NotEqualf(t, -1, at, "no foldable card %q", title)

	start := strings.LastIndex(body[:at], "<details ")
	require.NotEqual(t, -1, start)

	rest := body[at+len(head):]

	return body[start : start+strings.Index(body[start:], ">")+1], rest[:strings.Index(rest, "</span>")]
}

func path(cloud *model.CloudInstance, suffix string) string {
	return fmt.Sprintf("/webui/clouds/%d%s", cloud.ID, suffix)
}

func TestList(t *testing.T) {
	t.Parallel()

	t.Run("Given a user with every right", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, "/webui/clouds")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Instances cloud · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/clouds" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.Contains(t, body, `<li><span aria-current="page">Instances cloud</span></li>`)
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")

		// One coloured action, the search with its suggestions, the five type boxes, the count.
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/clouds/new">`)
		assert.Contains(t, body, `<option value="archive-s3"></option><option value="gcs-env"></option><option value="od-drive"></option><option value="share-az"></option>`)
		assert.Equal(t, 5, strings.Count(body, `type="checkbox" name="type"`), "five storage types")
		assert.Contains(t, body, `name="type" value="azfiles"> <span>Azure Files</span>`)
		assert.Contains(t, body, `aria-live="polite">4 instances</span>`)

		// Four sortable headers, sorted by name ascending; the rows by name, each name a link.
		assert.Equal(t, 4, rows(body))
		assert.Contains(t, body, `<th scope="col" aria-sort="ascending"><a href="/webui/clouds?sort=name&amp;dir=desc">Instance cloud</a></th>`)
		assert.Contains(t, body, `<th scope="col"><a href="/webui/clouds?sort=location&amp;dir=asc">Emplacement</a></th>`)
		indexes(t, body, "<tbody>", "archive-s3", "gcs-env", "od-drive", "share-az", "</tbody>")
		assert.Contains(t, body, `<a class="w-cell-title w-mono w-nowrap" href="`+path(s.s3, "")+`">archive-s3</a>`)

		// The type as a label (the stored alias kept in sight), the location, the authentication.
		assert.Contains(t, body, `>archive-s3</a></td><td>Amazon S3</td><td><span class="w-mono">archive</span></td><td>Clé renseignée</td>`)
		assert.Contains(t, body, `>share-az</a></td><td>Azure Files (azurefiles)</td><td><span class="w-mono">docs</span></td><td>Clé renseignée</td>`)
		assert.Contains(t, body, `>gcs-env</a></td><td>Google Cloud Storage (google)</td><td><span class="w-mono">b1</span></td><td>Variables d&#39;environnement</td>`)
		assert.Contains(t, body, `>od-drive</a></td><td>OneDrive</td><td><span class="w-mono">d-1</span></td><td>Clé renseignée</td>`)
		assert.NotContains(t, body, s3Secret)
		assert.NotContains(t, body, "AKIAEXAMPLE", "the key is on the sheet, not in the list")

		// The "⋯" menu: view, edit, delete last.
		indexes(t, body, ">archive-s3</a>", "Voir la fiche", "Modifier l&#39;instance", "Supprimer l&#39;instance", ">gcs-env</a>")
	})

	t.Run("Given a user with the read right only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/clouds").Body.String()
		assert.NotContains(t, body, "w-btn--primary", "no coloured action at all")
		assert.NotContains(t, body, "Modifier l&#39;instance")
		assert.NotContains(t, body, "Supprimer l&#39;instance")
		assert.Contains(t, body, "Voir la fiche")
	})

	t.Run("Given a search, filters and a sort", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, "/webui/clouds?q=ARCH").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, `aria-live="polite">4 instances · 1 affichée</span>`)

		body = s.get(t, permAll, "/webui/clouds?type=azfiles&type=gcs").Body.String()
		assert.Equal(t, 2, rows(body), "the alias azurefiles counts as Azure Files")
		assert.Contains(t, body, `<details class="w-disclose" open>`)
		assert.Contains(t, body, `name="type" value="azfiles" checked>`)
		assert.Contains(t, body, "Filtres · 2")

		body = s.get(t, permAll, "/webui/clouds?type=azurefiles").Body.String()
		assert.Equal(t, 1, rows(body), "an alias in the query filters its storage")

		body = s.get(t, permAll, "/webui/clouds?type=azblob").Body.String()
		assert.Contains(t, body, "Aucune instance ne correspond aux filtres")
		assert.Contains(t, body, `href="/webui/clouds?reset=1">Réinitialiser</a>`)

		body = s.get(t, permAll, "/webui/clouds?sort=auth&dir=asc").Body.String()
		indexes(t, body, "<tbody>", "gcs-env", "archive-s3", "od-drive", "share-az", "</tbody>")

		body = s.get(t, permAll, "/webui/clouds?sort=type&dir=desc").Body.String()
		indexes(t, body, "<tbody>", "od-drive", "gcs-env", "share-az", "archive-s3", "</tbody>")

		body = s.get(t, permAll, "/webui/clouds?q=zzz").Body.String()
		assert.Contains(t, body, "Aucune instance ne correspond à « zzz »")
		assert.Contains(t, body, `href="/webui/clouds?reset=1">Effacer la recherche</a>`)
	})

	t.Run("Given no instance at all", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		body := s.get(t, permWrite, "/webui/clouds").Body.String()
		assert.Contains(t, body, "Aucune instance cloud pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/clouds/new">`)

		body = s.get(t, permRead, "/webui/clouds").Body.String()
		assert.NotContains(t, body, "/webui/clouds/new", "no action without the write right")
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	t.Run("Given the S3 instance, named by a rule and a server", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, path(s.s3, "")+"?ret=q%3Darch")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Instance cloud « archive-s3 » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="/webui/clouds?q=arch">Instances cloud</a>`, "the breadcrumb keeps the list state")
		assert.Contains(t, body, "<dt>Type</dt><dd>Amazon S3<span class=\"w-muted\">Bucket S3 d&#39;Amazon ou d&#39;un service compatible.</span></dd>")
		assert.Contains(t, body, `<code class="w-code">archive-s3:dossier/fichier</code>`)
		assert.Contains(t, body, `<dt>Clé d&#39;accès</dt><dd><code class="w-code">AKIAEXAMPLE</code></dd>`)
		assert.Contains(t, body, "<dt>Secret</dt><dd>Défini</dd>")
		assert.Contains(t, body, "<dt>Identifiants par les variables d&#39;environnement</dt><dd>Non</dd>")
		assert.Contains(t, body, `<dt>Bucket</dt><dd><code class="w-code">archive</code></dd>`)
		assert.Contains(t, body, `<dt>Région</dt><dd><code class="w-code">eu-west-3</code></dd>`)
		assert.Contains(t, body, `<dt>Point d&#39;accès</dt><dd>Valeur par défaut de la passerelle</dd>`)
		assert.Contains(t, body, `<dt>Taille des morceaux envoyés</dt><dd><code class="w-code">8MiB</code></dd>`)
		assert.Contains(t, body, `<dt>Seuil de copie par morceaux</dt><dd><code class="w-code">4.656Gi</code></dd>`,
			"the default of the storage library, which the product applies (#701)")
		assert.Contains(t, body, `<dt><code class="w-code">acl</code></dt><dd><code class="w-code">private</code></dd>`, "an option the form does not name")
		assert.Contains(t, body, `<dt><code class="w-code">sse_kms_key_id</code></dt><dd><span class="w-muted">Masquée</span></dd>`,
			"an option the storage library masks: named, never rendered")
		assert.NotContains(t, body, kmsKey)
		assert.Contains(t, body, "<dt>Règles de transfert</dt>")
		assert.Contains(t, body, `<code class="w-code">pull-archive</code>`)
		assert.Contains(t, body, "<dt>Serveurs locaux</dt>")
		assert.Contains(t, body, `href="/webui/servers/`)
		assert.Contains(t, body, ">srv-cloud</a>")
		assert.NotContains(t, body, "Transferts</dt>", "no transfer names it")
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="`+path(s.s3, "/edit")+`?ret=q%3Darch">`)
		assert.NotContains(t, body, s3Secret)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given the Google instance authenticated by the environment, in English", func(t *testing.T) {
		t.Parallel()

		body := s.do(t, permRead, "en", http.MethodGet, path(s.gcs, ""), nil).Body.String()
		assert.Contains(t, body, "<title>Cloud instance “gcs-env” · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "<dt>Type</dt><dd>Google Cloud Storage")
		assert.Contains(t, body, `<dt>Credentials file</dt><dd><span class="w-muted">Not set</span></dd>`)
		assert.Contains(t, body, `<dt>Secret</dt><dd><span class="w-muted">Not set</span></dd>`)
		assert.Contains(t, body, "<dt>Credentials from the environment variables</dt><dd>Yes</dd>")
		assert.Contains(t, body, "<dt>The bucket enforces uniform access</dt><dd>Yes</dd>")
		assert.Contains(t, body, "No object names this instance yet.")
		assert.NotContains(t, body, "Edit the instance", "no edit without the write right")
		assert.NotContains(t, body, "popovertarget", "no empty menu")
	})

	t.Run("Given the OneDrive instance with its token", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, path(s.onedrive, "")).Body.String()
		assert.Contains(t, body, `<dt>Identifiant d&#39;application</dt><dd><code class="w-code">app-id</code></dd>`)
		assert.Contains(t, body, "<dt>Type de lecteur</dt><dd>Personnel (personal)</dd>")
		assert.Contains(t, body, `<dt>Région</dt><dd>global</dd>`, "the product default")
		assert.NotContains(t, body, odToken, "the token the storage wrote is a secret")
		assert.NotContains(t, body, odSecret)
		assert.NotContains(t, body, "Autres options", "the token is not an other option")
	})

	t.Run("Given a OneDrive instance without drive type", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		bare := &model.CloudInstance{Name: "od-bare", Type: "onedrive", Key: "app", Secret: "sec", Options: map[string]string{"tenant": "t", "drive_id": "d"}}
		require.NoError(t, s.db.Insert(bare).Run())

		body := s.get(t, permRead, path(bare, "")).Body.String()
		assert.Contains(t, body, `<dt>Type de lecteur</dt><dd>Non défini</dd>`, "a required option the instance lacks")

		body = s.get(t, permWrite, path(bare, "/edit")).Body.String()
		assert.Contains(t, body, `<option value="business" selected>Professionnel (business)</option>`, "the form proposes the usual drive type")
	})

	t.Run("Given the alias type", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, path(s.azfiles, "")).Body.String()
		assert.Contains(t, body, "<dt>Type</dt><dd>Azure Files (azurefiles)")
		assert.Contains(t, body, `<dt>Partage</dt><dd><code class="w-code">docs</code></dd>`)
	})

	t.Run("Given an unknown instance", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permRead, "/webui/clouds/999?ret=page%3D2")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Cette instance n&#39;existe pas ou a été supprimée.")
		assert.Contains(t, rec.Body.String(), `href="/webui/clouds?page=2"`, "the way back keeps the list state")
	})
}

func TestNewCloud(t *testing.T) {
	t.Parallel()

	t.Run("Given the empty form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.get(t, permWrite, "/webui/clouds/new")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Nouvelle instance cloud · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<form class="w-form" method="post" action="/webui/clouds/new" novalidate>`)
		assert.Contains(t, body, `<select class="w-select" id="type" name="type" required data-switch aria-describedby="type-h">`)
		assert.Contains(t, body, `<option value="" selected>Choisissez un type</option>`)
		assert.Contains(t, body, `<option value="s3">Amazon S3</option>`)
		assert.Contains(t, body, `<option value="onedrive">OneDrive</option>`)
		assert.Contains(t, body, `id="access_key" name="access_key" value="" autocomplete="off"`)
		assert.Contains(t, body, `id="access_secret" name="access_secret" type="password" autocomplete="off"`)
		assert.Contains(t, body, `<fieldset class="w-fieldset" data-case="s3 azblob azfiles gcs">`, "the environment credentials, for the storages that read them")
		assert.Equal(t, 5, strings.Count(body, `<details class="w-card w-card--fold" data-case="`), "one foldable card per storage")

		tag, summary := foldCard(t, body, "Accès")
		assert.Equal(t, `<details class="w-card w-card--fold" open>`, tag, "the access card is open: it holds what the gateway needs")
		assert.Equal(t, "Aucune", summary)
		tag, summary = foldCard(t, body, "Options Amazon S3")
		assert.Equal(t, `<details class="w-card w-card--fold" data-case="s3" open>`, tag, "a card with a required option opens")
		assert.Equal(t, "À renseigner", summary)
		tag, summary = foldCard(t, body, "Options Azure Blob")
		assert.Equal(t, `<details class="w-card w-card--fold" data-case="azblob">`, tag, "a card of defaults folds")
		assert.Equal(t, "Options par défaut de la passerelle", summary)
		assert.Contains(t, body, `id="s3_endpoint" name="s3_endpoint" value="" placeholder="s3.eu-west-3.amazonaws.com"`, "an example in the empty field")
		assert.Contains(t, body, `id="azblob_list_chunk" name="azblob_list_chunk" value="" placeholder="5000"`, "the library default in the empty field")
		assert.Contains(t, body, `id="onedrive_drive_id" name="onedrive_drive_id" value="" required placeholder="b!GoXL`)
		assert.Contains(t, body, "Vide : https://&lt;compte&gt;.blob.core.windows.net, le compte étant la clé.")
		assert.Contains(t, body, "Vide : https://&lt;compte&gt;.file.core.windows.net, le compte étant la clé.")
		assert.Contains(t, body, `id="s3_bucket" name="s3_bucket" value="" required`)
		assert.Contains(t, body, `id="gcs_project_number" name="gcs_project_number" value="" required`)
		assert.Contains(t, body, `id="onedrive_drive_type" name="onedrive_drive_type"`)
		assert.Contains(t, body, `<option value="business" selected>Professionnel (business)</option>`)
		assert.Contains(t, body, `name="gcs_bucket_policy_only" value="1"`)
		assert.Contains(t, body, `Vide : 5Mi ; une taille avec son unité, par exemple 8MiB.`, "the library default in the hint")
		assert.Contains(t, body, `<select class="w-select" id="onedrive_drive_type" name="onedrive_drive_type" required>`,
			"the library has no default drive type: the form always writes one")
		tag, summary = foldCard(t, body, "Autres options")
		assert.Equal(t, `<details class="w-card w-card--fold">`, tag, "the other options fold when empty")
		assert.Equal(t, "Aucune", summary)
		assert.Contains(t, body, `<textarea class="w-textarea w-mono" id="other" name="other"`)

		list := accepted(t, body, "s3")
		assert.Contains(t, list, `data-into="other" data-suffix="="`, "the chosen option written in the zone")
		assert.NotContains(t, list, `name=`, "the list holds no value of the form")
		assert.Contains(t, list, `<option value="">Options acceptées par Amazon S3 (`)
		assert.Contains(t, list, `<option value="storage_class" data-label="storage_class" data-text="`)
		assert.Contains(t, list, "valeurs : private, public-read")
		assert.Equal(t, strings.Count(list, `<option value="`)-1, strings.Count(list, "data-label="), "one choice per option")
		assert.Contains(t, body, `src="/webui/static/gui/combobox.js"`, "the list to type in of the kit")
		assert.NotContains(t, body, "<table", "no table in a card")

		assert.Contains(t, list, `<option value="list_chunk"`, "the gateway sets nothing itself (#701)")

		for _, key := range []string{"bucket", "region", "env_auth", "access_key_id", "session_token", "sse_kms_key_id", "memory_pool_flush_time", "ibm_api_key"} {
			assert.NotContainsf(t, list, `<option value="`+key+`"`,
				"%s is not offered: a field, a credential, a masked option, obsolete or of another provider", key)
		}

		assert.Contains(t, accepted(t, body, "onedrive"), `<option value="list_chunk"`, "the list of each storage")
		assert.NotContains(t, body, "rclone", "the interface names no third-party product")
		assert.Contains(t, body, `>Valider</button>`)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a valid S3 form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/clouds/new?ret=sort%3Dtype", url.Values{
			"name": {" backup-s3 "}, "type": {"s3"}, "access_key": {"AKIA2"}, "access_secret": {"shh"},
			"s3_bucket": {"backups"}, "s3_region": {"eu-west-1"}, "s3_chunk_size": {""},
			"gcs_bucket": {"ignored"}, "onedrive_drive_type": {"personal"},
			"other": {"acl=private\r\n\r\nstorage_class = GLACIER"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, "/webui/clouds?dir=asc&done=created&name=backup-s3&sort=type", rec.Header().Get("Location"))

		cloud := s.cloud(t, "backup-s3")
		assert.Equal(t, "s3", cloud.Type)
		assert.Equal(t, "AKIA2", cloud.Key)
		assert.Equal(t, "shh", cloud.Secret)
		assert.Equal(t, model.Map[string]{
			"bucket": "backups", "region": "eu-west-1", "acl": "private", "storage_class": "GLACIER",
		}, cloud.Options, "the fields of the chosen storage, the other options")

		body := s.get(t, permWrite, "/webui/clouds?done=created&name=backup-s3").Body.String()
		assert.Contains(t, body, "Instance cloud « backup-s3 » créée")
	})

	t.Run("Given other options the storage refuses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/clouds/new", url.Values{
			"name": {"refused-others"}, "type": {"s3"}, "s3_bucket": {"b"},
			"other": {"acl=private\nstorge_class=X\nsession_token=abc\nlist_chunk=50\nbucket=x\nenv_auth=true\naccess_key_id=k"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		indexes(t, body,
			"Amazon S3 ne connaît pas l&#39;option « storge_class » : elle serait sans effet.",
			"« session_token » est une option sensible : elle serait conservée et affichée en clair",
			"« bucket » se règle dans son champ, plus haut.",
			"« env_auth » se règle dans son champ, plus haut.",
			"« access_key_id » est une option sensible",
		)
		assert.NotContains(t, body, "« acl »", "an option the library accepts")
		assert.NotContains(t, body, "« list_chunk »", "an option the library accepts, the gateway setting nothing itself")
		assert.Contains(t, body, "storge_class=X\nsession_token=abc", "the zone comes back as entered")

		tag, _ := foldCard(t, body, "Autres options")
		assert.Equal(t, `<details class="w-card w-card--fold" open>`, tag, "the other options open on their error")

		_, err := internal.GetCloud(s.db, "refused-others")
		require.Error(t, err, "nothing saved")
	})

	t.Run("Given a Google form authenticated by the environment", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/clouds/new", url.Values{
			"name": {"gcs-prod"}, "type": {"gcs"}, "env_auth": {"1"},
			"gcs_bucket": {"prod"}, "gcs_project_number": {"7"}, "gcs_bucket_policy_only": {"1"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		cloud := s.cloud(t, "gcs-prod")
		assert.Equal(t, "gcs", cloud.Type)
		assert.Empty(t, cloud.Key)
		assert.Empty(t, cloud.Secret)
		assert.Equal(t, model.Map[string]{"bucket": "prod", "project_number": "7", "env_auth": "true", "bucket_policy_only": "true"}, cloud.Options)
	})

	t.Run("Given a OneDrive form left at the product defaults", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)

		rec := s.post(t, permWrite, "/webui/clouds/new", url.Values{
			"name": {"od-new"}, "type": {"onedrive"}, "access_key": {"app"}, "access_secret": {"sec"}, "env_auth": {"1"},
			"onedrive_tenant": {"t"}, "onedrive_drive_id": {"d"}, "onedrive_drive_type": {"business"}, "onedrive_region": {"global"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		cloud := s.cloud(t, "od-new")
		assert.Equal(t, model.Map[string]{"tenant": "t", "drive_id": "d", "drive_type": "business", "token": odToken}, cloud.Options,
			"the drive type is always written (the library has no default); a region left at the library default is not stored; "+
				"the environment switch does not apply to OneDrive; the storage wrote its token")
	})

	t.Run("Given an invalid form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/clouds/new", url.Values{"name": {""}, "type": {""}, "other": {"no-equal-sign"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<div class="w-banner w-banner--ko" role="alert">`)
		assert.Contains(t, body, "L&#39;instance n&#39;a pas été enregistrée.")
		assert.Contains(t, body, "Ce champ est obligatoire.")
		assert.Contains(t, body, "Choisissez un type.")
		assert.Contains(t, body, "Indiquez une option par ligne, sous la forme clé=valeur.")
		tag, _ := foldCard(t, body, "Autres options")
		assert.Equal(t, `<details class="w-card w-card--fold" open>`, tag, "the other options open on their error")

		rec = s.post(t, permWrite, "/webui/clouds/new", url.Values{"name": {"file"}, "type": {"azfiles"}, "azfiles_share_name": {""}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body = rec.Body.String()
		assert.Contains(t, body, "« file » est réservé au disque local.")
		assert.Contains(t, body, `id="azfiles_share_name" name="azfiles_share_name" value="" required placeholder="archive" aria-invalid="true"`)
		tag, _ = foldCard(t, body, "Options Azure Files")
		assert.Equal(t, `<details class="w-card w-card--fold" data-case="azfiles" open>`, tag, "a card that carries an error opens")
		assert.Contains(t, body, `<option value="azfiles" selected>Azure Files</option>`, "the type is kept")

		rec = s.post(t, permWrite, "/webui/clouds/new", url.Values{"name": {"archive-s3"}, "type": {"s3"}, "s3_bucket": {"b"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Une autre instance porte déjà ce nom.")

		// The storage refuses the configuration: its cause in the banner.
		rec = s.post(t, permWrite, "/webui/clouds/new", url.Values{
			"name": {"refused"}, "type": {"s3"}, "s3_bucket": {"b"}, "other": {failOpt + "=" + failValue},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Le stockage a refusé cette configuration : "+failCause)

		count, err := s.db.Count(&model.CloudInstance{}).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 4, count, "nothing created")
	})
}

func TestEditCloud(t *testing.T) {
	t.Parallel()

	t.Run("Given the form of the S3 instance", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, path(s.s3, "/edit")+"?ret=page%3D2").Body.String()
		assert.Contains(t, body, "<title>Modifier « archive-s3 » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `id="name" name="name" value="archive-s3"`)
		assert.Contains(t, body, `<option value="s3" selected>Amazon S3</option>`)
		assert.Contains(t, body, `id="access_key" name="access_key" value="AKIAEXAMPLE"`)
		assert.Contains(t, body, `id="access_secret" name="access_secret" type="password"`)
		assert.Contains(t, body, "Vide : inchangé.")
		assert.NotContains(t, body, s3Secret, "the secret is never rendered back")
		assert.Contains(t, body, `id="s3_bucket" name="s3_bucket" value="archive"`)
		assert.Contains(t, body, `id="s3_chunk_size" name="s3_chunk_size" value="8MiB"`)
		tag, summary := foldCard(t, body, "Autres options")
		assert.Equal(t, `<details class="w-card w-card--fold" open>`, tag, "the other options open with their value")
		assert.Equal(t, "Renseignées", summary)
		tag, summary = foldCard(t, body, "Options Amazon S3")
		assert.Equal(t, `<details class="w-card w-card--fold" data-case="s3" open>`, tag)
		assert.Equal(t, "3 options renseignées", summary, "bucket, region and a chunk size other than the default")
		_, summary = foldCard(t, body, "Accès")
		assert.Equal(t, "Clé renseignée", summary, "the key, the secret kept")
		assert.Contains(t, body, ">acl=private</textarea>")
		assert.Contains(t, body, `href="`+path(s.s3, "")+`?ret=page%3D2">`, "cancel goes back to the instance")
	})

	t.Run("Given a saved edition without secret", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, path(s.s3, "/edit")+"?ret=q%3Darch", url.Values{
			"name": {"archive-s3"}, "type": {"s3"}, "access_key": {"AKIAEXAMPLE"}, "access_secret": {""},
			"s3_bucket": {"archive-2"}, "s3_region": {""}, "s3_chunk_size": {"8MiB"}, "other": {"acl=private"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, path(s.s3, "")+"?done=updated&name=archive-s3&ret=q%3Darch", rec.Header().Get("Location"),
			"an edited instance comes back to its sheet")

		saved := s.cloud(t, "archive-s3")
		assert.Equal(t, s3Secret, saved.Secret, "an empty secret keeps the current one")
		assert.Equal(t, model.Map[string]{
			"bucket": "archive-2", "chunk_size": "8MiB", "acl": "private", "sse_kms_key_id": kmsKey,
		}, saved.Options, "a masked option set by the REST API stays as it is")
	})

	t.Run("Given an untouched edition of the alias instance", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, path(s.azfiles, "/edit"), url.Values{
			"name": {"share-az"}, "type": {"azfiles"}, "access_key": {"acct"}, "azfiles_share_name": {"docs"}, "azfiles_endpoint": {""},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		saved := s.cloud(t, "share-az")
		assert.Equal(t, "azurefiles", saved.Type, "the stored alias stays while the storage does not change")
		assert.Equal(t, "acct-key", saved.Secret)
		assert.Equal(t, model.Map[string]{"share_name": "docs"}, saved.Options, "the same configuration")
	})

	t.Run("Given a change of type and of secret", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, path(s.azfiles, "/edit"), url.Values{
			"name": {"share-az"}, "type": {"gcs"}, "access_key": {""}, "access_secret": {"{json}"},
			"gcs_bucket": {"moved"}, "gcs_project_number": {"9"}, "azfiles_share_name": {"docs"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		saved := s.cloud(t, "share-az")
		assert.Equal(t, "gcs", saved.Type, "the canonical key of the new storage")
		assert.Equal(t, "{json}", saved.Secret)
		assert.Equal(t, model.Map[string]{"bucket": "moved", "project_number": "9"}, saved.Options, "the options of the previous storage are gone")
	})

	t.Run("Given a name already taken", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, path(s.gcs, "/edit"), url.Values{"name": {"archive-s3"}, "type": {"gcs"}, "gcs_bucket": {"b"}, "gcs_project_number": {"1"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Une autre instance porte déjà ce nom.")
		assert.Equal(t, "gcs-env", s.cloud(t, "gcs-env").Name, "nothing changed")
	})
}

func TestDeleteCloud(t *testing.T) {
	t.Parallel()

	t.Run("Given an instance named by a rule and a server", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, path(s.s3, "/delete")+"?ret=q%3Darch").Body.String()
		assert.Contains(t, body, "<title>Supprimer l&#39;instance cloud « archive-s3 » ? · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Les chemins qui la nomment ne fonctionneront plus")
		assert.Contains(t, body, `<code class="w-code">pull-archive</code>`)
		assert.Contains(t, body, ">srv-cloud</a>")
		assert.Contains(t, body, `action="`+path(s.s3, "/delete")+`?ret=q%3Darch"`)
		assert.Contains(t, body, ">Supprimer</button>")

		rec := s.post(t, permAll, path(s.s3, "/delete")+"?ret=q%3Darch", nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/webui/clouds?done=deleted&name=archive-s3&q=arch", rec.Header().Get("Location"))

		_, err := internal.GetCloudByID(s.db, s.s3.ID)
		assert.True(t, database.IsNotFound(err))
	})

	t.Run("Given an instance nothing names", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permAll, path(s.gcs, "/delete")).Body.String()
		assert.Contains(t, body, "Aucun objet ne nomme cette instance")
		assert.NotContains(t, body, "Utilisée par")
		assert.Contains(t, body, ">Supprimer</button>")
	})

	t.Run("Given an unknown instance", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		assert.Equal(t, http.StatusNotFound, s.post(t, permAll, "/webui/clouds/999/delete", nil).Code)
	})
}

// No secret ever reaches a page: neither the secret, nor the token the
// storage wrote, in the list, the sheet, the forms, refused or not.
func TestNoSecret(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	for _, target := range []string{
		"/webui/clouds", path(s.s3, ""), path(s.s3, "/edit"), path(s.s3, "/delete"),
		path(s.onedrive, ""), path(s.onedrive, "/edit"),
	} {
		body := s.get(t, permAll, target).Body.String()
		assert.NotContains(t, body, s3Secret, target)
		assert.NotContains(t, body, odSecret, target)
		assert.NotContains(t, body, odToken, target)
		assert.NotContains(t, body, kmsKey, target)
	}

	body := s.post(t, permWrite, path(s.onedrive, "/edit"), url.Values{"name": {""}, "type": {"onedrive"}, "access_secret": {"typed-secret"}}).Body.String()
	assert.NotContains(t, body, "typed-secret")
	assert.NotContains(t, body, odToken)
	assert.NotContains(t, body, "token=", "the token is not an other option")
}

// The table of the storages names every key the product registers, once,
// and the options it marks required are the ones the product checks.
func TestStorageTable(t *testing.T) {
	t.Parallel()

	keys := knownStorageKeys()
	assert.Equal(t, []string{"aws", "azblob", "azfiles", "azureblob", "azurefiles", "gcs", "google", "onedrive", "s3", "sharepoint"}, keys)

	for _, key := range keys {
		s, ok := storageOf(key)
		require.True(t, ok, key)
		assert.NotEmpty(t, s.Creds, key)
	}

	for i := range storages {
		for _, o := range storages[i].resolved() {
			if o.Kind == kindText {
				assert.NotEmptyf(t, o.placeholder(), "%s %s: a text option shows an example or its default", storages[i].Key, o.Key)
			}

			if o.Initial != "" {
				assert.Containsf(t, o.Choices, o.Initial, "%s %s: the proposed choice is a choice", storages[i].Key, o.Key)
			}
		}
	}

	// The library of every storage is known: it knows the options the form
	// names (the bucket is the product's own), and its help texts name no
	// third-party product once read.
	for i := range storages {
		store := &storages[i]

		lib, known := libraryOf(store)
		require.Truef(t, known, "%s: the library is known", store.Key)
		assert.NotEmptyf(t, lib.listed(), "%s: options are offered", store.Key)

		for _, o := range store.Options {
			if o.Key != "bucket" {
				assert.Truef(t, lib.knows(o.Key), "%s: the library knows %s", store.Key, o.Key)
			}
		}

		for _, o := range lib.listed() {
			assert.NotContainsf(t, strings.ToLower(o.Help), "rclone", "%s %s", store.Key, o.Key)
		}
	}

	_, ok := storageOf("ftp")
	assert.False(t, ok)

	assert.True(t, isSecret("token"))
	assert.True(t, isSecret("secret_access_key"))
	assert.False(t, isSecret("bucket"))
	assert.Equal(t, "env", authOf("", "", map[string]string{"env_auth": "TRUE"}))
	assert.Equal(t, "key", authOf("k", "", nil))
	assert.Equal(t, "none", authOf("", "", nil))
}

func TestParseOthers(t *testing.T) {
	t.Parallel()

	others, order, err := parseOthers("c=\na=1\n\n b = two words \na=2")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "2", "b": "two words", "c": ""}, others)
	assert.Equal(t, []string{"c", "a", "b"}, order, "the keys in the order of their lines, once")

	_, _, err = parseOthers("novalue")
	require.Error(t, err)

	_, _, err = parseOthers("=1")
	require.Error(t, err)
}
