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
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/ordered"
)

type decryptMethod struct {
	KeyTypes    []string
	mkDecryptor func(*model.CryptoKey) (decryptFunc, error)
}

//nolint:gochecknoglobals //global var is needed here for future-proofing
var DecryptMethods = ordered.Map[string, *decryptMethod]{}

//nolint:gochecknoinits //init is needed here to initialize constants
func init() {
	DecryptMethods.Add(EncryptMethodAESCFB, &decryptMethod{
		KeyTypes:    []string{model.CryptoKeyTypeAES},
		mkDecryptor: makeAESCFBDecryptor,
	})
	DecryptMethods.Add(EncryptMethodAESCTR, &decryptMethod{
		KeyTypes:    []string{model.CryptoKeyTypeAES},
		mkDecryptor: makeAESCTRDecryptor,
	})
	DecryptMethods.Add(EncryptMethodAESOFB, &decryptMethod{
		KeyTypes:    []string{model.CryptoKeyTypeAES},
		mkDecryptor: makeAESOFBDecryptor,
	})
	DecryptMethods.Add(EncryptMethodPGP, &decryptMethod{
		KeyTypes:    []string{model.CryptoKeyTypePGPPrivate},
		mkDecryptor: makePGPDecryptor,
	})
}

var (
	ErrDecryptNoKeyName     = errors.New("missing decryption key name")
	ErrDecryptNoMethod      = errors.New("missing decryption method")
	ErrDecryptKeyNotFound   = errors.New("decryption key not found")
	ErrDecryptUnknownMethod = errors.New("unknown decryption method")
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

	method, ok := DecryptMethods.Get(d.Method)
	if !ok {
		return fmt.Errorf("%w %q", ErrDecryptUnknownMethod, d.Method)
	}

	var err error
	if d.decrypt, err = method.mkDecryptor(&cryptoKey); err != nil {
		return err
	}

	return nil
}

func (d *decrypt) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := d.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	return decryptFile(logger, transCtx, bool(d.KeepOriginal), d.OutputFile, d.decrypt)
}
