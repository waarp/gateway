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
	ErrDecryptPGPNoKeyName    = errors.New("missing PGP decryption key")
	ErrDecryptPGPKeyNotFound  = errors.New("cryptographic key not found")
	ErrDecryptPGPNoPrivateKey = errors.New("cryptographic key does not contain a private PGP key")
)

type decryptPGP struct {
	KeepOriginal bool   `json:"keepOriginal"`
	OutputFile   string `json:"outputFile"`
	PGPKeyName   string `json:"pgpKeyName"`

	signKey *pgp.Key
}

func (d *decryptPGP) ValidateDB(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, d); err != nil {
		return fmt.Errorf("failed to parse the PGP decryption parameters: %w", err)
	}

	if d.PGPKeyName == "" {
		return ErrDecryptPGPNoKeyName
	}

	var pgpKey model.CryptoKey
	if err := db.Get(&pgpKey, "name = ?", d.PGPKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w: %q", ErrDecryptPGPKeyNotFound, d.PGPKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve PGP key from database: %w", err)
	}

	if !isPGPPrivateKey(&pgpKey) {
		return fmt.Errorf("%q: %w", pgpKey.Name, ErrDecryptPGPNoPrivateKey)
	}

	var err error
	if d.signKey, err = pgp.NewKeyFromArmored(pgpKey.Key.String()); err != nil {
		return fmt.Errorf("failed to parse PGP decryption key: %w", err)
	}

	return nil
}

func (d *decryptPGP) Run(_ context.Context, params map[string]string,
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

func (d *decryptPGP) decrypt(src io.Reader, dst io.Writer) error {
	builder := pgp.PGPWithProfile(pgpprofile.RFC4880()).Decryption()

	decryptHandler, handlerErr := builder.DecryptionKey(d.signKey).New()
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
