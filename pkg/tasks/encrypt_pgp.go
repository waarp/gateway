package tasks

import (
	"errors"
	"fmt"
	"io"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	pgpprofile "github.com/ProtonMail/gopenpgp/v3/profile"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const EncryptMethodPGP = "PGP"

var ErrEncryptNotPGPKey = errors.New("the provided cryptographic key does not contain a PGP public key")

func isPGPPublicKey(key *model.CryptoKey) bool {
	return key.Type == model.CryptoKeyTypePGPPublic || key.Type == model.CryptoKeyTypePGPPrivate
}

func makePGPEncryptor(cryptoKey *model.CryptoKey) (encryptFunc, error) {
	if !isPGPPublicKey(cryptoKey) {
		return nil, ErrEncryptNotPGPKey
	}

	pgpKey, err := pgp.NewKeyFromArmored(cryptoKey.Key.String())
	if err != nil {
		return nil, fmt.Errorf("failed to parse PGP encryption key: %w", err)
	}

	if pgpKey.IsPrivate() {
		if pgpKey, err = pgpKey.ToPublic(); err != nil {
			return nil, fmt.Errorf("failed to parse PGP encryption key: %w", err)
		}
	}

	return func(src io.Reader, dst io.Writer) error {
		builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Encryption()

		encryptHandler, handlerErr := builder.Recipient(pgpKey).New()
		if handlerErr != nil {
			return fmt.Errorf("failed to create PGP encryption handler: %w", handlerErr)
		}

		encrypter, encrErr := encryptHandler.EncryptingWriter(dst, pgp.Armor)
		if encrErr != nil {
			return fmt.Errorf("failed to initialize PGP encrypter: %w", encrErr)
		}

		if _, err := io.Copy(encrypter, src); err != nil {
			return fmt.Errorf("failed to encrypt file: %w", err)
		}

		if err := encrypter.Close(); err != nil {
			return fmt.Errorf("failed to encrypt file: %w", err)
		}

		return nil
	}, nil
}
