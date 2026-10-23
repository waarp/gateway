package servers

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	partnerPassword = "s3cret-pa55word"
	anyRet          = "?ret=q%3Dsrv"
)

// sheetFixtures adds to the servers of the suite what the sheet shows: the
// identity of the SFTP server (a host key), of the R66 over TLS server (a
// password), an HTTPS server with its certificate, and two accounts on the
// SFTP server, one with a password and a public key, one without any means.
type sheetFixtures struct {
	https                  *model.LocalAgent
	hostKey, r66Pwd, cert  *model.Credential
	partner, nobody        *model.LocalAccount
	partnerPwd, partnerKey *model.Credential
}

func (s *suite) sheet(t *testing.T) sheetFixtures {
	t.Helper()
	s.fixtures(t)

	f := sheetFixtures{
		https: &model.LocalAgent{Name: "srv-https-01", Protocol: "https", Address: types.Addr("localhost", 8443)},
	}
	require.NoError(t, s.db.Insert(f.https).Run())

	f.hostKey = &model.Credential{LocalAgentID: s.sftp.NullableID(), Name: "cle-hote", Type: sftp.AuthSSHPrivateKey, Value: testhelpers.RSAPk}
	f.r66Pwd = &model.Credential{LocalAgentID: s.r66.NullableID(), Name: "mot-de-passe-r66", Type: auth.Password, Value: partnerPassword}
	f.cert = &model.Credential{
		LocalAgentID: f.https.NullableID(), Name: "cert-localhost", Type: auth.TLSCertificate,
		Value: testhelpers.LocalhostCert, Value2: testhelpers.LocalhostKey,
	}
	f.partner = &model.LocalAccount{LocalAgentID: s.sftp.ID, Login: "partner", IPAddresses: types.IPList{"10.20.30.4", "::1"}}
	f.nobody = &model.LocalAccount{LocalAgentID: s.sftp.ID, Login: "nobody"}

	require.NoError(t, s.db.Insert(f.hostKey).Run())
	require.NoError(t, s.db.Insert(f.r66Pwd).Run())
	require.NoError(t, s.db.Insert(f.cert).Run())
	require.NoError(t, s.db.Insert(f.partner).Run())
	require.NoError(t, s.db.Insert(f.nobody).Run())

	f.partnerPwd = &model.Credential{LocalAccountID: f.partner.NullableID(), Name: "mot-de-passe", Type: auth.Password, Value: partnerPassword}
	f.partnerKey = &model.Credential{LocalAccountID: f.partner.NullableID(), Name: "cle-partner", Type: sftp.AuthSSHPublicKey, Value: testhelpers.SSHPbk}
	require.NoError(t, s.db.Insert(f.partnerPwd).Run())
	require.NoError(t, s.db.Insert(f.partnerKey).Run())

	return f
}

func (s *suite) accountPath(acc *model.LocalAccount, suffix string) string {
	return fmt.Sprintf("/webui/servers/%d/accounts/%d%s", acc.LocalAgentID, acc.ID, suffix)
}

