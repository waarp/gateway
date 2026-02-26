//nolint:dupl // keep tasks separate in case they change in the future
package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/ordered"
)

type verifyMethod struct {
	KeyTypes   []string
	mkVerifier func(*model.CryptoKey) (verifyFunc, error)
}

//nolint:gochecknoglobals //global var is needed here for future-proofing
var VerifyMethods = ordered.Map[string, *verifyMethod]{}

//nolint:gochecknoinits //init is needed here to initialize constants
func init() {
	VerifyMethods.Add(SignMethodHMACSHA256, &verifyMethod{
		KeyTypes:   []string{model.CryptoKeyTypeHMAC},
		mkVerifier: makeHMACSHA256Verifier,
	})
	VerifyMethods.Add(SignMethodHMACSHA384, &verifyMethod{
		KeyTypes:   []string{model.CryptoKeyTypeHMAC},
		mkVerifier: makeHMACSHA384Verifier,
	})
	VerifyMethods.Add(SignMethodHMACSHA512, &verifyMethod{
		KeyTypes:   []string{model.CryptoKeyTypeHMAC},
		mkVerifier: makeHMACSHA512Verifier,
	})
	VerifyMethods.Add(SignMethodHMACMD5, &verifyMethod{
		KeyTypes:   []string{model.CryptoKeyTypeHMAC},
		mkVerifier: makeHMACMD5Verifier,
	})
	VerifyMethods.Add(SignMethodPGP, &verifyMethod{
		KeyTypes:   []string{model.CryptoKeyTypePGPPrivate, model.CryptoKeyTypePGPPublic},
		mkVerifier: makePGPVerifier,
	})
}

var (
	ErrVerifyNoKeyName     = errors.New("missing verification key name")
	ErrVerifyNoMethod      = errors.New("missing verification method")
	ErrVerifyKeyNotFound   = errors.New("verification key not found")
	ErrVerifyUnknownMethod = errors.New("unknown verification method")
)

type verifyFunc func(file io.Reader, expected []byte) error

type verify struct {
	KeyName       string `json:"keyName"`
	Method        string `json:"method"`
	SignatureFile string `json:"signatureFile"`

	verify verifyFunc
}

func (v *verify) ValidateDB(db database.ReadAccess, params map[string]string) error {
	*v = verify{}
	if err := utils.JSONConvert(params, v); err != nil {
		return fmt.Errorf("failed to parse the verification parameters: %w", err)
	}

	if v.KeyName == "" {
		return ErrVerifyNoKeyName
	}

	if v.Method == "" {
		return ErrVerifyNoMethod
	}

	var cryptoKey model.CryptoKey
	if err := db.Get(&cryptoKey, "name = ?", v.KeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrVerifyKeyNotFound, v.KeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve verification key from database: %w", err)
	}

	method, ok := VerifyMethods.Get(v.Method)
	if !ok {
		return fmt.Errorf("%w: %s", ErrVerifyUnknownMethod, v.Method)
	}

	var err error
	if v.verify, err = method.mkVerifier(&cryptoKey); err != nil {
		return err
	}

	return nil
}

func (v *verify) Run(_ context.Context, params map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext, _ any,
) error {
	if err := v.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	sig, sigErr := fs.ReadFullFile(v.SignatureFile)
	if sigErr != nil {
		logger.Errorf("Failed to read signature file: %v", sigErr)

		return fmt.Errorf("failed to read signature file: %w", sigErr)
	}

	file, openErr := fs.Open(transCtx.Transfer.LocalPath)
	if openErr != nil {
		logger.Errorf("Failed to open signature file: %v", openErr)

		return fmt.Errorf("failed to open signature file: %w", openErr)
	}

	defer file.Close() //nolint:errcheck //this error is inconsequential

	if err := v.verify(file, sig); err != nil {
		logger.Errorf("Failed to check signature: %v", err)

		return fmt.Errorf("failed to check signature: %w", err)
	}

	return nil
}
