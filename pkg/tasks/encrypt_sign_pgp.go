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
	ErrEncryptSignPGPNoEncryptionKey = errors.New("missing PGP encryption key")
	ErrEncryptSignPGPNoSignatureKey  = errors.New("missing PGP signature key")
	ErrEncryptSignPGPKeyNotFound     = errors.New("PGP key not found")
	ErrEncryptSignPGPNoPrivateKey    = errors.New("PGP key does not contain a private key")
	ErrEncryptSignPGPNoPublicKey     = errors.New("PGP key does not contain a public key")
)

type encryptSignPGP struct {
	KeepOriginal bool   `json:"keepOriginal"`
	OutputFile   string `json:"outputFile"`

	//nolint:tagliatelle //goCamel does not recognize "PGP" as an acronym
	EncryptionPGPKeyName string `json:"encryptionPGPKeyName"`
	//nolint:tagliatelle //goCamel does not recognize "PGP" as an acronym
	SignaturePGPKeyName string `json:"signaturePGPKeyName"`

	encryptKey, signKey *pgp.Key
}

func (e *encryptSignPGP) checkParams(params map[string]string) error {
	if err := utils.JSONConvert(params, e); err != nil {
		return fmt.Errorf("failed to parse the PGP encryption parameters: %w", err)
	}

	if e.EncryptionPGPKeyName == "" {
		return ErrEncryptSignPGPNoEncryptionKey
	}

	if e.SignaturePGPKeyName == "" {
		return ErrEncryptSignPGPNoSignatureKey
	}

	return nil
}

func (e *encryptSignPGP) parseParams(db database.ReadAccess, params map[string]string) error {
	if err := e.checkParams(params); err != nil {
		return err
	}

	var encryptKey model.PGPKey
	if err := db.Get(&encryptKey, "name = ?", e.EncryptionPGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrEncryptSignPGPKeyNotFound, e.EncryptionPGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if encryptKey.PublicKey == "" {
		return fmt.Errorf("%q: %w", encryptKey.Name, ErrEncryptSignPGPNoPublicKey)
	}

	var parseErr1 error
	if e.encryptKey, parseErr1 = pgp.NewKeyFromArmored(encryptKey.PublicKey); parseErr1 != nil {
		return fmt.Errorf("failed to parse PGP encryption key: %w", parseErr1)
	}

	var signKey model.PGPKey
	if err := db.Get(&signKey, "name = ?", e.EncryptionPGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrEncryptSignPGPKeyNotFound, e.EncryptionPGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if signKey.PrivateKey == "" {
		return fmt.Errorf("%q %w", signKey.Name, ErrEncryptSignPGPNoPrivateKey)
	}

	var parseErr2 error
	if e.signKey, parseErr2 = pgp.NewKeyFromArmored(signKey.PrivateKey.String()); parseErr2 != nil {
		return fmt.Errorf("failed to parse PGP signature key: %w", parseErr2)
	}

	return nil
}

func (e *encryptSignPGP) Validate(params map[string]string) error {
	return e.checkParams(params)
}

func (e *encryptSignPGP) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := e.parseParams(db, params); err != nil {
		logger.Error("Failed to parse PGP encryption parameters: %v", err)

		return err
	}

	if err := encryptFile(logger, transCtx, e.KeepOriginal, e.OutputFile, e.encryptAndSign); err != nil {
		return err
	}

	return nil
}

func (e *encryptSignPGP) encryptAndSign(src io.Reader, dst io.Writer) error {
	builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Encryption()

	encryptHandler, handlerErr := builder.Recipient(e.encryptKey).SigningKey(e.signKey).New()
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
}
