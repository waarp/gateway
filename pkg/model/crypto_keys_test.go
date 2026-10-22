package model

import (
	"fmt"
	"testing"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCryptoKeyBeforeWrite(t *testing.T) {
	t.Parallel()
	db := dbtest.TestDatabase(t)

	const aesKey = "0123456789abcdefhijklABCDEFHIJKLMNOPQRSTUVW="

	existing := &CryptoKey{
		Name: "existing",
		Type: CryptoKeyTypeAES,
		Key:  aesKey,
	}
	require.NoError(t, db.Insert(existing).Run())

	t.Run("AES key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: CryptoKeyTypeAES,
			Key:  aesKey,
		}
		require.NoError(t, key.BeforeWrite(db))
	})

	t.Run("HMAC key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "hmac-key",
			Type: CryptoKeyTypeHMAC,
			Key:  aesKey,
		}
		require.NoError(t, key.BeforeWrite(db))
	})

	t.Run("PGP public key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "hmac-key",
			Type: CryptoKeyTypePGPPublic,
			Key:  testhelpers.TestPGPPublicKey,
		}
		require.NoError(t, key.BeforeWrite(db))
	})

	t.Run("PGP private key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "hmac-key",
			Type: CryptoKeyTypePGPPrivate,
			Key:  testhelpers.TestPGPPrivateKey,
		}
		require.NoError(t, key.BeforeWrite(db))
	})

	t.Run("No name", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Type: CryptoKeyTypeAES,
			Key:  aesKey,
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"the cryptographic key's name is missing")
	})

	t.Run("No type", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Key:  aesKey,
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"the cryptographic key's type is missing")
	})

	t.Run("No value", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: CryptoKeyTypeAES,
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"the cryptographic key value is missing")
	})

	t.Run("Duplicate name", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: existing.Name,
			Type: CryptoKeyTypeAES,
			Key:  aesKey,
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			fmt.Sprintf("a cryptographic key named %q already exists", key.Name))
	})

	t.Run("Unknown type", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: "not a type",
			Key:  "0123456789abcdefhijklABCDEFHIJKL",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			fmt.Sprintf("unknown cryptographic key type %q", key.Type))
	})

	t.Run("Not base64 AES value", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: CryptoKeyTypeAES,
			Key:  "not base64",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"failed to decode AES key")
	})

	t.Run("Wrong length AES value", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: CryptoKeyTypeAES,
			Key:  "0123456789abcdefhijklABCDEFHIJKLMNOPQRST",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"AES keys must be 16, 24, or 32 bytes long")
	})

	t.Run("Not base64 HMAC value", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: CryptoKeyTypeHMAC,
			Key:  "not base64",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"failed to decode HMAC key")
	})

	t.Run("Illegal PGP public key value", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "pgp-key",
			Type: CryptoKeyTypePGPPublic,
			Key:  "not a PGP key",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"failed to parse PGP public key")
	})

	t.Run("Illegal PGP private key value", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "pgp-key",
			Type: CryptoKeyTypePGPPrivate,
			Key:  "not a PGP key",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"failed to parse PGP private key")
	})

	t.Run("PGP private key in public key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "pgp-key",
			Type: CryptoKeyTypePGPPublic,
			Key:  testhelpers.TestPGPPrivateKey,
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"the given PGP key is not a public key")
	})

	t.Run("PGP public key in private key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "pgp-key",
			Type: CryptoKeyTypePGPPrivate,
			Key:  testhelpers.TestPGPPublicKey,
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"the given PGP key is not a private key")
	})

	t.Run("Locked PGP key", func(t *testing.T) {
		t.Parallel()

		pgpKey, err := pgp.NewKeyFromArmored(testhelpers.TestPGPPrivateKey)
		require.NoError(t, err)
		require.NoError(t, pgpKey.GetEntity().EncryptPrivateKeys([]byte("sesame"), nil))

		lockedKey, err := pgpKey.Armor()
		require.NoError(t, err)

		key := &CryptoKey{
			Name: "pgp-key",
			Type: CryptoKeyTypePGPPrivate,
			Key:  lockedKey,
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"the given PGP key is encrypted")
	})
}