// upload posts a multipart form: fields and files (by field name).
func (s *suite) upload(t *testing.T, perms model.PermsMask, target string, fields, files map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var body bytes.Buffer

	writer := multipart.NewWriter(&body)
	for name, value := range fields {
		require.NoError(t, writer.WriteField(name, value))
	}

	for name, content := range files {
		part, err := writer.CreateFormFile(name, name+".pem")
		require.NoError(t, err)
		_, err = part.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, writer.Close())

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: perms}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")

	req := httptest.NewRequestWithContext(ctx, http.MethodPost, target, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

// credOf reads a means back from the database.
func (s *suite) credOf(t *testing.T, id int64) *model.Credential {
	t.Helper()

	var cred model.Credential
	require.NoError(t, s.db.Get(&cred, "id=?", id).Run())

	return &cred
}

func (s *suite) credCount(t *testing.T, owner model.CredOwnerTable) uint64 {
	t.Helper()

	n, err := s.db.Count(&model.Credential{}).Where(owner.GetCredCond()).Run()
	require.NoError(t, err)

	return n
}

// The kinds the screen offers are the ones the model accepts for the owner
// and the protocol: the table of the screen never offers what the model
// refuses.
func TestServerAuthTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given the SFTP server, its host key and every right", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(s.sftp, "/auth")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Serveur « srv-sftp-01 » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="/webui/servers?q=srv">Serveurs locaux</a></li>`, "the breadcrumb goes back to the list as left")
		assert.Contains(t, body, `<nav class="w-tabs" aria-label="Onglets de la fiche">`)
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "")+`?ret=q%3Dsrv">Détails</a>`)
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/auth")+`?ret=q%3Dsrv" aria-current="page">Authentification <span class="w-tabs__count">1</span></a>`)
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dsrv">Comptes locaux <span class="w-tabs__count">2</span></a>`)
		assert.Contains(t, body, "Identité présentée par le serveur aux partenaires")
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/auth/new")+`?ret=q%3Dsrv">`, "a second host key may be added")
		assert.Contains(t, body, `>cle-hote</a>`)
		assert.Contains(t, body, "<td>Clé privée SSH</td>")
		assert.Contains(t, body, "<td>Clé privée ssh-rsa, non affichée</td>")
		assert.Contains(t, body, "1 moyen")
		assert.Contains(t, body, `href="`+s.path(s.sftp, fmt.Sprintf("/auth/%d", f.hostKey.ID))+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, `class="is-danger" href="`+s.path(s.sftp, fmt.Sprintf("/auth/%d/delete", f.hostKey.ID))+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, `<input type="hidden" name="ret" value="q=srv">`, "the search keeps the list state")
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given the HTTPS server and its certificate", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.path(f.https, "/auth")).Body.String()
		assert.Contains(t, body, "<td>Certificat TLS et sa clé</td>")
		assert.Contains(t, body, `<span class="w-status w-status--ok">Valide jusqu&#39;au 29/01/2084</span>`)
		assert.Contains(t, body, `<span class="w-cell-sub">localhost · clé privée définie, non affichée</span>`)
		assert.NotContains(t, body, "w-btn--primary", "no coloured action without the write right")
		assert.NotContains(t, body, "/edit", "no edition without the write right")
		assert.NotContains(t, body, "/delete", "no deletion without the delete right")
	})

	t.Run("Given the R66 over TLS server and its password", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, s.path(s.r66, "/auth")).Body.String()
		assert.Contains(t, body, "<td>Mot de passe</td>")
		assert.Contains(t, body, "<td>Défini, non affiché</td>")
		assert.Contains(t, body, `href="`+s.path(s.r66, "/auth/new")+`">`, "a certificate may still be added")
		assert.NotContains(t, body, "/delete", "no deletion without the delete right")
	})

	t.Run("Given the plain PeSIT server", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.pesit, "/auth")).Body.String()
		assert.Contains(t, body, "Ce protocole ne demande aucun moyen d&#39;authentification au serveur")
		assert.NotContains(t, body, "/auth/new", "nothing can be added")
		assert.Contains(t, body, `Authentification <span class="w-tabs__count">0</span>`)
	})

	t.Run("Given a search and a sort", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.sftp, "/auth?q=zzz")).Body.String()
		assert.Contains(t, body, "Aucun moyen ne correspond à « zzz »")
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/auth")+`">Effacer la recherche</a>`)

		body = s.get(t, permAll, s.path(s.sftp, "/auth?sort=type&dir=desc&ret=q%3Dsrv")).Body.String()
		assert.Contains(t, body, `aria-sort="descending"`)
		assert.Contains(t, body, "Moyens d&#39;authentification, page 1 sur 1, triés par type")
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/auth")+`?ret=q%3Dsrv&amp;sort=name&amp;dir=asc"`, "the sort links keep the list state")
	})

	t.Run("Given an unknown server", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, "/webui/servers/999/auth")
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "Serveur introuvable")
	})
}

func TestAccountsTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given the SFTP server and every right", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(s.sftp, "/accounts")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dsrv" aria-current="page">Comptes locaux <span class="w-tabs__count">2</span></a>`)
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/accounts/new")+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, "2 comptes")
		indexes(t, body, "<tbody>", `>nobody</a>`, `>partner</a>`)
		assert.Contains(t, body, `<span class="w-status w-status--warn">Aucun moyen : ne peut pas se connecter</span>`)
		assert.Contains(t, body, "<td>Mot de passe, Clé publique SSH</td>")
		assert.Contains(t, body, `href="`+s.accountPath(f.partner, "")+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, `href="`+s.accountPath(f.partner, "/edit")+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, `class="is-danger" href="`+s.accountPath(f.partner, "/delete")+`?ret=q%3Dsrv">`)
		assert.NotContains(t, body, "¿")
	})

	t.Run("Given a sort by authentication, descending", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.path(s.sftp, "/accounts?sort=auth&dir=desc")).Body.String()
		indexes(t, body, "<tbody>", `>partner</a>`, `>nobody</a>`)
		assert.Contains(t, body, "Comptes locaux, page 1 sur 1, triés par authentification")
		assert.NotContains(t, body, "w-btn--primary")
		assert.NotContains(t, body, "/edit")
	})

	t.Run("Given a search without match", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.sftp, "/accounts?q=zzz")).Body.String()
		assert.Contains(t, body, "Aucun compte ne correspond à « zzz »")
	})

	t.Run("Given a server without any account", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(f.https, "/accounts")).Body.String()
		assert.Contains(t, body, "Aucun compte local pour l&#39;instant")
		assert.Contains(t, body, `href="`+s.path(f.https, "/accounts/new")+`">`)
	})
}

func TestAccountDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given the partner account and every right", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.accountPath(f.partner, "")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Compte « partner » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "")+`?ret=q%3Dsrv">srv-sftp-01</a></li>`, "the breadcrumb goes through the server")
		assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dsrv">Comptes locaux</a></li>`)
		assert.Contains(t, body, `<a href="`+s.accountPath(f.partner, "")+`?ret=q%3Dsrv" aria-current="page">Détails</a>`)
		assert.Contains(t, body, `<a href="`+s.accountPath(f.partner, "/auth")+`?ret=q%3Dsrv">Authentification <span class="w-tabs__count">2</span></a>`)
		assert.Contains(t, body, "<dt>Serveur</dt><dd>srv-sftp-01 (SFTP)</dd>")
		assert.Contains(t, body, `<dt>Identifiant</dt><dd><code class="w-code">partner</code></dd>`)
		assert.Contains(t, body, `<dt>Adresses IP autorisées</dt><dd><code class="w-code">10.20.30.4, ::1</code></dd>`)
		assert.Contains(t, body, "<dt>Moyens d&#39;authentification</dt><dd>Mot de passe, Clé publique SSH</dd>")
		assert.Contains(t, body, `href="`+s.accountPath(f.partner, "/edit")+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, `class="w-icon-btn is-danger" href="`+s.accountPath(f.partner, "/delete")+`?ret=q%3Dsrv"`,
			"the only entry of the menu is an icon button")
		assert.NotContains(t, body, "Voir le compte", "no link to itself")
	})

	t.Run("Given the account without any means, read only", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.accountPath(f.nobody, "")).Body.String()
		assert.Contains(t, body, "<dt>Adresses IP autorisées</dt><dd>Toutes</dd>")
		assert.Contains(t, body, "<dt>Moyens d&#39;authentification</dt><dd>Aucun</dd>")
		assert.NotContains(t, body, "/edit")
		assert.NotContains(t, body, "/delete")
	})

	t.Run("Given an unknown account", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(s.sftp, "/accounts/999")+anyRet)
		require.Equal(t, http.StatusNotFound, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Compte introuvable · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dsrv">`)
	})

	t.Run("Given an account of another server", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, s.path(s.r66, fmt.Sprintf("/accounts/%d", f.partner.ID))).Code)
	})
}

