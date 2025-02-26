//nolint:dupl // keep tasks separate in case they change in the future
package tasks

import (
	"context"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrVerifyNoKeyName     = errors.New("missing verification key name")
	ErrVerifyNoMethod      = errors.New("missing verification method")
	ErrVerifyKeyNotFound   = errors.New("verification key not found")
	ErrVerifyInvalidMethod = errors.New("invalid verification method")
)

type verifyFunc func(file io.Reader, expected []byte) error

type verify struct {
	KeyName       string `json:"keyName"`
	Method        string `json:"method"`
	SignatureFile string `json:"signatureFile"`

	verify verifyFunc
}

func (v *verify) ValidateDB(db database.ReadAccess, params map[string]string) error {
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
	if err := db.Get(&cryptoKey, "name = ?", v.KeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrVerifyKeyNotFound, v.KeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve verification key from database: %w", err)
	}

	switch v.Method {
	case SignMethodHMACSHA256:
		return v.makeHMACSHA256Verifier(&cryptoKey)
	case SignMethodHMACSHA384:
		return v.makeHMACSHA384Verifier(&cryptoKey)
	case SignMethodHMACSHA512:
		return v.makeHMACSHA512Verifier(&cryptoKey)
	case SignMethodHMACMD5:
		return v.makeHMACMD5Verifier(&cryptoKey)
	case SignMethodPGP:
		return v.makePGPVerifier(&cryptoKey)
	default:
		return fmt.Errorf("%w: %s", ErrVerifyInvalidMethod, v.Method)
	}
}

func (v *verify) Run(_ context.Context, params map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := v.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	sig, sigErr := fs.ReadFullFile(v.SignatureFile)
	if sigErr != nil {
		logger.Error("Failed to read signature file: %v", sigErr)

		return fmt.Errorf("failed to read signature file: %w", sigErr)
	}

	file, openErr := fs.Open(transCtx.Transfer.LocalPath)
	if openErr != nil {
		logger.Error("Failed to open signature file: %v", openErr)

		return fmt.Errorf("failed to open signature file: %w", openErr)
	}

	defer file.Close() //nolint:errcheck //this error is inconsequential

	if err := v.verify(file, sig); err != nil {
		logger.Error("Failed to check signature: %v", err)

		return fmt.Errorf("failed to check signature: %w", err)
	}

	return nil
}
