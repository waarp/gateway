package certinfo

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCertificate(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "AB:CD:EF", colons([]byte{0xab, 0xcd, 0xef}))

	info, ok := Certificate(testhelpers.ClientFooCert)
	require.True(t, ok)
	assert.Equal(t, "foo", info.Subject)
	assert.Equal(t, "foo", info.Issuer)
	assert.Len(t, info.Fingerprint, 95, "32 bytes in hex pairs with colons")
	assert.Contains(t, info.Public, "-----BEGIN CERTIFICATE-----")

	echo := func(key string, _ ...any) string { return key }
	assert.Equal(t, StateOK, info.Expiry(func(_ string, _ ...any) string { return "02/01/2006" }, time.Now()).State)
	assert.Equal(t, StateWarn, info.Expiry(echo, info.NotAfter.Add(-24*time.Hour)).State)
	assert.Equal(t, StateKO, info.Expiry(echo, info.NotAfter.Add(time.Hour)).State)

	_, ok = Certificate("garbage")
	assert.False(t, ok)

	var none Info
	assert.Empty(t, none.Expiry(echo, time.Now()).State, "no validity, no state")
}

func TestKey(t *testing.T) {
	t.Parallel()

	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	key, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)

	info := Key(key)
	assert.Equal(t, ssh.KeyAlgoED25519, info.KeyType)
	assert.Equal(t, ssh.FingerprintSHA256(key), info.Fingerprint)
	assert.True(t, info.NotAfter.IsZero())
}