func TestAccountAuthTab(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	rec := s.get(t, permAll, s.accountPath(f.partner, "/auth"))
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "Moyens que le partenaire présente pour ce compte")
	assert.Contains(t, body, `<a href="`+s.accountPath(f.partner, "/auth")+`" aria-current="page">Authentification <span class="w-tabs__count">2</span></a>`)
	assert.Contains(t, body, "<td>Mot de passe</td>")
	assert.Contains(t, body, "<td>Clé publique SSH</td>")
	assert.Contains(t, body, "<td>ssh-rsa · SHA256:")
	assert.Contains(t, body, `href="`+s.accountPath(f.partner, "/auth/new")+`">`, "a second key may be added")
	assert.Contains(t, body, `href="`+s.path(s.sftp, "/accounts")+`">`, "back to the server")

	body = s.get(t, permAll, s.accountPath(f.nobody, "/auth")).Body.String()
	assert.Contains(t, body, "Ce compte ne peut pas se connecter tant qu&#39;il n&#39;a aucun moyen")
	assert.Contains(t, body, `href="`+s.accountPath(f.nobody, "/auth/new")+`">`)
}

func TestCredDetail(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given the certificate of the HTTPS server", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(f.https, fmt.Sprintf("/auth/%d", f.cert.ID))+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Moyen « cert-localhost » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Certificat TLS et sa clé pour le serveur « srv-https-01 »")
		assert.Contains(t, body, `<dt>État</dt><dd><span class="w-status w-status--ok">Valide jusqu&#39;au 29/01/2084</span></dd>`)
		assert.Contains(t, body, `<dt>Sujet</dt><dd><code class="w-code">localhost</code></dd>`)
		assert.Contains(t, body, "<dt>Émetteur</dt><dd>Auto-signé</dd>")
		assert.Contains(t, body, "<dt>Validité</dt><dd>du 01/01/1970 au 29/01/2084</dd>")
		assert.Contains(t, body, "<dt>Empreinte SHA-256</dt><dd><code class=\"w-code\">")
		assert.Contains(t, body, "<dt>Clé privée</dt><dd>Définie, jamais affichée</dd>")
		assert.Contains(t, body, "Certificat (PEM)")
		assert.Contains(t, body, "-----BEGIN CERTIFICATE-----")
		assert.Contains(t, body, `href="`+s.path(f.https, fmt.Sprintf("/auth/%d/edit", f.cert.ID))+`?ret=q%3Dsrv">`)
		assert.Contains(t, body, `class="w-icon-btn is-danger" href="`+s.path(f.https, fmt.Sprintf("/auth/%d/delete", f.cert.ID))+`?ret=q%3Dsrv"`)
		assert.Contains(t, body, `<a href="`+s.path(f.https, "")+`?ret=q%3Dsrv">srv-https-01</a></li>`)
	})

	t.Run("Given the public key of the partner", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permRead, s.accountPath(f.partner, fmt.Sprintf("/auth/%d", f.partnerKey.ID))).Body.String()
		assert.Contains(t, body, "Clé publique SSH pour le compte « partner » du serveur « srv-sftp-01 »")
		assert.Contains(t, body, `<dt>Type de clé</dt><dd><code class="w-code">ssh-rsa</code></dd>`)
		assert.Contains(t, body, `<dt>Empreinte SHA-256</dt><dd><code class="w-code">SHA256:`)
		assert.Contains(t, body, "Clé publique</h2>")
		assert.Contains(t, body, "ssh-rsa AAAAB3NzaC1yc2E")
		assert.NotContains(t, body, "/edit")
	})

	t.Run("Given the host key of the SFTP server", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.path(s.sftp, fmt.Sprintf("/auth/%d", f.hostKey.ID))).Body.String()
		assert.Contains(t, body, `<dt>Type de clé</dt><dd><code class="w-code">ssh-rsa</code></dd>`)
		assert.Contains(t, body, "<dt>Clé privée</dt><dd>Définie, jamais affichée</dd>")
		assert.NotContains(t, body, "PRIVATE KEY")
	})

	t.Run("Given the password of the partner", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permAll, s.accountPath(f.partner, fmt.Sprintf("/auth/%d", f.partnerPwd.ID))).Body.String()
		assert.Contains(t, body, "<dt>Contenu</dt><dd>Définie, jamais affichée</dd>")
	})

	t.Run("Given an unknown means, or one of another owner", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.path(s.sftp, "/auth/999")+anyRet)
		require.Equal(t, http.StatusNotFound, rec.Code)
		assert.Contains(t, rec.Body.String(), "<title>Moyen d&#39;authentification introuvable · gw-test · Waarp Gateway</title>")
		assert.Contains(t, rec.Body.String(), `href="`+s.path(s.sftp, "/auth")+`?ret=q%3Dsrv">`)

		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, s.path(s.sftp, fmt.Sprintf("/auth/%d", f.partnerPwd.ID))).Code,
			"a means of an account is not reachable through the server")
	})
}

