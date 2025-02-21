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
	ErrEncryptPGPNoKeyName   = errors.New("missing PGP encryption key")
	ErrEncryptPGPKeyNotFound = errors.New("cryptographic key not found")
	ErrEncryptPGPNoPublicKey = errors.New("cryptographic key does not contain a public PGP key")
)

type encryptPGP struct {
	KeepOriginal bool   `json:"keepOriginal"`
	OutputFile   string `json:"outputFile"`
	PGPKeyName   string `json:"pgpKeyName"`

	encryptKey *pgp.Key
}

func (e *encryptPGP) ValidateDB(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, e); err != nil {
		return fmt.Errorf("failed to parse the PGP encryption parameters: %w", err)
	}

	if e.PGPKeyName == "" {
		return ErrEncryptPGPNoKeyName
	}

	var pgpKey model.CryptoKey
	if err := db.Get(&pgpKey, "name = ?", e.PGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrEncryptPGPKeyNotFound, e.PGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if !isPGPPrivateKey(&pgpKey) && !isPGPPublicKey(&pgpKey) {
		return fmt.Errorf("%q: %w", pgpKey.Name, ErrEncryptPGPNoPublicKey)
	}

	var err error
	if e.encryptKey, err = pgp.NewKeyFromArmored(pgpKey.Key.String()); err != nil {
		return fmt.Errorf("failed to parse PGP encryption key: %w", err)
	}

	if e.encryptKey.IsPrivate() {
		if e.encryptKey, err = e.encryptKey.ToPublic(); err != nil {
			return fmt.Errorf("failed to parse PGP encryption key: %w", err)
		}
	}

	return nil
}

func (e *encryptPGP) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := e.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	if err := encryptFile(logger, transCtx, e.KeepOriginal, e.OutputFile, e.encrypt); err != nil {
		return err
	}

	return nil
}

func (e *encryptPGP) encrypt(src io.Reader, dst io.Writer) error {
	builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Encryption()

	encryptHandler, handlerErr := builder.Recipient(e.encryptKey).New()
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

func isPGPPrivateKey(key *model.CryptoKey) bool {
	return key.Type == model.CryptoKeyTypePGPPrivate
}

func isPGPPublicKey(key *model.CryptoKey) bool {
	return key.Type == model.CryptoKeyTypePGPPublic
}
