package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestCryptoKeyBeforeWrite(t *testing.T) {
	t.Parallel()
	db := dbtest.TestDatabase(t)

	existing := &CryptoKey{
		Name: "existing",
		Type: CryptoKeyTypeAES,
		Key:  "0123456789abcdefhijklABCDEFHIJKL",
	}
	require.NoError(t, db.Insert(existing).Run())

	t.Run("AES key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: CryptoKeyTypeAES,
			Key:  "0123456789abcdefhijklABCDEFHIJKL",
		}
		require.NoError(t, key.BeforeWrite(db))
	})

	t.Run("HMAC key", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "hmac-key",
			Type: CryptoKeyTypeHMAC,
			Key:  "0123456789abcdefhijklABCDEFHIJKL",
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
			Key:  "0123456789abcdefhijklABCDEFHIJKL",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"the cryptographic key's name is missing")
	})

	t.Run("No type", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Key:  "0123456789abcdefhijklABCDEFHIJKL",
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
			Key:  "0123456789abcdefhijklABCDEFHIJKL",
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

	t.Run("Illegal AES value", func(t *testing.T) {
		t.Parallel()

		key := &CryptoKey{
			Name: "aes-key",
			Type: CryptoKeyTypeAES,
			Key:  "0123456789abcdefhijklABCDEFHIJK",
		}
		require.ErrorContains(t, key.BeforeWrite(db),
			"AES keys must be 16, 24, or 32 bytes long")
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
}
