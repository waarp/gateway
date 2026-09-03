package auth

import (
	"crypto/x509"
	"net/http"
	"strings"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestVerifyTLSAuthorityCert(t *testing.T) {
	t.Parallel()

	handler := TLSAuthorityHandler{}

	t.Run("Valid cert", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, handler.Validate(rootCertPEM))
	})

	t.Run("Invalid cert", func(t *testing.T) {
		t.Parallel()

		require.ErrorIs(t, handler.Validate(leafCertPEM), ErrNotCACertificate)
	})
}

func TestVerifyTrustedCert(t *testing.T) {
	t.Parallel()

	validate := func(pem, host string) error {
		return TLSTrustedCertHandler{}.Validate(nil, pem, "", "", host, true)
	}

	t.Run("Valid cert", func(t *testing.T) {
		t.Parallel()
		require.NoError(t, validate(leafCertPEM, "waarp-gateway.test"))
	})

	t.Run("Invalid host", func(t *testing.T) {
		t.Parallel()
		var err x509.HostnameError
		require.ErrorAs(t, validate(leafCertPEM, "invalid.host"), &err)
	})

	t.Run("Invalid usage", func(t *testing.T) {
		t.Parallel()
		var err x509.CertificateInvalidError
		require.ErrorAs(t, validate(clientCertPEM, ""), &err)
		assert.Equal(t, x509.IncompatibleUsage, err.Reason)
	})
}

func TestVerifyTLSCert(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	authority := &model.Authority{
		Name:           "test_root_ca",
		Type:           AuthorityTLS,
		PublicIdentity: rootCertPEM,
		ValidHosts:     []string{"localhost", "waarp-gateway.test"},
	}
	require.NoError(t, db.Insert(authority).Run())

	serverCert := &model.Authority{
		Name:           "server_cert",
		Type:           AuthorityTLS,
		PublicIdentity: serverCertPEM,
	}
	require.NoError(t, db.Insert(serverCert).Run())

	clientCert := &model.Authority{
		Name:           "client_cert",
		Type:           AuthorityTLS,
		PublicIdentity: clientCertPEM,
	}
	require.NoError(t, db.Insert(clientCert).Run())

	validate := func(cert, key, host string) error {
		return TLSCertHandler{}.Validate(db, cert, key, "", host, true)
	}

	joinCerts := func(pems ...string) string {
		return strings.Join(pems, "\n")
	}

	t.Run("Valid chain", func(t *testing.T) {
		t.Parallel()
		chain := joinCerts(leafCertPEM, intermediateCertPEM)

		require.NoError(t, validate(chain, leafKeyPEM, "localhost"))
	})

	t.Run("Valid self-signed", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, validate(testhelpers.OtherLocalhostCert,
			testhelpers.OtherLocalhostKey, "localhost"))
	})

	t.Run("Invalid host", func(t *testing.T) {
		t.Parallel()
		chain := joinCerts(leafCertPEM, intermediateCertPEM)

		var err x509.HostnameError
		require.ErrorAs(t, validate(chain, leafKeyPEM, "invalid.host"), &err)
	})

	t.Run("Missing intermediate", func(t *testing.T) {
		t.Parallel()
		chain := joinCerts(leafCertPEM)

		var err x509.UnknownAuthorityError
		require.ErrorAs(t, validate(chain, leafKeyPEM, "localhost"), &err)
	})

	t.Run("Invalid key", func(t *testing.T) {
		t.Parallel()
		chain := joinCerts(leafCertPEM, intermediateCertPEM)

		err := validate(chain, intermediateKeyPEM, "localhost")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "private key does not match public key")
	})

	t.Run("Invalid usage", func(t *testing.T) {
		t.Parallel()

		var err x509.CertificateInvalidError
		require.ErrorAs(t, validate(clientCertPEM, clientPKeyPEM, ""), &err)
		assert.Equal(t, x509.IncompatibleUsage, err.Reason)
	})
}

func TestAuthentCert(t *testing.T) {
	t.Parallel()

	const protocol = "https"

	modeltest.AddDummyProtoConfig(protocol)
	db := dbtest.TestDatabase(t)

	handler := TLSTrustedCertHandler{}

	joinCerts := func(pems ...string) string {
		return strings.Join(pems, "\n")
	}

	authority := &model.Authority{
		Name:           "test_root_ca",
		Type:           AuthorityTLS,
		PublicIdentity: rootCertPEM,
	}
	require.NoError(t, db.Insert(authority).Run())

	partner := &model.RemoteAgent{
		Name:     "test_chain",
		Protocol: protocol,
		Address:  types.Addr("localhost", 443),
	}
	require.NoError(t, db.Insert(partner).Run())

	server := &model.LocalAgent{
		Name:     "test_server",
		Protocol: protocol,
		Address:  types.Addr("localhost", 1234),
	}
	require.NoError(t, db.Insert(server).Run())

	localAccount := &model.LocalAccount{LocalAgentID: server.ID, Login: "test_login"}
	require.NoError(t, db.Insert(localAccount).Run())

	t.Run("Valid public", func(t *testing.T) {
		t.Parallel()

		publicPartner := &model.RemoteAgent{
			Name:     "waarp.fr",
			Protocol: protocol,
			Address:  types.Addr("waarp.fr", 443),
		}
		require.NoError(t, db.Insert(publicPartner).Run())

		resp, err := http.Get("https://waarp.fr")
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())

		result, err := handler.Authenticate(db, publicPartner, resp.TLS.PeerCertificates)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Empty(t, result.Reason)
	})

	t.Run("Valid from authority", func(t *testing.T) {
		t.Parallel()

		pemChain := joinCerts(leafCertPEM, intermediateCertPEM)
		chain, err := utils.ParsePEMCertChain(pemChain)
		require.NoError(t, err)

		result, err := handler.Authenticate(db, partner, chain)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Empty(t, result.Reason)
	})

	t.Run("Valid self-signed", func(t *testing.T) {
		t.Parallel()

		cred := &model.Credential{
			RemoteAgentID: partner.NullableID(),
			Name:          "self_signed_certificate",
			Type:          TLSTrustedCertificate,
			Value:         serverCertPEM,
		}
		require.NoError(t, db.Insert(cred).Run())

		chain, err := utils.ParsePEMCertChain(serverCertPEM)
		require.NoError(t, err)

		result, err := handler.Authenticate(db, partner, chain)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Empty(t, result.Reason)
	})

	t.Run("Know client cert", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, db.Insert(&model.Credential{
			LocalAccountID: localAccount.NullableID(),
			Type:           TLSTrustedCertificate,
			Value:          leafCertPEM,
		}).Run())

		pemChain := joinCerts(leafCertPEM, intermediateCertPEM, rootCertPEM)
		chain, err := utils.ParsePEMCertChain(pemChain)
		require.NoError(t, err)

		result, err := handler.Authenticate(db, localAccount, chain)
		require.NoError(t, err)
		assert.True(t, result.Success)
		assert.Empty(t, result.Reason)
	})

	t.Run("Unknown client cert", func(t *testing.T) {
		t.Parallel()

		chain, err := utils.ParsePEMCertChain(intermediateCertPEM)
		require.NoError(t, err)

		result, err := handler.Authenticate(db, localAccount, chain)
		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Reason, "unknown client certificate")
	})

	t.Run("Unknown authority", func(t *testing.T) {
		t.Parallel()

		chain, err := utils.ParsePEMCertChain(leafCertPEM)
		require.NoError(t, err)

		result, err := handler.Authenticate(db, partner, chain)
		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.Contains(t, result.Reason, "certificate signed by unknown authority")
	})
}
