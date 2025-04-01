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
	ErrDecryptNoKeyName     = errors.New("missing decryption key name")
	ErrDecryptNoMethod      = errors.New("missing decryption method")
	ErrDecryptKeyNotFound   = errors.New("decryption key not found")
	ErrDecryptInvalidMethod = errors.New("invalid decryption method")
)

type decryptFunc func(src io.Reader, dst io.Writer) error

type decrypt struct {
	KeyName      string   `json:"keyName"`
	KeepOriginal jsonBool `json:"keepOriginal"`
	OutputFile   string   `json:"outputFile"`
	Method       string   `json:"method"`

	decrypt decryptFunc
}

func (d *decrypt) ValidateDB(db database.ReadAccess, params map[string]string) error {
	*d = decrypt{}
	if err := utils.JSONConvert(params, d); err != nil {
		return fmt.Errorf("failed to parse the decryption parameters: %w", err)
	}

	if d.KeyName == "" {
		return ErrDecryptNoKeyName
	}

	if d.Method == "" {
		return ErrDecryptNoMethod
	}

	var cryptoKey model.CryptoKey
	if err := db.Get(&cryptoKey, "name=?", d.KeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrDecryptKeyNotFound, d.KeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve decryption key from database: %w", err)
	}

	switch d.Method {
	case EncryptMethodAESCFB:
		return d.makeAESCFBDecryptor(&cryptoKey)
	case EncryptMethodAESCTR:
		return d.makeAESCTRDecryptor(&cryptoKey)
	case EncryptMethodAESOFB:
		return d.makeAESOFBDecryptor(&cryptoKey)
	case EncryptMethodPGP:
		return d.makePGPDecryptor(&cryptoKey)
	default:
		return fmt.Errorf("%w: %s", ErrDecryptInvalidMethod, d.Method)
	}
}

func (d *decrypt) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := d.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	return decryptFile(logger, transCtx, bool(d.KeepOriginal), d.OutputFile, d.decrypt)
}
