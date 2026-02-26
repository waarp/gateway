//nolint:dupl // keep tasks separate in case they change in the future
package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/ordered"
)

type encryptSignMethod struct {
	KeyTypesEncrypt []string
	KeyTypesSign    []string
	mkEncryptSigner func(*model.CryptoKey, *model.CryptoKey) (encryptSignFunc, error)
}

//nolint:gochecknoglobals //global var is needed here for future-proofing
var EncryptSignMethods = ordered.Map[string, *encryptSignMethod]{}

//nolint:gochecknoinits //init is needed here to initialize constants
func init() {
	EncryptSignMethods.Add(EncryptSignMethodPGP, &encryptSignMethod{
		KeyTypesEncrypt: []string{model.CryptoKeyTypePGPPublic, model.CryptoKeyTypePGPPrivate},
		KeyTypesSign:    []string{model.CryptoKeyTypePGPPrivate},
		mkEncryptSigner: makePGPSignEncryptor,
	})
}

var (
	ErrEncryptSignNoEncryptionKeyName = errors.New("missing encryption key name")
	ErrEncryptSignNoSignatureKeyName  = errors.New("missing signature key name")
	ErrEncryptSignNoMethod            = errors.New("missing encryption/signature method")
	ErrEncryptSignKeyNotFound         = errors.New("cryptographic key not found")
	ErrEncryptSignUnknownMethod       = errors.New("unknown encryption/signature method")
)

type encryptSignFunc func(src io.Reader, dst io.Writer) error

type encryptSign struct {
	KeepOriginal jsonBool `json:"keepOriginal"`
	OutputFile   string   `json:"outputFile"`

	Method         string `json:"method"`
	EncryptKeyName string `json:"encryptKeyName"`
	SignKeyName    string `json:"signKeyName"`

	encryptSign encryptSignFunc
}

func (e *encryptSign) ValidateDB(db database.ReadAccess, params map[string]string) error {
	*e = encryptSign{}
	if err := utils.JSONConvert(params, e); err != nil {
		return fmt.Errorf("failed to parse the encryption parameters: %w", err)
	}

	if e.EncryptKeyName == "" {
		return ErrEncryptSignNoEncryptionKeyName
	}

	if e.SignKeyName == "" {
		return ErrEncryptSignNoSignatureKeyName
	}

	if e.Method == "" {
		return ErrEncryptSignNoMethod
	}

	var eCryptoKey model.CryptoKey
	if err := db.Get(&eCryptoKey, "name=?", e.EncryptKeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrEncryptSignKeyNotFound, e.EncryptKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve encryption key from database: %w", err)
	}

	var sCryptoKey model.CryptoKey
	if err := db.Get(&sCryptoKey, "name=?", e.SignKeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrEncryptSignKeyNotFound, e.SignKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve signature key from database: %w", err)
	}

	method, ok := EncryptSignMethods.Get(e.Method)
	if !ok {
		return fmt.Errorf("%w %q", ErrEncryptSignUnknownMethod, e.Method)
	}

	var err error
	if e.encryptSign, err = method.mkEncryptSigner(&eCryptoKey, &sCryptoKey); err != nil {
		return err
	}

	return nil
}

func (e *encryptSign) Run(_ context.Context, params map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := e.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	return encryptFile(logger, transCtx, bool(e.KeepOriginal), e.OutputFile, e.encryptSign)
}
