package tasks

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	pgpprofile "github.com/ProtonMail/gopenpgp/v3/profile"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var ErrVerifyNotPGPKey = errors.New("the provided cryptographic key does not contain a PGP key")

func isPGPVerifyKey(cryptoKey *model.CryptoKey) bool {
	return cryptoKey.Type == model.CryptoKeyTypePGPPublic || cryptoKey.Type == model.CryptoKeyTypePGPPrivate
}

func (v *verify) makePGPVerifier(cryptoKey *model.CryptoKey) error {
	if !isPGPVerifyKey(cryptoKey) {
		return ErrVerifyNotPGPKey
	}

	pgpKey, parsErr := pgp.NewKeyFromArmored(cryptoKey.Key.String())
	if parsErr != nil {
		return fmt.Errorf("failed to parse PGP verification key: %w", parsErr)
	}

	if pgpKey.IsPrivate() {
		pgpKey, parsErr = pgpKey.ToPublic()
		if parsErr != nil {
			return fmt.Errorf("failed to parse PGP public key: %w", parsErr)
		}
	}

	v.verify = func(file io.Reader, expected []byte) error {
		builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Verify()
		sigReader := bytes.NewReader(expected)

		verifyHandler, handlerErr := builder.VerificationKey(pgpKey).New()
		if handlerErr != nil {
			return fmt.Errorf("failed to create PGP verification handler: %w", handlerErr)
		}

		verifier, verifyErr := verifyHandler.VerifyingReader(file, sigReader, pgp.Auto)
		if verifyErr != nil {
			return fmt.Errorf("failed to initialize PGP verifier: %w", verifyErr)
		}

		if _, err := verifier.DiscardAllAndVerifySignature(); err != nil {
			return fmt.Errorf("failed to sign file signature: %w", err)
		}

		return nil
	}

	return nil
}
