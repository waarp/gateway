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
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/certchain/chaintest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// uploadChain posts a multipart form: the fields, then the files of one field,
// in the order given.
func (s *suite) uploadChain(t *testing.T, target string, fields map[string]string, fileField string, files ...string,
) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	for k, v := range fields {
		require.NoError(t, w.WriteField(k, v))
	}

	for i, content := range files {
		part, err := w.CreateFormFile(fileField, fmt.Sprintf("cert-%d.pem", i))
		require.NoError(t, err)

		_, err = part.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, w.Close())

	user := &model.User{Username: "tester", Owner: "gw-test", Permissions: permWrite}
	ctx := context.WithValue(context.Background(), constants.ContextUserKey, user)
	ctx = context.WithValue(ctx, constants.ContextLanguageKey, "fr")
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, target, body)
	req.Header.Set("Content-Type", w.FormDataContentType())

	rec := httptest.NewRecorder()
	s.router.ServeHTTP(rec, req)

	return rec
}

// TestCertificateChain: the certificate of an HTTPS server given with its
// intermediates (#776): in any order, in one paste or in several files; the
// root set apart and said; a foreign certificate refused; the refusal of
// the product explained when the chain reaches no known authority; the
// chain shown on the sheet and in the list.
func TestCertificateChain(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	pki := chaintest.New(t)

	srv := &model.LocalAgent{Name: "srv-chaine", Protocol: "https", Address: types.Addr("127.0.0.1", 8444)}
	require.NoError(t, s.db.Insert(srv).Run())

	form := func(name, cert string) url.Values {
		return url.Values{"name": {name}, "type": {"tls_certificate"}, "cert_pem": {cert}, "key_pem": {pki.LeafKey}}
	}

	t.Run("the form does not send while the certificate or the key is missing", func(t *testing.T) {
		body := s.get(t, permWrite, s.path(srv, "/auth/new")).Body.String()
		assert.Contains(t, body, `<div class="w-file" data-needs data-needs-paste="cert_pem">`)
		assert.Contains(t, body, `<div class="w-file" data-needs data-needs-paste="key_pem">`)
		assert.Contains(t, body, `data-needs-note hidden>Le contenu demandé est à déposer ou à coller avant de valider.</span>`)
		assert.Contains(t, body, `type="submit" data-needs-submit`)
	})

	t.Run("the chain reaches no known authority: the refusal says who signs it", func(t *testing.T) {
		rec := s.uploadChain(t, s.path(srv, "/auth/new"),
			map[string]string{"name": "sans-racine", "type": "tls_certificate", "key_pem": pki.LeafKey},
			"cert_file", chaintest.PEM(pki.Inter), chaintest.PEM(pki.Leaf))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)

		body := rec.Body.String()
		assert.Contains(t, body, "« Test intermediate » est signé par « Test root », que la passerelle ne connaît pas",
			"the last certificate of the chain and its issuer")
		assert.Contains(t, body, "enregistrez « Test root » dans Autorités d")
		assert.Contains(t, body, `placeholder="-----BEGIN CERTIFICATE-----">-----BEGIN CERTIFICATE-----`,
			"the chain uploaded comes back in the paste field, in order")
		assert.Equal(t, 3, strings.Count(body, "-----BEGIN CERTIFICATE-----"), "the placeholder, then the two blocks pasted back")
		assert.NotContains(t, body, "EC PRIVATE KEY-----", "never the private key")
		assert.Contains(t, body, "un fichier déposé est à choisir de nouveau")
	})

	require.NoError(t, s.db.Insert(&model.Authority{
		Name: "racine-test", Type: auth.AuthorityTLS, PublicIdentity: chaintest.PEM(pki.Root),
	}).Run())

	t.Run("the certificate alone, its intermediate missing", func(t *testing.T) {
		rec := s.post(t, permWrite, s.path(srv, "/auth/new"), form("seul", chaintest.PEM(pki.Leaf)))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(),
			"« 127.0.0.1 » est signé par « Test intermediate », que la passerelle ne connaît pas")
		assert.Contains(t, rec.Body.String(), "Déposez le certificat intermédiaire de « Test intermediate »")
	})

	t.Run("a foreign certificate is refused and named", func(t *testing.T) {
		rec := s.post(t, permWrite, s.path(srv, "/auth/new"), form("etranger", chaintest.PEM(pki.Leaf, pki.Inter, pki.Stranger)))
		require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
		assert.Contains(t, rec.Body.String(), "Le certificat « Stranger » n&#39;appartient pas à cette chaîne")
	})

	t.Run("pasted in any order with the root: stored in order, the root set apart", func(t *testing.T) {
		rec := s.post(t, permWrite, s.path(srv, "/auth/new"), form("chaine", chaintest.PEM(pki.Root, pki.Inter, pki.Leaf)))
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

		location := rec.Header().Get("Location")
		assert.Contains(t, location, "note=root_dropped")
		assert.Contains(t, location, "root=Test+root")

		creds, err := srv.GetCredentials(s.db, auth.TLSCertificate)
		require.NoError(t, err)
		require.Len(t, creds, 1)
		assert.Equal(t, chaintest.PEM(pki.Leaf, pki.Inter), creds[0].Value, "the certificate, then its intermediate; no root")

		body := s.get(t, permWrite, location).Body.String()
		assert.Contains(t, body, "Moyen « chaine » ajouté")
		assert.Contains(t, body, "La racine « Test root » déposée avec la chaîne n&#39;est pas conservée")
		assert.Contains(t, body, "chaîne de 2 certificats", "the list says the chain")

		body = s.get(t, permWrite, s.path(srv, fmt.Sprintf("/auth/%d", creds[0].ID))).Body.String()
		assert.Contains(t, body, "<dt>Chaîne de certification</dt><dd>chaîne de 2 certificats</dd>")
		assert.Contains(t, body, "<td>Certificat</td><td class=\"w-mono\">127.0.0.1</td><td class=\"w-mono\">Test intermediate</td>")
		assert.Contains(t, body, "<td>Intermédiaire</td><td class=\"w-mono\">Test intermediate</td><td class=\"w-mono\">Test root</td>")
		assert.Contains(t, body, "Chaîne de certification (PEM)</span>")
		assert.Equal(t, 2, strings.Count(body, "-----BEGIN CERTIFICATE-----"), "the chain in full")

		require.NoError(t, s.db.Delete(creds[0]).Run())
	})

	t.Run("two files in the field, the intermediate first", func(t *testing.T) {
		rec := s.uploadChain(t, s.path(srv, "/auth/new"),
			map[string]string{"name": "fichiers", "type": "tls_certificate", "key_pem": pki.LeafKey},
			"cert_file", chaintest.PEM(pki.Inter), chaintest.PEM(pki.Leaf))
		require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
		assert.NotContains(t, rec.Header().Get("Location"), "note=", "nothing beyond the chain")

		creds, err := srv.GetCredentials(s.db, auth.TLSCertificate)
		require.NoError(t, err)
		require.Len(t, creds, 1)
		assert.Equal(t, chaintest.PEM(pki.Leaf, pki.Inter), creds[0].Value)
	})
}
