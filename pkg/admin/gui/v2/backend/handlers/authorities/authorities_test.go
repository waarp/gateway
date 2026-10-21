package authorities

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"mime/multipart"
	"net"
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
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	permRead  = model.PermAdminRead
	permWrite = model.PermAdminRead | model.PermAdminWrite
	permAll   = model.PermAdminRead | model.PermAdminWrite | model.PermAdminDelete

	partnerIP = "127.0.0.2"
)

// ca is a certificate authority made for the tests: its certificate in PEM
// and its key, to sign other certificates.
type ca struct {
	PEM  string
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

func certPEM(der []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
}

// newCA makes a certificate authority valid from an hour ago until the given
// time.
func newCA(t *testing.T, name string, until time.Time) *ca {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              until,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)

	return &ca{PEM: certPEM(der), cert: cert, key: key}
}

// leaf makes a server certificate for an IP address, signed by the authority.
func (c *ca) leaf(t *testing.T, ip string) string {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "partner " + ip},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP(ip)},
	}

	der, err := x509.CreateCertificate(rand.Reader, tmpl, c.cert, &key.PublicKey, c.key)
	require.NoError(t, err)

	return certPEM(der)
}

// sshKey makes an SSH public key line.
func sshKey(t *testing.T) (string, ssh.PublicKey) {
	t.Helper()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	key, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)

	return strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key))) + " ca@example", key
}

// suite is a database, the handlers on it and the router that names their
// routes as the web interface does.
type suite struct {
	db     *database.DB
	router *mux.Router

	valid, soon      *ca
	sshLine          string
	sshPub           ssh.PublicKey
	tls, tlsSoon, sh *model.Authority
	partner          *model.RemoteAgent
	signedCert       *model.Credential
}

func newSuite(t *testing.T) *suite {
	t.Helper()

	s := &suite{db: dbtest.TestDatabase(t), router: mux.NewRouter()}
	h := New(s.db, testhelpers.GetTestLogger(t))

	r := s.router.PathPrefix(constants.WebuiPrefix).Subrouter()
	r.Path("/authorities").Methods(http.MethodGet).HandlerFunc(h.List)
	r.Path("/authorities/new").Methods(http.MethodGet).HandlerFunc(h.NewForm)
	r.Path("/authorities/new").Methods(http.MethodPost).HandlerFunc(h.Create)
	r.Path("/authorities/{id:[0-9]+}").Methods(http.MethodGet).HandlerFunc(h.Detail)
	r.Path("/authorities/{id:[0-9]+}/edit").Methods(http.MethodGet).HandlerFunc(h.EditForm)
	r.Path("/authorities/{id:[0-9]+}/edit").Methods(http.MethodPost).HandlerFunc(h.Update)
	r.Path("/authorities/{id:[0-9]+}/delete").Methods(http.MethodGet).HandlerFunc(h.DeleteForm)
	r.Path("/authorities/{id:[0-9]+}/delete").Methods(http.MethodPost).HandlerFunc(h.Delete)

	return s
}

// fixtures inserts three authorities: a TLS one valid for two years, open to
// any host, that signs the certificate of a partner; a TLS one that expires
// in ten days, restricted to two hosts; an SSH one restricted to a host and
// its port.
func (s *suite) fixtures(t *testing.T) {
	t.Helper()

	s.valid = newCA(t, "Waarp Test CA", time.Now().Add(2*365*24*time.Hour))
	s.soon = newCA(t, "Soon Expiring CA", time.Now().Add(10*24*time.Hour))
	s.sshLine, s.sshPub = sshKey(t)

	s.tls = &model.Authority{Name: "ca-partners", Type: auth.AuthorityTLS, PublicIdentity: s.valid.PEM}
	s.tlsSoon = &model.Authority{
		Name: "ca-soon", Type: auth.AuthorityTLS, PublicIdentity: s.soon.PEM,
		ValidHosts: []string{"a.example.org", "b.example.org", "c.example.org"},
	}
	s.sh = &model.Authority{
		Name: "ssh-ca", Type: sftp.AuthoritySSHCert, PublicIdentity: s.sshLine, ValidHosts: []string{"sftp.example.org:22"},
	}

	for _, a := range []*model.Authority{s.tls, s.tlsSoon, s.sh} {
		require.NoError(t, s.db.Insert(a).Run())
	}

	s.partner = &model.RemoteAgent{
		Name: "part-r66-tls", Protocol: "r66-tls", Address: types.Addr(partnerIP, 6667),
		ProtoConfig: map[string]any{"serverLogin": "waarp", "minTLSVersion": "v1.2", "serverPassword": "legacy-secret"},
	}
	require.NoError(t, s.db.Insert(s.partner).Run())

	s.signedCert = &model.Credential{
		RemoteAgentID: s.partner.NullableID(), Name: "cert-partner", Type: auth.TLSTrustedCertificate,
		Value: s.valid.leaf(t, partnerIP),
	}
	require.NoError(t, s.db.Insert(s.signedCert).Run())
}

