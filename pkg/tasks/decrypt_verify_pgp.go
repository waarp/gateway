package tasks

import (
	"fmt"
	"io"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	pgpprofile "github.com/ProtonMail/gopenpgp/v3/profile"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const DecryptVerifyMethodPGP = "PGP"

func makePGPVerifyDecryptor(dCryptoKey, vCryptoKey *model.CryptoKey) (decryptVerifyFunc, error) {
	if !isPGPPrivateKey(dCryptoKey) {
		return nil, ErrDecryptNotPGPKey
	}

	if !isPGPPublicKey(vCryptoKey) {
		return nil, ErrVerifyNotPGPKey
	}

	decryptKey, parsErr := pgp.NewKeyFromArmored(dCryptoKey.Key.String())
	if parsErr != nil {
		return nil, fmt.Errorf("failed to parse PGP decryption key: %w", parsErr)
	}

	verifyKey, parsErr := pgp.NewKeyFromArmored(vCryptoKey.Key.String())
	if parsErr != nil {
		return nil, fmt.Errorf("failed to parse PGP verification key: %w", parsErr)
	}

	if verifyKey.IsPrivate() {
		verifyKey, parsErr = verifyKey.ToPublic()
		if parsErr != nil {
			return nil, fmt.Errorf("failed to parse PGP verification key: %w", parsErr)
		}
	}

	return func(src io.Reader, dst io.Writer) error {
		builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Decryption()

		decryptHandler, handlerErr := builder.DecryptionKey(decryptKey).VerificationKey(verifyKey).New()
		if handlerErr != nil {
			return fmt.Errorf("failed to create PGP decryption handler: %w", handlerErr)
		}

		decrypter, decrErr := decryptHandler.DecryptingReader(src, pgp.Armor)
		if decrErr != nil {
			return fmt.Errorf("failed to initialize PGP decrypter: %w", decrErr)
		}

		if _, err := io.Copy(dst, decrypter); err != nil {
			return fmt.Errorf("failed to decrypt file: %w", err)
		}

		if _, err := decrypter.VerifySignature(); err != nil {
			return fmt.Errorf("failed to verify PGP signature: %w", err)
		}

		return nil
	}, nil
}
