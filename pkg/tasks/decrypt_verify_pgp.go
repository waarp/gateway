package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"
	pgp "github.com/ProtonMail/gopenpgp/v3/crypto"
	pgpprofile "github.com/ProtonMail/gopenpgp/v3/profile"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrDecryptVerifyPGPNoDecryptKey = errors.New("missing PGP decryption key")
	ErrDecryptVerifyPGPNoVerifyKey  = errors.New("missing PGP verification key")
	ErrDecryptVerifyPGPKeyNotFound  = errors.New("cryptographic key not found")
	ErrDecryptVerifyPGPNoPrivateKey = errors.New("cryptographic key does not contain a private PGP key")
	ErrDecryptVerifyPGPNoPublicKey  = errors.New("cryptographic key does not contain a public PGP key")
)

type decryptVerifyPGP struct {
	KeepOriginal bool   `json:"keepOriginal"`
	OutputFile   string `json:"outputFile"`

	//nolint:tagliatelle //goCamel does not recognize "PGP" as an acronym
	DecryptionPGPKeyName string `json:"decryptionPGPKeyName"`
	//nolint:tagliatelle //goCamel does not recognize "PGP" as an acronym
	VerificationPGPKeyName string `json:"verificationPGPKeyName"`

	decryptKey, verifyKey *pgp.Key
}

func (d *decryptVerifyPGP) ValidateDB(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, d); err != nil {
		return fmt.Errorf("failed to parse the PGP decryption parameters: %w", err)
	}

	if d.DecryptionPGPKeyName == "" {
		return ErrDecryptVerifyPGPNoDecryptKey
	}

	if d.VerificationPGPKeyName == "" {
		return ErrDecryptVerifyPGPNoVerifyKey
	}

	var decryptKey model.CryptoKey
	if err := db.Get(&decryptKey, "name = ?", d.DecryptionPGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrDecryptVerifyPGPKeyNotFound, d.DecryptionPGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if !isPGPPrivateKey(&decryptKey) {
		return fmt.Errorf("%q: %w", decryptKey.Name, ErrDecryptVerifyPGPNoPrivateKey)
	}

	var parseErr1 error
	if d.decryptKey, parseErr1 = pgp.NewKeyFromArmored(decryptKey.Key.String()); parseErr1 != nil {
		return fmt.Errorf("failed to parse PGP decryption key: %w", parseErr1)
	}

	var verifyKey model.CryptoKey
	if err := db.Get(&verifyKey, "name = ?", d.VerificationPGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrDecryptVerifyPGPKeyNotFound, d.VerificationPGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if !isPGPPrivateKey(&decryptKey) && !isPGPPublicKey(&verifyKey) {
		return fmt.Errorf("%q: %w", verifyKey.Name, ErrDecryptVerifyPGPNoPublicKey)
	}

	var err error
	if d.verifyKey, err = pgp.NewKeyFromArmored(verifyKey.Key.String()); err != nil {
		return fmt.Errorf("failed to parse PGP verification key: %w", err)
	}

	if d.verifyKey.IsPrivate() {
		if d.verifyKey, err = d.verifyKey.ToPublic(); err != nil {
			return fmt.Errorf("failed to parse PGP verification key: %w", err)
		}
	}

	return nil
}

func (d *decryptVerifyPGP) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := d.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	if err := decryptFile(logger, transCtx, d.KeepOriginal, d.OutputFile, d.decrypt); err != nil {
		return err
	}

	return nil
}

func (d *decryptVerifyPGP) decrypt(src io.Reader, dst io.Writer) error {
	builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Decryption()

	decryptHandler, handlerErr := builder.DecryptionKey(d.decryptKey).VerificationKey(d.verifyKey).New()
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
}