// raw inserts an authority without the checks of the model, as an import of
// an older version could leave it: expired, or with an unreadable identity.
func (s *suite) raw(t *testing.T, name, typ, identity string) int64 {
	t.Helper()

	require.NoError(t, s.db.Exec("INSERT INTO "+model.TableAuthorities+" (name, type, public_identity) VALUES (?, ?, ?)",
		name, typ, identity))

	var a model.Authority
	require.NoError(t, s.db.Get(&a, "name=?", name).Run())

	return a.ID
}

func (s *suite) request(t *testing.T, perms model.PermsMask, method, target string, body *bytes.Buffer, contentType string,
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

func (s *suite) post(t *testing.T, perms model.PermsMask, target string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()

	return s.request(t, perms, http.MethodPost, target, bytes.NewBufferString(form.Encode()), "application/x-www-form-urlencoded")
}

// upload posts a multipart form: the fields, then a file in the given field.
func (s *suite) upload(t *testing.T, target string, fields map[string]string, fileField, content string,
) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}

	part, err := w.CreateFormFile(fileField, "identity.pem")
	require.NoError(t, err)

	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	return s.request(t, permWrite, http.MethodPost, target, body, w.FormDataContentType())
}

func (s *suite) authority(t *testing.T, name string) *model.Authority {
	t.Helper()

	var a model.Authority
	require.NoError(t, s.db.Get(&a, "name=?", name).Run())

	return &a
}

func path(a *model.Authority, suffix string) string {
	return "/webui/authorities/" + strconv.FormatInt(a.ID, 10) + suffix
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
		s.raw(t, "ca-expired", auth.AuthorityTLS, newCA(t, "Old CA", time.Now().Add(-24*time.Hour)).PEM)
		s.raw(t, "ca-broken", auth.AuthorityTLS, "garbage")

		rec := s.get(t, permAll, "/webui/authorities")
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Autorités d&#39;authentification · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="/webui/authorities" aria-current="page"`, "the sidebar entry leads to the new screen")
		assert.NotContains(t, body, "¿", "every key must be in the catalogue")
		assert.Equal(t, 1, strings.Count(body, "w-btn--primary"))
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="/webui/authorities/new">`)
		assert.Equal(t, 2, strings.Count(body, `type="checkbox" name="type"`), "two types")
		assert.Equal(t, 3, strings.Count(body, `type="checkbox" name="validity"`), "three validity states")
		assert.Contains(t, body, `aria-live="polite">5 autorités</span>`)

		assert.Equal(t, 5, rows(body))
		indexes(t, body, "<tbody>", ">ca-broken<", ">ca-expired<", ">ca-partners<", ">ca-soon<", ">ssh-ca<", "</tbody>")
		assert.Contains(t, body, `>ca-partners</a></td><td>TLS</td><td>Waarp Test CA</td><td><span class="w-status w-status--ok">Valide jusqu&#39;au`)
		assert.Contains(t, body, `>ca-soon</a></td><td>TLS</td><td>Soon Expiring CA</td><td><span class="w-status w-status--warn">`)
		assert.Contains(t, body, `>ca-expired</a></td><td>TLS</td><td>Old CA</td><td><span class="w-status w-status--ko">`)
		assert.Contains(t, body, `>ca-broken</a></td><td>TLS</td><td><span class="w-muted">Illisible</span></td>`)
		assert.Contains(t, body, `>ssh-ca</a></td><td>SSH</td><td>ssh-ed25519</td><td><span class="w-muted">—</span></td><td>sftp.example.org:22</td>`)
		assert.Contains(t, body, "<td>Tous les hôtes</td>")
		assert.Contains(t, body, "<td>a.example.org, b.example.org et 1 autre</td>")
		indexes(t, body, ">ca-partners</a>", "Voir la fiche", "Modifier l&#39;autorité", "Supprimer l&#39;autorité", ">ca-soon</a>")
	})

	t.Run("Given filters and a sort by validity", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/authorities?validity=warn").Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, ">ca-soon</a>")
		assert.Contains(t, body, `name="validity" value="warn" checked`)

		body = s.get(t, permRead, "/webui/authorities?type="+sftp.AuthoritySSHCert).Body.String()
		assert.Equal(t, 1, rows(body))
		assert.Contains(t, body, ">ssh-ca</a>")

		body = s.get(t, permRead, "/webui/authorities?sort=validity&dir=asc").Body.String()
		indexes(t, body, ">ca-soon<", ">ca-partners<", ">ssh-ca<")

		body = s.get(t, permRead, "/webui/authorities?q=zz").Body.String()
		assert.Contains(t, body, "Aucune autorité ne commence par « zz »")
	})

	t.Run("Given a user with the read right only", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, "/webui/authorities").Body.String()
		assert.NotContains(t, body, "w-btn--primary", "no coloured action without the write right")
		assert.NotContains(t, body, "Modifier l&#39;autorité")
		assert.NotContains(t, body, "Supprimer l&#39;autorité")
		assert.Contains(t, body, "Voir la fiche")
	})

	t.Run("Given no authority at all", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permAll, "/webui/authorities").Body.String()
		assert.Contains(t, body, "Aucune autorité d&#39;authentification pour l&#39;instant")
		assert.Contains(t, body, `href="/webui/authorities/new"`)
	})
}

