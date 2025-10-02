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

var (
	ErrDecryptVerifyNoDecryptionKeyName   = errors.New("missing decryption key name")
	ErrDecryptVerifyNoVerificationKeyName = errors.New("missing verification key name")
	ErrDecryptVerifyNoMethod              = errors.New("missing decryption/verification method")
	ErrDecryptVerifyKeyNotFound           = errors.New("cryptographic key not found")
	ErrDecryptVerifyInvalidMethod         = errors.New("invalid decryption/verification method")
)

type decryptVerifyFunc func(src io.Reader, dst io.Writer) error

type decryptVerifyMethod struct {
	KeyTypesDecrypt   []string
	KeyTypesVerify    []string
	mkDecryptVerifier func(*model.CryptoKey, *model.CryptoKey) (decryptVerifyFunc, error)
}

//nolint:gochecknoglobals //global var is needed here for future-proofing
var DecryptVerifyMethods = ordered.Map[string, *decryptVerifyMethod]{}

//nolint:gochecknoinits //init is needed here to initialize constants
func init() {
	DecryptVerifyMethods.Add(DecryptVerifyMethodPGP, &decryptVerifyMethod{
		KeyTypesDecrypt:   []string{model.CryptoKeyTypePGPPrivate},
		KeyTypesVerify:    []string{model.CryptoKeyTypePGPPublic, model.CryptoKeyTypePGPPrivate},
		mkDecryptVerifier: makePGPVerifyDecryptor,
	})
}

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

	method, ok := DecryptVerifyMethods.Get(d.Method)
	if !ok {
		return fmt.Errorf("%w %q", ErrEncryptSignUnknownMethod, d.Method)
	}

	var err error
	if d.decryptVerify, err = method.mkDecryptVerifier(&dCryptoKey, &vCryptoKey); err != nil {
		return err
	}

	return nil
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
