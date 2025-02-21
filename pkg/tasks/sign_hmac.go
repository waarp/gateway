package tasks

import (
	"context"
	"crypto/hmac"
	"crypto/md5" //nolint:gosec //MD5 is needed for compatibility
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
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
	ErrSignHMACInvalidAlgorithm = errors.New("invalid HMAC signature algorithm")
	ErrSignHMACNoKeyName        = errors.New("missing HMAC signature key")
	ErrSignHMACKeyNotFound      = errors.New("HMAC key not found")
	ErrSignHMACNoKey            = errors.New("cryptographic key does not contain an HMAC key")
)

type hmacAlgorithm string

const (
	hmacAlgoSHA256 hmacAlgorithm = "SHA256"
	hmacAlgoSHA384 hmacAlgorithm = "SHA384"
	hmacAlgoSHA512 hmacAlgorithm = "SHA512"
	hmacAlgoMD5    hmacAlgorithm = "MD5"
)

func (h *hmacAlgorithm) UnmarshalJSON(data []byte) error {
	var algoStr string
	if err := json.Unmarshal(data, &algoStr); err != nil {
		return err //nolint:wrapcheck //wrapping adds nothing here
	}

	switch algo := hmacAlgorithm(algoStr); algo {
	case hmacAlgoSHA256, hmacAlgoSHA384, hmacAlgoSHA512, hmacAlgoMD5:
		*h = algo

		return nil
	default:
		return fmt.Errorf("%w: %q", ErrSignHMACInvalidAlgorithm, algoStr)
	}
}

type hmacKeyParam struct {
	HMACKeyName string `json:"hmacKeyName"`

	key []byte
}

//nolint:dupl //best keep separate from the AES equivalent
func (h *hmacKeyParam) validateDB(db database.ReadAccess) error {
	if h.HMACKeyName == "" {
		return ErrSignHMACNoKeyName
	}

	var hmacKey model.CryptoKey
	if err := db.Get(&hmacKey, "name = ?", h.HMACKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrSignHMACKeyNotFound, h.HMACKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve HMAC key from database: %w", err)
	}

	if !isHMACKey(&hmacKey) || hmacKey.Key == "" {
		return fmt.Errorf("%q: %w", hmacKey.Name, ErrSignHMACNoKey)
	}

	var decErr error
	if h.key, decErr = base64.StdEncoding.DecodeString(hmacKey.Key.String()); decErr != nil {
		return fmt.Errorf("failed to decode the AES key: %w", decErr)
	}

	return nil
}

type signHMAC struct {
	hmacKeyParam
	Algorithm  hmacAlgorithm `json:"algorithm"`
	OutputFile string        `json:"outputFile"`
}

func (s *signHMAC) ValidateDB(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, s); err != nil {
		return fmt.Errorf("failed to parse the HMAC signature parameters: %w", err)
	}

	return s.validateDB(db)
}

func (s *signHMAC) Run(_ context.Context, params map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
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

	sig, sigErr := s.makeSignature(file)
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

func (s *signHMAC) makeSignature(file io.Reader) ([]byte, error) {
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
		return nil, fmt.Errorf("%w: %s", ErrSignHMACInvalidAlgorithm, s.Algorithm)
	}

	hasher := hmac.New(hashFunc, s.key)
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, fmt.Errorf("failed to compute file signature: %w", err)
	}

	return hasher.Sum(nil), nil
}

func isHMACKey(key *model.CryptoKey) bool {
	return key.Type == model.CryptoKeyTypeHMAC
}