func TestDetail(t *testing.T) {
	t.Parallel()

	t.Run("Given the TLS authority that signs a partner certificate", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.get(t, permAll, path(s.tls, ""))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Autorité « ca-partners »")
		assert.Contains(t, body, "Autorité de certification TLS <code class=\"w-code\">tls_authority</code>")
		assert.Contains(t, body, `<code class="w-code">Waarp Test CA</code>`)
		assert.Contains(t, body, "-----BEGIN CERTIFICATE-----", "the certificate in full, folded")
		assert.Contains(t, body, "Tous les hôtes")
		assert.Contains(t, body, "Comparés au nom d&#39;hôte du serveur, sans port")
		assert.Contains(t, body, `href="/webui/partners/`+strconv.FormatInt(s.partner.ID, 10)+`/auth/`+
			strconv.FormatInt(s.signedCert.ID, 10)+`">cert-partner</a> · Partenaire « part-r66-tls »`)
		assert.Contains(t, body, `class="w-btn w-btn--primary" href="`+path(s.tls, "/edit")+`"`)
	})

	t.Run("Given the SSH authority", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permRead, path(s.sh, "")).Body.String()
		assert.Contains(t, body, `<code class="w-code">ssh-ed25519</code>`)
		assert.Contains(t, body, ssh.FingerprintSHA256(s.sshPub))
		assert.Contains(t, body, `<code class="w-code">sftp.example.org:22</code>`)
		assert.Contains(t, body, "avec son port")
		assert.NotContains(t, body, "Certificats signés", "SSH certificates are not stored")
		assert.NotContains(t, body, "w-btn--primary", "no edition without the write right")
	})

	t.Run("Given an unreadable identity", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		id := s.raw(t, "ca-broken", auth.AuthorityTLS, "garbage")

		body := s.get(t, permRead, "/webui/authorities/"+strconv.FormatInt(id, 10)).Body.String()
		assert.Contains(t, body, "L&#39;identité publique enregistrée est illisible")
	})

	t.Run("Given an unknown authority", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		rec := s.get(t, permRead, "/webui/authorities/999")
		assert.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Autorité introuvable")
	})
}

