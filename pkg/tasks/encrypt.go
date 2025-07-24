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

//nolint:gochecknoglobals //global var is needed here for future-proofing
var EncryptMethods = map[string]struct {
	KeyTypes    []string
	mkEncryptor func(*model.CryptoKey) (encryptFunc, error)
}{
	EncryptMethodAESCFB: {
		KeyTypes:    []string{model.CryptoKeyTypeAES},
		mkEncryptor: makeAESCFBEncryptor,
	},
	EncryptMethodAESCTR: {
		KeyTypes:    []string{model.CryptoKeyTypeAES},
		mkEncryptor: makeAESCTREncryptor,
	},
	EncryptMethodAESOFB: {
		KeyTypes:    []string{model.CryptoKeyTypeAES},
		mkEncryptor: makeAESOFBEncryptor,
	},
	EncryptMethodPGP: {
		KeyTypes:    []string{model.CryptoKeyTypePGPPublic, model.CryptoKeyTypePGPPrivate},
		mkEncryptor: makePGPEncryptor,
	},
}

var (
	ErrEncryptNoKeyName     = errors.New("missing encryption key name")
	ErrEncryptNoMethod      = errors.New("missing encryption method")
	ErrEncryptKeyNotFound   = errors.New("encryption key not found")
	ErrEncryptUnknownMethod = errors.New("unknown encryption method")
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

	method, ok := EncryptMethods[e.Method]
	if !ok {
		return fmt.Errorf("%w %q", ErrEncryptUnknownMethod, e.Method)
	}

	var err error
	if e.encrypt, err = method.mkEncryptor(&cryptoKey); err != nil {
		return err
	}

	return nil
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
