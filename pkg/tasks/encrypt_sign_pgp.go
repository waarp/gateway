package tasks

import (
	"fmt"
	"io"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	pgpprofile "github.com/ProtonMail/gopenpgp/v3/profile"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const EncryptSignMethodPGP = "PGP"

func makePGPSignEncryptor(eCryptoKey, sCryptoKey *model.CryptoKey) (encryptSignFunc, error) {
	if !isPGPPublicKey(eCryptoKey) {
		return nil, ErrEncryptNotPGPKey
	}

	if !isPGPPrivateKey(sCryptoKey) {
		return nil, ErrSignNotPGPKey
	}

	encryptKey, parsErr := pgp.NewKeyFromArmored(eCryptoKey.Key.String())
	if parsErr != nil {
		return nil, fmt.Errorf("failed to parse PGP encryption key: %w", parsErr)
	}

	if encryptKey.IsPrivate() {
		if encryptKey, parsErr = encryptKey.ToPublic(); parsErr != nil {
			return nil, fmt.Errorf("failed to parse PGP encryption key: %w", parsErr)
		}
	}

	signKey, parsErr := pgp.NewKeyFromArmored(sCryptoKey.Key.String())
	if parsErr != nil {
		return nil, fmt.Errorf("failed to parse PGP signing key: %w", parsErr)
	}

	return func(src io.Reader, dst io.Writer) error {
		builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Encryption()

		encryptHandler, handlerErr := builder.Recipient(encryptKey).SigningKey(signKey).New()
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
