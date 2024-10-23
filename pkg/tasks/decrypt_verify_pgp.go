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
	ErrDecryptVerifyPGPKeyNotFound  = errors.New("PGP key not found")
	ErrDecryptVerifyPGPNoPrivateKey = errors.New("PGP key does not contain a private key")
	ErrDecryptVerifyPGPNoPublicKey  = errors.New("PGP key does not contain a public key")
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

func (d *decryptVerifyPGP) checkParams(params map[string]string) error {
	if err := utils.JSONConvert(params, d); err != nil {
		return fmt.Errorf("failed to parse the PGP decryption parameters: %w", err)
	}

	if d.DecryptionPGPKeyName == "" {
		return ErrDecryptVerifyPGPNoDecryptKey
	}

	if d.VerificationPGPKeyName == "" {
		return ErrDecryptVerifyPGPNoVerifyKey
	}

	return nil
}

func (d *decryptVerifyPGP) parseParams(db database.ReadAccess, params map[string]string) error {
	if err := d.checkParams(params); err != nil {
		return err
	}

	var decryptKey model.PGPKey
	if err := db.Get(&decryptKey, "name = ?", d.DecryptionPGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrDecryptVerifyPGPKeyNotFound, d.DecryptionPGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if decryptKey.PrivateKey == "" {
		return fmt.Errorf("%q: %w", decryptKey.Name, ErrDecryptVerifyPGPNoPrivateKey)
	}

	var parseErr1 error
	if d.decryptKey, parseErr1 = pgp.NewKeyFromArmored(decryptKey.PrivateKey.String()); parseErr1 != nil {
		return fmt.Errorf("failed to parse PGP decryption key: %w", parseErr1)
	}

	var verifyKey model.PGPKey
	if err := db.Get(&verifyKey, "name = ?", d.VerificationPGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrDecryptVerifyPGPKeyNotFound, d.VerificationPGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if verifyKey.PublicKey == "" {
		return fmt.Errorf("%q: %w", verifyKey.Name, ErrDecryptVerifyPGPNoPublicKey)
	}

	var err error
	if d.verifyKey, err = pgp.NewKeyFromArmored(verifyKey.PublicKey); err != nil {
		return fmt.Errorf("failed to parse PGP verification key: %w", err)
	}

	return nil
}

func (d *decryptVerifyPGP) Validate(params map[string]string) error {
	return d.checkParams(params)
}

func (d *decryptVerifyPGP) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := d.parseParams(db, params); err != nil {
		logger.Error("Failed to parse PGP decryption parameters: %v", err)

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