func TestCreateCred(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given the form of a new means of the SFTP server", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, s.path(s.sftp, "/auth/new")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Nouveau moyen d&#39;authentification · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `enctype="multipart/form-data"`)
		assert.Contains(t, body, `<select class="w-select" id="type" name="type" required data-switch="type"`)
		assert.Contains(t, body, `<option value="" selected>Choisissez un type</option><option value="ssh_private_key">Clé privée SSH</option></select>`)
		assert.Contains(t, body, `data-case="trusted_tls_certificate ssh_private_key ssh_public_key" data-of="type"`)
		assert.Contains(t, body, `<input type="file" id="file" name="file"`)
		assert.Contains(t, body, `name="pem"`)
		assert.Contains(t, body, `Authentification <span class="w-tabs__count">1</span>`)
		assert.Contains(t, body, `href="`+s.path(s.sftp, "/auth")+`?ret=q%3Dsrv">`, "cancel goes back to the tab")
	})

	t.Run("Given the form of the HTTPS server with its certificate already set", func(t *testing.T) {
		t.Parallel()

		body := s.get(t, permWrite, s.path(f.https, "/auth/new")).Body.String()
		assert.Contains(t, body, `<option value="tls_certificate">Certificat TLS et sa clé</option>`, "several certificates are possible")
	})

	t.Run("Given a certificate uploaded for a new HTTPS server", func(t *testing.T) {
		t.Parallel()

		srv := &model.LocalAgent{Name: "srv-https-02", Protocol: "https", Address: types.Addr("localhost", 8444)}
		require.NoError(t, s.db.Insert(srv).Run())

		rec := s.upload(t, permWrite, s.path(srv, "/auth/new")+anyRet,
			map[string]string{"name": "cert-2026", "type": "tls_certificate"},
			map[string]string{"cert_file": testhelpers.LocalhostCert, "key_file": testhelpers.LocalhostKey})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, s.path(srv, "/auth")+"?done=created&kind=cred&name=cert-2026&ret=q%3Dsrv", rec.Header().Get("Location"))

		creds, err := srv.GetCredentials(s.db)
		require.NoError(t, err)
		require.Len(t, creds, 1)
		assert.Equal(t, "cert-2026", creds[0].Name)
		assert.Contains(t, creds[0].Value, "-----BEGIN CERTIFICATE-----")
		assert.Contains(t, creds[0].Value2, "-----BEGIN PRIVATE KEY-----", "read back decrypted by the model")

		body := s.get(t, permWrite, rec.Header().Get("Location")).Body.String()
		assert.Contains(t, body, "Moyen « cert-2026 » ajouté")
	})

	t.Run("Given a certificate pasted for a new FTPS server", func(t *testing.T) {
		t.Parallel()

		srv := &model.LocalAgent{Name: "srv-ftps-01", Protocol: "ftps", Address: types.Addr("localhost", 2121)}
		require.NoError(t, s.db.Insert(srv).Run())

		rec := s.post(t, permWrite, s.path(srv, "/auth/new"), url.Values{
			"name": {""}, "type": {"tls_certificate"},
			"cert_pem": {"\r\n" + strings.ReplaceAll(testhelpers.LocalhostCert, "\n", "\r\n")}, "key_pem": {testhelpers.LocalhostKey},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, s.path(srv, "/auth")+"?done=created&kind=cred&name=Certificat+TLS+et+sa+cl%C3%A9", rec.Header().Get("Location"),
			"an empty name takes the name of the kind")
	})

	t.Run("Given a public key and a password posted for the account without any", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.accountPath(f.nobody, "/auth/new"), url.Values{
			"name": {"cle-nobody"}, "type": {"ssh_public_key"}, "pem": {testhelpers.SSHPbk},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		rec = s.post(t, permWrite, s.accountPath(f.nobody, "/auth/new"), url.Values{
			"name": {"mdp"}, "type": {"password"}, "password": {"nobody-pa55"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		creds, err := f.nobody.GetCredentials(s.db, auth.Password)
		require.NoError(t, err)
		require.Len(t, creds, 1)
		assert.True(t, utils.IsHash(creds[0].Value), "the password is stored hashed")

		body := s.get(t, permWrite, s.accountPath(f.nobody, "/auth/new")).Body.String()
		assert.NotContains(t, body, `<option value="password">`, "a second password is not offered")
		assert.Contains(t, body, `<option value="ssh_public_key">`)
	})

	t.Run("Given forms the screen refuses", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.accountPath(f.partner, "/auth/new"), url.Values{"name": {"x"}, "type": {"ssh_public_key"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez le contenu du moyen d&#39;authentification.")
		assert.Contains(t, rec.Body.String(), `<option value="ssh_public_key" selected>`, "the chosen kind is kept")
		assert.Contains(t, rec.Body.String(), `value="x"`, "the name is kept")

		rec = s.post(t, permWrite, s.accountPath(f.partner, "/auth/new"), url.Values{"name": {"x"}, "type": {"password"}, "password": {"p"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un moyen de ce type existe déjà")

		rec = s.post(t, permWrite, s.accountPath(f.partner, "/auth/new"), url.Values{"name": {"x"}, "type": {"trusted_tls_certificate"}, "pem": {"nope"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Choisissez un type admis par le protocole.")

		rec = s.post(t, permWrite, s.accountPath(f.partner, "/auth/new"), url.Values{"name": {"x"}, "type": {"ssh_public_key"}, "pem": {"not a key"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Le contenu n&#39;a pas été accepté : ")
		assert.Contains(t, rec.Body.String(), `>not a key</textarea>`, "the pasted public content is kept")

		rec = s.post(t, permWrite, s.path(s.sftp, "/auth/new"), url.Values{"name": {"x"}, "type": {"ssh_private_key"}, "pem": {"not a key"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.NotContains(t, rec.Body.String(), "not a key", "a pasted private key is not echoed")

		rec = s.post(t, permWrite, s.path(s.sftp, "/auth/new"), url.Values{"name": {"cle-hote"}, "type": {"ssh_private_key"}, "pem": {testhelpers.RSAPk}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre moyen porte déjà ce nom.")

		assert.EqualValues(t, 1, s.credCount(t, s.sftp), "nothing was saved")
	})
}

func TestEditCred(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given the password of the partner", func(t *testing.T) {
		t.Parallel()

		before := s.credOf(t, f.partnerPwd.ID).Value
		require.True(t, utils.IsHash(before))

		rec := s.get(t, permWrite, s.accountPath(f.partner, fmt.Sprintf("/auth/%d/edit", f.partnerPwd.ID))+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Modifier « mot-de-passe » · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, `<input class="w-input" id="type" value="Mot de passe" readonly`)
		assert.Contains(t, body, "Défini, non affiché · Laissez vide pour conserver le mot de passe actuel")
		assert.NotContains(t, body, `data-case="trusted_tls_certificate`, "only the block of the kind")
		assert.NotContains(t, body, before)
		assert.NotContains(t, body, partnerPassword)

		rec = s.post(t, permWrite, s.accountPath(f.partner, fmt.Sprintf("/auth/%d/edit", f.partnerPwd.ID))+anyRet, url.Values{"name": {"mdp-partner"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, s.accountPath(f.partner, "/auth")+"?done=updated&kind=cred&name=mdp-partner&ret=q%3Dsrv", rec.Header().Get("Location"))

		after := s.credOf(t, f.partnerPwd.ID)
		assert.Equal(t, "mdp-partner", after.Name)
		assert.Equal(t, before, after.Value, "an empty password keeps the current one")

		rec = s.post(t, permWrite, s.accountPath(f.partner, fmt.Sprintf("/auth/%d/edit", f.partnerPwd.ID)), url.Values{"name": {"mdp-partner"}, "password": {"new-pa55"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		changed := s.credOf(t, f.partnerPwd.ID).Value
		assert.NotEqual(t, before, changed)
		assert.True(t, utils.IsHash(changed))
	})

	t.Run("Given the password of the R66 server, stored encrypted", func(t *testing.T) {
		t.Parallel()

		// The model encrypts an external password when it is written and
		// decrypts it when it is read: the means read back holds the plain
		// password, and an edition that keeps it stores it again as it was.
		require.Equal(t, partnerPassword, s.credOf(t, f.r66Pwd.ID).Value)

		rec := s.post(t, permWrite, s.path(s.r66, fmt.Sprintf("/auth/%d/edit", f.r66Pwd.ID)), url.Values{"name": {"mdp-r66"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		after := s.credOf(t, f.r66Pwd.ID)
		assert.Equal(t, "mdp-r66", after.Name)
		assert.Equal(t, partnerPassword, after.Value, "an empty password keeps the current one, readable as before")
	})

	t.Run("Given the certificate of the HTTPS server", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, s.path(f.https, fmt.Sprintf("/auth/%d/edit", f.cert.ID)))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "Contenu en place : localhost · Valide jusqu&#39;au 29/01/2084")
		assert.Contains(t, body, "Contenu en place : Clé privée définie, non affichée")
		assert.NotContains(t, body, strings.Split(testhelpers.LocalhostKey, "\n")[1])
		assert.NotContains(t, body, "<option", "the kind does not change")

		rec = s.post(t, permWrite, s.path(f.https, fmt.Sprintf("/auth/%d/edit", f.cert.ID)), url.Values{"name": {"cert-renamed"}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		after := s.credOf(t, f.cert.ID)
		assert.Equal(t, "cert-renamed", after.Name)
		assert.Equal(t, testhelpers.LocalhostCert, after.Value, "the certificate is kept")
		assert.Equal(t, testhelpers.LocalhostKey, after.Value2, "the key is kept")

		rec = s.post(t, permWrite, s.path(f.https, fmt.Sprintf("/auth/%d/edit", f.cert.ID)), url.Values{"name": {"cert-renamed"}, "cert_pem": {testhelpers.ClientFooCert}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code, "a certificate that does not match the key in place")
		assert.Contains(t, rec.Body.String(), "Le contenu n&#39;a pas été accepté")
	})
}

func TestDeleteCred(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	rec := s.get(t, permAll, s.path(s.r66, fmt.Sprintf("/auth/%d/delete", f.r66Pwd.ID))+anyRet)
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "<title>Supprimer le moyen « mot-de-passe-r66 » ? · gw-test · Waarp Gateway</title>")
	assert.Contains(t, body, "Les partenaires qui vérifient cette identité du serveur ne pourront plus se connecter.")
	assert.Contains(t, body, "<dt>Pour</dt><dd>le serveur « srv-r66-01 »</dd>")
	assert.Contains(t, body, `<form class="sheet-form-actions" method="post" action="`+s.path(s.r66, fmt.Sprintf("/auth/%d/delete", f.r66Pwd.ID))+`?ret=q%3Dsrv">`)

	rec = s.post(t, permAll, s.path(s.r66, fmt.Sprintf("/auth/%d/delete", f.r66Pwd.ID))+anyRet, nil)
	require.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, s.path(s.r66, "/auth")+"?done=deleted&kind=cred&name=mot-de-passe-r66&ret=q%3Dsrv", rec.Header().Get("Location"))
	assert.EqualValues(t, 0, s.credCount(t, s.r66))

	body = s.get(t, permAll, rec.Header().Get("Location")).Body.String()
	assert.Contains(t, body, "Moyen « mot-de-passe-r66 » supprimé")
	assert.Contains(t, body, "Aucun moyen d&#39;authentification pour l&#39;instant")

	body = s.get(t, permAll, s.accountPath(f.partner, fmt.Sprintf("/auth/%d/delete", f.partnerPwd.ID))).Body.String()
	assert.Contains(t, body, "Ce partenaire ne pourra plus s&#39;authentifier par ce moyen.")
}

func TestCreateAccount(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given the form of a new account of the PeSIT server", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permWrite, s.path(s.pesit, "/accounts/new")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Nouveau compte · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "Compte sur le serveur « srv-pesit-01 »")
		assert.Contains(t, body, `<select class="w-select" id="means_type" name="means_type" data-switch="means_type"`)
		assert.Contains(t, body, `<option value="" selected>Aucun pour l&#39;instant</option><option value="password">Mot de passe</option></select>`)
		assert.Contains(t, body, `data-case="password" data-of="means_type"`)
		assert.Contains(t, body, `name="means_password"`)
		assert.Contains(t, body, `aria-current="page">Comptes locaux`, "under the accounts tab")
		assert.Contains(t, body, `href="`+s.path(s.pesit, "/accounts")+`?ret=q%3Dsrv">`)
	})

	t.Run("Given an account posted with its first means", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.path(s.pesit, "/accounts/new")+anyRet, url.Values{
			"login": {" banque-x "}, "ips": {"203.0.113.10\r\n2001:DB8::1, 203.0.113.11"},
			"means_name": {""}, "means_type": {"password"}, "means_password": {"pa55"},
		})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.Equal(t, s.path(s.pesit, "/accounts")+"?done=created&kind=account&name=banque-x&ret=q%3Dsrv", rec.Header().Get("Location"))

		var acc model.LocalAccount
		require.NoError(t, s.db.Get(&acc, "login=?", "banque-x").Run())
		assert.Equal(t, s.pesit.ID, acc.LocalAgentID)
		assert.Equal(t, types.IPList{"203.0.113.10", "2001:db8::1", "203.0.113.11"}, acc.IPAddresses, "canonical addresses")

		creds, err := acc.GetCredentials(s.db)
		require.NoError(t, err)
		require.Len(t, creds, 1)
		assert.Equal(t, "Mot de passe", creds[0].Name, "an empty name takes the name of the kind")
		assert.True(t, utils.IsHash(creds[0].Value))

		body := s.get(t, permWrite, rec.Header().Get("Location")).Body.String()
		assert.Contains(t, body, "Compte « banque-x » créé")
		assert.Contains(t, body, `>banque-x</a>`)
	})

	t.Run("Given an account posted without any means", func(t *testing.T) {
		t.Parallel()

		rec := s.post(t, permWrite, s.path(s.r66, "/accounts/new"), url.Values{"login": {"alone"}, "means_type": {""}})
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		var acc model.LocalAccount
		require.NoError(t, s.db.Get(&acc, "login=?", "alone").Run())
		assert.EqualValues(t, 0, s.credCount(t, &acc))
	})

	t.Run("Given forms the screen or the model refuses", func(t *testing.T) {
		t.Parallel()

		before, err := s.db.Count(&model.LocalAccount{}).Where("local_agent_id=?", s.sftp.ID).Run()
		require.NoError(t, err)

		rec := s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{"login": {""}, "ips": {"10.0.0.1\nnot-an-ip"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez l&#39;identifiant du compte.")
		assert.Contains(t, rec.Body.String(), "« not-an-ip » n&#39;est pas une adresse IP.")
		assert.Contains(t, rec.Body.String(), "10.0.0.1\nnot-an-ip</textarea>", "the addresses are kept")

		rec = s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{"login": {"partner"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Un autre compte de ce serveur porte déjà cet identifiant.")

		// The first means is refused: the account is not created either.
		rec = s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{
			"login": {"half"}, "means_type": {"ssh_public_key"}, "means_pem": {"not a key"},
		})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Le contenu n&#39;a pas été accepté")
		assert.Contains(t, rec.Body.String(), `<option value="ssh_public_key" selected>`)

		rec = s.post(t, permWrite, s.path(s.sftp, "/accounts/new"), url.Values{"login": {"half"}, "means_type": {"password"}})
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Indiquez le contenu du moyen d&#39;authentification.")

		after, err := s.db.Count(&model.LocalAccount{}).Where("local_agent_id=?", s.sftp.ID).Run()
		require.NoError(t, err)
		assert.Equal(t, before, after, "nothing was saved")
	})

	_ = f
}

func TestEditAccount(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	rec := s.get(t, permWrite, s.accountPath(f.partner, "/edit")+anyRet)
	require.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.Contains(t, body, "<title>Modifier « partner » · gw-test · Waarp Gateway</title>")
	assert.Contains(t, body, `value="partner"`)
	assert.Contains(t, body, "10.20.30.4\n::1</textarea>")
	assert.NotContains(t, body, "means_type", "no first means on edition")
	assert.Contains(t, body, `aria-current="page">Détails</a>`, "under the tabs of the account")
	assert.Contains(t, body, `href="`+s.accountPath(f.partner, "")+`?ret=q%3Dsrv">`, "cancel goes back to the account")

	rec = s.post(t, permWrite, s.accountPath(f.partner, "/edit")+anyRet, url.Values{"login": {"partner-2"}, "ips": {""}})
	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
	assert.Equal(t, s.accountPath(f.partner, "")+"?done=updated&kind=account&name=partner-2&ret=q%3Dsrv", rec.Header().Get("Location"))

	var acc model.LocalAccount
	require.NoError(t, s.db.Get(&acc, "id=?", f.partner.ID).Run())
	assert.Equal(t, "partner-2", acc.Login)
	assert.Empty(t, acc.IPAddresses)
	assert.EqualValues(t, 2, s.credCount(t, &acc), "the means are kept")

	body = s.get(t, permWrite, rec.Header().Get("Location")).Body.String()
	assert.Contains(t, body, "Compte « partner-2 » enregistré")

	rec = s.post(t, permWrite, s.accountPath(f.partner, "/edit"), url.Values{"login": {"nobody"}})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), "Un autre compte de ce serveur porte déjà cet identifiant.")
}

func TestDeleteAccount(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	t.Run("Given an account used by a running transfer", func(t *testing.T) {
		t.Parallel()

		rule := &model.Rule{Name: "receive", IsSend: false, Path: "receive"}
		require.NoError(t, s.db.Insert(rule).Run())
		require.NoError(t, s.db.Insert(&model.Transfer{RuleID: rule.ID, LocalAccountID: f.nobody.NullableID(), DestFilename: "file"}).Run())

		rec := s.get(t, permAll, s.accountPath(f.nobody, "/delete"))
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<strong>Ce compte ne peut pas être supprimé.</strong>Des transferts en cours l&#39;utilisent.")
		assert.Contains(t, body, `href="/webui/transfers">`)
		assert.NotContains(t, body, `<form class="sheet-form-actions"`, "no deletion offered")

		rec = s.post(t, permAll, s.accountPath(f.nobody, "/delete"), nil)
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	})

	t.Run("Given the partner account", func(t *testing.T) {
		t.Parallel()

		rec := s.get(t, permAll, s.accountPath(f.partner, "/delete")+anyRet)
		require.Equal(t, http.StatusOK, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "<title>Supprimer le compte « partner » ? · gw-test · Waarp Gateway</title>")
		assert.Contains(t, body, "ce partenaire ne pourra plus se connecter au serveur « srv-sftp-01 »")
		assert.Contains(t, body, `<dt>Adresses IP autorisées</dt><dd>10.20.30.4, ::1</dd>`)
		assert.Contains(t, body, "<dt>Moyens d&#39;authentification</dt><dd>Mot de passe, Clé publique SSH</dd>")

		rec = s.post(t, permAll, s.accountPath(f.partner, "/delete")+anyRet, nil)
		require.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, s.path(s.sftp, "/accounts")+"?done=deleted&kind=account&name=partner&ret=q%3Dsrv", rec.Header().Get("Location"))

		n, err := s.db.Count(&model.Credential{}).Where("local_account_id=?", f.partner.ID).Run()
		require.NoError(t, err)
		assert.EqualValues(t, 0, n, "the means go with the account")

		body = s.get(t, permAll, rec.Header().Get("Location")).Body.String()
		assert.Contains(t, body, "Compte « partner » supprimé")
		assert.Equal(t, http.StatusNotFound, s.get(t, permAll, s.accountPath(f.partner, "")).Code)
	})
}

// No secret value ever reaches a page: not the password given, not its
// hash, not a private key, whatever the page of the sheet.
func TestNoSecretInPages(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	f := s.sheet(t)

	secrets := map[string]string{
		"the password":         partnerPassword,
		"the hash":             s.credOf(t, f.partnerPwd.ID).Value,
		"the SSH private key":  strings.Split(testhelpers.RSAPk, "\n")[1],
		"the TLS private key":  strings.Split(testhelpers.LocalhostKey, "\n")[1],
		"the stored SSH key":   s.credOf(t, f.hostKey.ID).Value,
		"the stored TLS key":   s.credOf(t, f.cert.ID).Value2,
		"the encrypted values": "AES:",
	}

	for _, path := range []string{
		s.path(s.sftp, "/auth"), s.path(s.sftp, fmt.Sprintf("/auth/%d", f.hostKey.ID)),
		s.path(s.sftp, fmt.Sprintf("/auth/%d/edit", f.hostKey.ID)), s.path(s.sftp, fmt.Sprintf("/auth/%d/delete", f.hostKey.ID)),
		s.path(f.https, "/auth"), s.path(f.https, fmt.Sprintf("/auth/%d", f.cert.ID)),
		s.path(f.https, fmt.Sprintf("/auth/%d/edit", f.cert.ID)), s.path(f.https, fmt.Sprintf("/auth/%d/delete", f.cert.ID)),
		s.path(s.sftp, "/accounts"), s.accountPath(f.partner, ""), s.accountPath(f.partner, "/auth"),
		s.accountPath(f.partner, fmt.Sprintf("/auth/%d", f.partnerPwd.ID)),
		s.accountPath(f.partner, fmt.Sprintf("/auth/%d/edit", f.partnerPwd.ID)),
		s.accountPath(f.partner, fmt.Sprintf("/auth/%d/delete", f.partnerPwd.ID)),
		s.accountPath(f.partner, "/delete"),
	} {
		rec := s.get(t, permAll, path)
		require.Equalf(t, http.StatusOK, rec.Code, "GET %s", path)

		for what, secret := range secrets {
			assert.NotContainsf(t, rec.Body.String(), secret, "GET %s shows %s", path, what)
		}
	}
}

func TestSheetHelpers(t *testing.T) {
	t.Parallel()

	q := parseQuery(url.Values{"q": {" x "}, "sort": {"auth"}, "dir": {"desc"}, "page": {"2"}, "size": {"25"}, "protocol": {"sftp"}}, accountSortKeys)
	assert.Equal(t, "dir=desc&page=2&q=x&size=25&sort=auth", sortedQuery(q.Encode(q.Page)), "the protocols belong to the servers list only")
	assert.Equal(t, "name", parseQuery(url.Values{"sort": {"state"}}, accountSortKeys).Sort, "a sort of another list falls back")

	b := liststate.Parse(url.Values{"page": {"9"}, "size": {"10"}}, accountSortKeys, pageSizes).Bounds(23)
	assert.Equal(t, liststate.Bounds{Page: 3, Pages: 3, From: 20, To: 23}, b, "the page is bounded by the count")

	ips, bad := parseIPs("10.0.0.1, 10.0.0.2;\r\n 2001:DB8::1\t")
	assert.Empty(t, bad)
	assert.Equal(t, types.IPList{"10.0.0.1", "10.0.0.2", "2001:db8::1"}, ips)
	_, bad = parseIPs("10.0.0.1\n10.0.0.0/24")
	assert.Equal(t, "10.0.0.0/24", bad, "a range is not an address")

	assert.Equal(t, "/webui/servers/1/accounts/2/auth", accountURL(1, 2, authSuffix))
}

// sortedQuery re-encodes a query with its keys in a stable order.
func sortedQuery(encoded string) string {
	v, err := url.ParseQuery(encoded)
	if err != nil {
		return encoded
	}

	return v.Encode()
}

// The tab strip of the sheet stays on every page of the server: greyed on
// the form of a new server, live on edition and on the deletion page
// (Driss, 2026-10-20).
func TestSheetTabsOnForms(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	s.fixtures(t)

	body := s.get(t, permAll, "/webui/servers/new").Body.String()
	assert.Contains(t, body, `<nav class="w-tabs" aria-label="Onglets de la fiche"><span aria-current="page">Détails</span>`)
	assert.Contains(t, body, `<span class="w-tabs__disabled" aria-disabled="true" title="Disponible après la création">Authentification</span>`)
	assert.Contains(t, body, `<span class="w-tabs__disabled" aria-disabled="true" title="Disponible après la création">Comptes locaux</span>`)
	assert.NotContains(t, body, "w-tabs__count", "nothing to count before the creation")

	body = s.get(t, permAll, s.path(s.sftp, "/edit")+anyRet).Body.String()
	assert.Contains(t, body, `<a href="`+s.path(s.sftp, "")+`?ret=q%3Dsrv" aria-current="page">Détails</a>`)
	assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/accounts")+`?ret=q%3Dsrv">Comptes locaux <span class="w-tabs__count">`)
	assert.NotContains(t, body, "w-tabs__disabled")

	body = s.get(t, permAll, s.path(s.sftp, "/delete")).Body.String()
	assert.Contains(t, body, `<a href="`+s.path(s.sftp, "/auth")+`">Authentification <span class="w-tabs__count">`)

	rec := s.post(t, permWrite, "/webui/servers/new", url.Values{"name": {""}})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	assert.Contains(t, rec.Body.String(), `<span class="w-tabs__disabled"`, "the refused form keeps the greyed strip")
}