func TestNewAuthority(t *testing.T) {
	t.Parallel()

	t.Run("Given the empty form", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		body := s.get(t, permWrite, "/webui/authorities/new").Body.String()
		assert.Contains(t, body, `enctype="multipart/form-data"`)
		assert.Contains(t, body, `<select class="w-select" id="type" name="type" required data-switch aria-describedby="type-h">`)
		assert.Contains(t, body, `<option value="tls_authority">Autorité de certification TLS</option>`)
		assert.Contains(t, body, `<option value="ssh_cert_authority">Autorité de certification SSH</option>`)
		assert.Contains(t, body, `data-case="tls_authority"`)
		assert.Contains(t, body, `data-case="ssh_cert_authority"`)
		assert.Contains(t, body, `name="tls_authority_file"`)
		assert.Contains(t, body, `placeholder="-----BEGIN CERTIFICATE-----"`)
		assert.Contains(t, body, "À renseigner")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a TLS authority pasted, with hosts", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		c := newCA(t, "Pasted CA", time.Now().Add(365*24*time.Hour))

		rec := s.post(t, permWrite, "/webui/authorities/new", url.Values{
			"name": {" pasted "}, "type": {auth.AuthorityTLS}, "tls_authority_text": {c.PEM},
			"ssh_cert_authority_text": {"ignored"}, "hosts": {"b.example.org\r\n\r\na.example.org\nb.example.org\n"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), "done=created")

		a := s.authority(t, "pasted")
		assert.Equal(t, auth.AuthorityTLS, a.Type)
		assert.Equal(t, strings.TrimSpace(c.PEM), a.PublicIdentity)
		assert.Equal(t, []string{"a.example.org", "b.example.org"}, a.ValidHosts, "blanks and duplicates dropped")
	})

	t.Run("Given an SSH authority uploaded as a file", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		line, _ := sshKey(t)

		rec := s.upload(t, "/webui/authorities/new", map[string]string{"name": "uploaded", "type": sftp.AuthoritySSHCert},
			"ssh_cert_authority_file", line+"\r\n")
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, line, s.authority(t, "uploaded").PublicIdentity)
	})

	t.Run("Given forms the screen or the model refuses", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, "/webui/authorities/new", url.Values{"name": {""}, "type": {""}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body := rec.Body.String()
		assert.Contains(t, body, "L&#39;autorité n&#39;a pas été enregistrée.")
		assert.Contains(t, body, "Ce champ est obligatoire.")
		assert.Contains(t, body, "Choisissez un type.")

		rec = s.post(t, permWrite, "/webui/authorities/new", url.Values{"name": {"x"}, "type": {auth.AuthorityTLS}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce champ est obligatoire.", "the identity is required")

		rec = s.post(t, permWrite, "/webui/authorities/new",
			url.Values{"name": {"x"}, "type": {auth.AuthorityTLS}, "tls_authority_text": {"garbage"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		body = rec.Body.String()
		assert.Contains(t, body, "Ce contenu n&#39;est pas un certificat lisible au format PEM.")
		assert.Contains(t, body, ">garbage</textarea>", "the pasted text comes back")

		rec = s.post(t, permWrite, "/webui/authorities/new",
			url.Values{"name": {"x"}, "type": {auth.AuthorityTLS}, "tls_authority_text": {s.valid.leaf(t, partnerIP)}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce certificat n&#39;est pas celui d&#39;une autorité de certification.")

		rec = s.post(t, permWrite, "/webui/authorities/new", url.Values{
			"name": {"x"}, "type": {auth.AuthorityTLS},
			"tls_authority_text": {newCA(t, "Old CA", time.Now().Add(-24*time.Hour)).PEM},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce certificat ne se vérifie pas : x509: certificate has expired")

		rec = s.post(t, permWrite, "/webui/authorities/new",
			url.Values{"name": {"ca-partners"}, "type": {auth.AuthorityTLS}, "tls_authority_text": {s.soon.PEM}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Une autre autorité porte déjà ce nom.")

		rec = s.post(t, permWrite, "/webui/authorities/new",
			url.Values{"name": {"x"}, "type": {sftp.AuthoritySSHCert}, "ssh_cert_authority_text": {"not a key"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce contenu n&#39;est pas une clé publique lisible")

		count, err := s.db.Count(&model.Authority{}).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 3, count, "nothing created")
	})
}

func TestEditAuthority(t *testing.T) {
	t.Parallel()

	t.Run("Given the form of the restricted TLS authority", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		body := s.get(t, permWrite, path(s.tlsSoon, "/edit")).Body.String()
		assert.Contains(t, body, `id="name" name="name" value="ca-soon"`)
		assert.Contains(t, body, `<option value="tls_authority" selected>`)
		assert.Contains(t, body, "a.example.org\nb.example.org\nc.example.org</textarea>")
		assert.Contains(t, body, "Soon Expiring CA · Expire le", "what is in place")
		assert.Contains(t, body, "Déposer un nouveau fichier le remplace")
		assert.Contains(t, body, "3 hôtes")
	})

	t.Run("Given a saved edition without identity", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, path(s.tlsSoon, "/edit"), url.Values{
			"name": {"ca-soon-2"}, "type": {auth.AuthorityTLS}, "tls_authority_text": {""}, "hosts": {"z.example.org"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Contains(t, rec.Header().Get("Location"), path(s.tlsSoon, "")+"?done=updated")

		a := s.authority(t, "ca-soon-2")
		assert.Equal(t, s.soon.PEM, a.PublicIdentity, "an empty identity keeps the current one")
		assert.Equal(t, []string{"z.example.org"}, a.ValidHosts, "the hosts are replaced")
	})

	t.Run("Given a change of type without identity", func(t *testing.T) {
		t.Parallel()

		s := newSuite(t)
		s.fixtures(t)

		rec := s.post(t, permWrite, path(s.tls, "/edit"), url.Values{"name": {"ca-partners"}, "type": {sftp.AuthoritySSHCert}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Ce champ est obligatoire.")
		assert.Equal(t, auth.AuthorityTLS, s.authority(t, "ca-partners").Type, "nothing changed")
	})
}

func TestDeleteAuthority(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	body := s.get(t, permAll, path(s.tls, "/delete")).Body.String()
	assert.Contains(t, body, "Supprimer l&#39;autorité « ca-partners » ?")
	assert.Contains(t, body, "Les certificats qu&#39;elle signe ne seront plus acceptés")
	assert.Contains(t, body, ">cert-partner</a> · Partenaire « part-r66-tls »")
	assert.Contains(t, body, ">Supprimer</button>")

	rec := s.post(t, permAll, path(s.tlsSoon, "/delete"), url.Values{})
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Contains(t, rec.Header().Get("Location"), "done=deleted")

	count, err := s.db.Count(&model.Authority{}).Where("name=?", "ca-soon").Run()
	require.NoError(t, err)
	assert.Zero(t, count)

	hosts, err := s.db.Count(&model.Host{}).Where("authority_id=?", s.tlsSoon.ID).Run()
	require.NoError(t, err)
	assert.Zero(t, hosts, "the hosts go with the authority")
}

func TestSigns(t *testing.T) {
	t.Parallel()

	one := newCA(t, "One", time.Now().Add(time.Hour))
	other := newCA(t, "Other", time.Now().Add(time.Hour))
	leaf := one.leaf(t, partnerIP)

	assert.True(t, signs(one.cert, leaf))
	assert.True(t, signs(one.cert, other.PEM+leaf), "anywhere in the chain")
	assert.False(t, signs(other.cert, leaf))
	assert.False(t, signs(one.cert, one.PEM), "the authority itself does not count")
	assert.False(t, signs(one.cert, "garbage"))
}

func TestHostsText(t *testing.T) {
	t.Parallel()

	echo := func(key string, args ...any) string {
		return strings.TrimSpace(fmt.Sprintln(append([]any{key}, args...)...))
	}

	assert.Equal(t, "authorities.hosts.all", hostsText(echo, nil))
	assert.Equal(t, "a, b", hostsText(echo, []string{"a", "b"}))
	assert.Equal(t, "authorities.hosts.more_one hosts a, b", hostsText(echo, []string{"a", "b", "c"}))
	assert.Equal(t, "authorities.hosts.more hosts a, b n 2", hostsText(echo, []string{"a", "b", "c", "d"}))
	assert.Equal(t, []string{"a", "b"}, parseHosts(" a \n\n b\na\n"))
}
