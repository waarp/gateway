package model

import (
	"fmt"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	CryptoKeyTypeAES        = "AES"
	CryptoKeyTypeHMAC       = "HMAC"
	CryptoKeyTypePGPPrivate = "PGP-PRIVATE"
	CryptoKeyTypePGPPublic  = "PGP-PUBLIC"
)

func (k *CryptoKey) checkKey() error {
	switch k.Type {
	case CryptoKeyTypeAES:
		return k.checkKeyAES()
	case CryptoKeyTypeHMAC:
		return k.checkKeyHMAC()
	case CryptoKeyTypePGPPrivate:
		return k.checkKeyPGPPrivate()
	case CryptoKeyTypePGPPublic:
		return k.checkKeyPGPPublic()
	default:
		return database.NewValidationErrorf("unknown cryptographic key type %q", k.Type)
	}
}

func (k *CryptoKey) checkKeyAES() error {
	if k.Key == "" {
		return database.NewValidationError("the AES key is missing")
	}

	const (
		aes128KeyLen = 16
		aes192KeyLen = 24
		aes256KeyLen = 32
	)

	switch len(k.Key) {
	case aes128KeyLen, aes192KeyLen, aes256KeyLen:
	default:
		return database.NewValidationError("AES keys must be 16, 24, or 32 bytes long")
	}

	return nil
}

func (k *CryptoKey) checkKeyHMAC() error {
	if k.Key == "" {
		return database.NewValidationError("the HMAC key is missing")
	}

	return nil
}

func (k *CryptoKey) checkKeyPGPPrivate() error {
	privKey, privErr := pgp.NewKeyFromArmored(k.Key.String())
	if privErr != nil {
		return fmt.Errorf("failed to parse PGP private key: %w", privErr)
	}

	if !privKey.IsPrivate() {
		return database.NewValidationError("the given PGP key is not a private key")
	}

	return nil
}

func (k *CryptoKey) checkKeyPGPPublic() error {
	pubKey, err := pgp.NewKeyFromArmored(k.Key.String())
	if err != nil {
		return fmt.Errorf("failed to parse PGP public key: %w", err)
	}

	if pubKey.IsPrivate() {
		return database.NewValidationError("the given PGP key is not a public key")
	}

	return nil
}
