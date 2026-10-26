package partners

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/certchain/chaintest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// TestTrustedCertificateChain: a trusted certificate of a partner given
// with its chain keeps the certificate alone (the trust bears on it,
// strictly), and the form says it (#776).
func TestTrustedCertificateChain(t *testing.T) {
	t.Parallel()

	s := newSuite(t)
	pki := chaintest.New(t)

	partner := &model.RemoteAgent{Name: "part-chaine", Protocol: "https", Address: types.Addr("127.0.0.1", 8445)}
	require.NoError(t, s.db.Insert(partner).Run())

	rec := s.post(t, permWrite, s.path(partner, "/auth/new"), url.Values{
		"name": {"confiance"}, "type": {"trusted_tls_certificate"}, "pem": {chaintest.PEM(pki.Root, pki.Inter, pki.Leaf)},
	})
	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())

	location := rec.Header().Get("Location")
	assert.Contains(t, location, "note=trusted_leaf_only")

	creds, err := partner.GetCredentials(s.db, auth.TLSTrustedCertificate)
	require.NoError(t, err)
	require.Len(t, creds, 1)
	assert.Equal(t, chaintest.PEM(pki.Leaf), creds[0].Value, "the certificate alone")

	body := s.get(t, permWrite, location).Body.String()
	assert.Contains(t, body, "Moyen « confiance » ajouté")
	assert.Contains(t, body, "La confiance porte sur le certificat seul")
	assert.NotContains(t, body, "chaîne de", "no chain kept")

	// A certificate alone, as before: no note.
	rec = s.post(t, permWrite, s.path(partner, "/auth/new"), url.Values{
		"name": {"seul"}, "type": {"trusted_tls_certificate"}, "pem": {chaintest.PEM(pki.Leaf)},
	})
	require.Equal(t, http.StatusSeeOther, rec.Code, rec.Body.String())
	assert.NotContains(t, rec.Header().Get("Location"), "note=")
}
