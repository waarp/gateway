package means

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/pesit"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// Every kind the tables offer is known to the model for the owner, in the
// registry the owner belongs to.
func TestTypesKnownToModel(t *testing.T) {
	t.Parallel()

	for proto := range protocols.List() {
		for _, typ := range ServerTypes(proto) {
			assert.NotNilf(t, authentication.GetExternalAuthMethod(typ, proto), "%s on a %s server", typ, proto)
		}

		for _, typ := range LocalAccountTypes(proto) {
			assert.NotNilf(t, authentication.GetInternalAuthHandler(typ, proto), "%s on a %s account", typ, proto)
		}

		assert.Contains(t, LocalAccountTypes(proto), auth.Password, "a password is always possible on an account")
	}

	for proto := range protocols.List() {
		for _, typ := range PartnerTypes(proto) {
			assert.NotNilf(t, authentication.GetInternalAuthHandler(typ, proto), "%s on a %s partner", typ, proto)
		}

		for _, typ := range RemoteAccountTypes(proto) {
			assert.NotNilf(t, authentication.GetExternalAuthMethod(typ, proto), "%s on a %s remote account", typ, proto)
		}

		assert.Contains(t, RemoteAccountTypes(proto), auth.Password, "a password is always possible on a remote account")
	}

	assert.Equal(t, []string{auth.Password, pesit.PreConnectionAuth}, RemoteAccountTypes("pesit"))
	assert.Equal(t, []string{auth.Password, auth.TLSCertificate, pesit.PreConnectionAuth}, RemoteAccountTypes("pesit-tls"),
		"the password of a PeSIT over TLS account is sent whatever the encryption")
	assert.Equal(t, []string{auth.Password, sftp.AuthSSHPrivateKey}, RemoteAccountTypes("sftp"))
	assert.Equal(t, familyPair, FamilyOf(pesit.PreConnectionAuth))

	assert.Equal(t, []string{auth.Password}, PartnerTypes("pesit"), "the client checks the password of a PeSIT partner")
	assert.Equal(t, []string{auth.Password, auth.TLSTrustedCertificate}, PartnerTypes("pesit-tls"))
	assert.Equal(t, []string{sftp.AuthSSHPublicKey}, PartnerTypes("sftp"), "the host key of the SFTP server")
	assert.Empty(t, PartnerTypes("http"))

	assert.Equal(t, []string{sftp.AuthSSHPrivateKey}, ServerTypes("sftp"))
	assert.Empty(t, ServerTypes("http"), "a plain HTTP server presents nothing")
	assert.Equal(t, []string{auth.Password, sftp.AuthSSHPublicKey}, LocalAccountTypes("sftp"))
	assert.Equal(t, []string{auth.Password, auth.TLSTrustedCertificate}, LocalAccountTypes("pesit-tls"))

	srv := &Owner{Protocol: "sftp", Allowed: ServerTypes("sftp")}
	acc := &Owner{Protocol: "sftp", Verified: true, Allowed: LocalAccountTypes("sftp")}
	assert.True(t, acc.Unique(auth.Password), "one password per account")
	assert.False(t, acc.Unique(sftp.AuthSSHPublicKey), "several public keys per account")
	assert.False(t, srv.Unique(sftp.AuthSSHPrivateKey))

	pwd := &model.Credential{Type: auth.Password}
	assert.Equal(t, []string{sftp.AuthSSHPublicKey}, acc.Remaining([]*model.Credential{pwd}), "a unique kind in place is no longer offered")
	assert.Equal(t, LocalAccountTypes("sftp"), acc.Remaining(nil))
}

// A means is shown by what identifies it, never by its secret.
func TestView(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "AB:CD:EF", colons([]byte{0xab, 0xcd, 0xef}))

	v := viewOf(&model.Credential{Type: auth.TLSTrustedCertificate, Value: testhelpers.ClientFooCert})
	assert.Equal(t, "foo", v.Subject)
	assert.Equal(t, "foo", v.Issuer)
	assert.Len(t, v.Fingerprint, 95, "32 bytes in hex pairs with colons")
	assert.Equal(t, stateOK, v.expiry(func(_ string, _ ...any) string { return "02/01/2006" }, time.Now()).State)
	assert.Equal(t, stateWarn, v.expiry(func(key string, _ ...any) string { return key }, v.NotAfter.Add(-24*time.Hour)).State)
	assert.Equal(t, stateKO, v.expiry(func(key string, _ ...any) string { return key }, v.NotAfter.Add(time.Hour)).State)

	broken := viewOf(&model.Credential{Type: auth.TLSCertificate, Value: "garbage"})
	assert.Empty(t, broken.Subject)
	assert.Empty(t, broken.expiry(func(key string, _ ...any) string { return key }, time.Now()).State)

	echo := func(key string, _ ...any) string { return key }
	assert.Equal(t, "auth.content.secret", Summary(echo, &model.Credential{Type: auth.Password, Value: "s3cret"}))
	assert.Equal(t, "auth.content.set", Summary(echo, &model.Credential{Type: auth.TLSCertificate, Value: "garbage"}))

	pre := viewOf(&model.Credential{Type: pesit.PreConnectionAuth, Value: "GWLOGIN", Value2: "pa55"})
	assert.Equal(t, "GWLOGIN", pre.Login)
	assert.True(t, pre.HasPassword)
	assert.Equal(t, "auth.content.pre_login · auth.content.pre_password", pre.summary(echo))
	none := viewOf(&model.Credential{Type: pesit.PreConnectionAuth})
	assert.Equal(t, "auth.content.pre_no_login · auth.content.pre_no_password", none.summary(echo), "both parts are optional")
}
