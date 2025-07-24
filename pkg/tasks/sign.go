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

//nolint:gochecknoglobals //global var is needed here for future-proofing
var SignMethods = map[string]struct {
	KeyTypes []string
	mkSigner func(*model.CryptoKey) (signFunc, error)
}{
	SignMethodHMACSHA256: {
		KeyTypes: []string{model.CryptoKeyTypeHMAC},
		mkSigner: makeHMACSHA256Signer,
	},
	SignMethodHMACSHA384: {
		KeyTypes: []string{model.CryptoKeyTypeHMAC},
		mkSigner: makeHMACSHA384Signer,
	},
	SignMethodHMACSHA512: {
		KeyTypes: []string{model.CryptoKeyTypeHMAC},
		mkSigner: makeHMACSHA512Signer,
	},
	SignMethodHMACMD5: {
		KeyTypes: []string{model.CryptoKeyTypeHMAC},
		mkSigner: makeHMACMD5Signer,
	},
	SignMethodPGP: {
		KeyTypes: []string{model.CryptoKeyTypePGPPrivate},
		mkSigner: makePGPSigner,
	},
}

var (
	ErrSignNoKeyName     = errors.New("missing signature key name")
	ErrSignNoMethod      = errors.New("missing signature method")
	ErrSignKeyNotFound   = errors.New("signature key not found")
	ErrSignUnknownMethod = errors.New("unknown signature method")
)

type signFunc func(file io.Reader) ([]byte, error)

type sign struct {
	KeyName    string `json:"keyName"`
	Method     string `json:"method"`
	OutputFile string `json:"outputFile"`

	sign signFunc
}

func (s *sign) ValidateDB(db database.ReadAccess, params map[string]string) error {
	*s = sign{}
	if err := utils.JSONConvert(params, s); err != nil {
		return fmt.Errorf("failed to parse the encryption parameters: %w", err)
	}

	if s.KeyName == "" {
		return ErrSignNoKeyName
	}

	if s.Method == "" {
		return ErrSignNoMethod
	}

	var cryptoKey model.CryptoKey
	if err := db.Get(&cryptoKey, "name=?", s.KeyName).Owner().Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrSignKeyNotFound, s.KeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve signature key from database: %w", err)
	}

	method, ok := SignMethods[s.Method]
	if !ok {
		return fmt.Errorf("%w %q", ErrSignUnknownMethod, s.Method)
	}

	var err error
	if s.sign, err = method.mkSigner(&cryptoKey); err != nil {
		return err
	}

	return nil
}

func (s *sign) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := s.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	file, openErr := fs.Open(transCtx.Transfer.LocalPath)
	if openErr != nil {
		logger.Error("Failed to open signature file: %v", openErr)

		return fmt.Errorf("failed to open signature file: %w", openErr)
	}

	defer file.Close() //nolint:errcheck //this error is inconsequential

	sig, sigErr := s.sign(file)
	if sigErr != nil {
		logger.Error("Failed to sign file: %v", sigErr)

		return fmt.Errorf("failed to sign file: %w", sigErr)
	}

	outputFile := s.OutputFile
	if outputFile == "" {
		outputFile = transCtx.Transfer.LocalPath + ".sig"
	}

	if err := fs.WriteFullFile(outputFile, sig); err != nil {
		logger.Error("Failed to write HMAC signature file: %v", err)

		return fmt.Errorf("failed to write HMAC signature file: %w", err)
	}

	return nil
}
