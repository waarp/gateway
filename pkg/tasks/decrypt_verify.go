//nolint:dupl // keep tasks separate in case they change in the future
package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrDecryptVerifyNoDecryptionKeyName   = errors.New("missing decryption key name")
	ErrDecryptVerifyNoVerificationKeyName = errors.New("missing verification key name")
	ErrDecryptVerifyNoMethod              = errors.New("missing decryption/verification method")
	ErrDecryptVerifyKeyNotFound           = errors.New("cryptographic key not found")
	ErrDecryptVerifyInvalidMethod         = errors.New("invalid decryption/verification method")
)

type decryptVerifyFunc func(src io.Reader, dst io.Writer) error

type decryptVerify struct {
	KeepOriginal jsonBool `json:"keepOriginal"`
	OutputFile   string   `json:"outputFile"`

	Method         string `json:"method"`
	DecryptKeyName string `json:"decryptKeyName"`
	VerifyKeyName  string `json:"verifyKeyName"`

	decryptVerify decryptVerifyFunc
}

func (d *decryptVerify) ValidateDB(db database.ReadAccess, params map[string]string) error {
	*d = decryptVerify{}
	if err := utils.JSONConvert(params, d); err != nil {
		return fmt.Errorf("failed to parse the encryption parameters: %w", err)
	}

	if d.DecryptKeyName == "" {
		return ErrDecryptVerifyNoDecryptionKeyName
	}

	if d.VerifyKeyName == "" {
		return ErrDecryptVerifyNoVerificationKeyName
	}

	if d.Method == "" {
		return ErrDecryptVerifyNoMethod
	}

	var dCryptoKey model.CryptoKey
	if err := db.Get(&dCryptoKey, "name=?", d.DecryptKeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrDecryptVerifyKeyNotFound, d.DecryptKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve decryption key from database: %w", err)
	}

	var vCryptoKey model.CryptoKey
	if err := db.Get(&vCryptoKey, "name=?", d.VerifyKeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrDecryptVerifyKeyNotFound, d.VerifyKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve verification key from database: %w", err)
	}

	switch d.Method {
	case EncryptSignMethodPGP:
		return d.makePGPVerifyDecryptor(&dCryptoKey, &vCryptoKey)
	default:
		return fmt.Errorf("%w: %s", ErrDecryptVerifyInvalidMethod, d.Method)
	}
}

func (d *decryptVerify) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := d.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	return decryptFile(logger, transCtx, bool(d.KeepOriginal), d.OutputFile, d.decryptVerify)
}
