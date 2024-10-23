package tasks

import (
	"context"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec //MD5 is needed for compatibility
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"hash"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrVerifyHMACNoSigFile        = errors.New("missing HMAC signature file")
	ErrVerifyHMACNoKey            = errors.New("missing HMAC signature key")
	ErrVerifyHMACInvalidAlgorithm = errors.New("invalid HMAC signature algorithm")
	ErrVerifyHMACInvalidSignature = errors.New("invalid HMAC signature")
)

type verifyHMAC struct {
	Algorithm     hmacAlgorithm `json:"algorithm"`
	SignatureFile string        `json:"signatureFile"`
	Key           []byte        `json:"key"`
}

func (s *verifyHMAC) parseParams(params map[string]string) error {
	if err := utils.JSONConvert(params, s); err != nil {
		return fmt.Errorf("failed to parse the HMAC signature parameters: %w", err)
	}

	if s.SignatureFile == "" {
		return ErrVerifyHMACNoSigFile
	}

	if len(s.Key) == 0 {
		return ErrVerifyHMACNoKey
	}

	return nil
}

func (s *verifyHMAC) Validate(params map[string]string) error {
	return s.parseParams(params)
}

func (s *verifyHMAC) Run(_ context.Context, params map[string]string, _ *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := s.parseParams(params); err != nil {
		logger.Error(err.Error())

		return err
	}

	sig, sigErr := fs.ReadFullFile(s.SignatureFile)
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

	if err := s.checkSignature(file, sig); err != nil {
		logger.Error("Failed to check signature: %v", err)

		return fmt.Errorf("failed to check signature: %w", err)
	}

	return nil
}

func (s *verifyHMAC) checkSignature(file io.Reader, expected []byte) error {
	var hashFunc func() hash.Hash

	switch s.Algorithm {
	case hmacAlgoSHA256:
		hashFunc = sha256.New
	case hmacAlgoSHA384:
		hashFunc = sha512.New384
	case hmacAlgoSHA512:
		hashFunc = sha512.New
	case hmacAlgoMD5:
		hashFunc = md5.New
	default:
		return fmt.Errorf("%w: %q", ErrVerifyHMACInvalidAlgorithm, s.Algorithm)
	}

	hasher := hmac.New(hashFunc, s.Key)
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to compute file signature: %w", err)
	}

	actual := hasher.Sum(nil)
	if !hmac.Equal(expected, actual) {
		return ErrVerifyHMACInvalidSignature
	}

	return nil
}
