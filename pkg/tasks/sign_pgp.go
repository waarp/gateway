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

const SignMethodPGP = "PGP"

var ErrSignNotPGPKey = errors.New("the provided cryptographic key does not contain a PGP key")

func isPGPSignKey(cryptoKey *model.CryptoKey) bool {
	return cryptoKey.Type == model.CryptoKeyTypePGPPrivate
}

func makePGPSigner(cryptoKey *model.CryptoKey) (signFunc, error) {
	if !isPGPSignKey(cryptoKey) {
		return nil, ErrSignNotPGPKey
	}

	pgpKey, parsErr := pgp.NewKeyFromArmored(cryptoKey.Key.String())
	if parsErr != nil {
		return nil, fmt.Errorf("failed to parse PGP signing key: %w", parsErr)
	}

	return func(file io.Reader) ([]byte, error) {
		builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Sign().Detached()

		signHandler, handlerErr := builder.SigningKey(pgpKey).New()
		if handlerErr != nil {
			return nil, fmt.Errorf("failed to create PGP signature handler: %w", handlerErr)
		}

		signature := &bytes.Buffer{}

		signer, signErr := signHandler.SigningWriter(signature, pgp.Armor)
		if signErr != nil {
			return nil, fmt.Errorf("failed to initialize PGP signer: %w", signErr)
		}

		if _, err := io.Copy(signer, file); err != nil {
			return nil, fmt.Errorf("failed to sign file: %w", err)
		}

		if err := signer.Close(); err != nil {
			return nil, fmt.Errorf("failed to sign file: %w", err)
		}

		return signature.Bytes(), nil
	}, nil
}
