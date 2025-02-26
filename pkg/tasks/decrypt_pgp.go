package tasks

import (
	"errors"
	"fmt"
	"io"

	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	pgpprofile "github.com/ProtonMail/gopenpgp/v3/profile"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var ErrDecryptNotPGPKey = errors.New("the provided cryptographic key does not contain a PGP private key")

func isPGPPrivateKey(key *model.CryptoKey) bool {
	return key.Type == model.CryptoKeyTypePGPPrivate
}

func (d *decrypt) makePGPDecryptor(cryptoKey *model.CryptoKey) error {
	if !isPGPPrivateKey(cryptoKey) {
		return ErrDecryptNotPGPKey
	}

	pgpKey, err := pgp.NewKeyFromArmored(cryptoKey.Key.String())
	if err != nil {
		return fmt.Errorf("failed to parse PGP decryption key: %w", err)
	}

	d.decrypt = func(src io.Reader, dst io.Writer) error {
		return pgpDecrypt(src, dst, pgpKey)
	}

	return nil
}

func pgpDecrypt(src io.Reader, dst io.Writer, pgpKey *pgp.Key) error {
	builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Decryption()

	decryptHandler, handlerErr := builder.DecryptionKey(pgpKey).New()
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

	return nil
}
