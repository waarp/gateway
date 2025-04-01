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
	ErrEncryptNoKeyName     = errors.New("missing encryption key name")
	ErrEncryptNoMethod      = errors.New("missing encryption method")
	ErrEncryptKeyNotFound   = errors.New("encryption key not found")
	ErrEncryptInvalidMethod = errors.New("invalid encryption method")
)

type encryptFunc func(src io.Reader, dst io.Writer) error

type encrypt struct {
	KeyName      string   `json:"keyName"`
	KeepOriginal jsonBool `json:"keepOriginal"`
	OutputFile   string   `json:"outputFile"`
	Method       string   `json:"method"`

	encrypt encryptFunc
}

func (e *encrypt) ValidateDB(db database.ReadAccess, params map[string]string) error {
	*e = encrypt{}
	if err := utils.JSONConvert(params, e); err != nil {
		return fmt.Errorf("failed to parse the encryption parameters: %w", err)
	}

	if e.KeyName == "" {
		return ErrEncryptNoKeyName
	}

	if e.Method == "" {
		return ErrEncryptNoMethod
	}

	var cryptoKey model.CryptoKey
	if err := db.Get(&cryptoKey, "name=?", e.KeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrEncryptKeyNotFound, e.KeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve encryption key from database: %w", err)
	}

	switch e.Method {
	case EncryptMethodAESCFB:
		return e.makeAESCFBEncryptor(&cryptoKey)
	case EncryptMethodAESCTR:
		return e.makeAESCTREncryptor(&cryptoKey)
	case EncryptMethodAESOFB:
		return e.makeAESOFBEncryptor(&cryptoKey)
	case EncryptMethodPGP:
		return e.makePGPEncryptor(&cryptoKey)
	default:
		return fmt.Errorf("%w: %s", ErrEncryptInvalidMethod, e.Method)
	}
}

func (e *encrypt) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := e.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	return encryptFile(logger, transCtx, bool(e.KeepOriginal), e.OutputFile, e.encrypt)
}
