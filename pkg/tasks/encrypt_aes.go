package tasks

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrEncryptAESNoKeyName   = errors.New("missing AES encryption key")
	ErrEncryptAESKeyNotFound = errors.New("AES key not found")
	ErrEncryptAESNoKey       = errors.New("cryptographic key does not contain an AES key")
)

type aesKeyParam struct {
	AESKeyName string `json:"aesKeyName"`
	key        []byte
}

//nolint:dupl //best keep separate from the HMAC equivalent
func (a *aesKeyParam) validateDB(db database.ReadAccess) error {
	if a.AESKeyName == "" {
		return ErrEncryptAESNoKeyName
	}

	var aesKey model.CryptoKey
	if err := db.Get(&aesKey, "name = ?", a.AESKeyName).Run(); database.IsNotFound(err) {
		return fmt.Errorf("%w %q", ErrEncryptAESKeyNotFound, a.AESKeyName)
	} else if err != nil {
		return fmt.Errorf("failed to retrieve AES key from database: %w", err)
	}

	if !isAESKey(&aesKey) || aesKey.Key == "" {
		return fmt.Errorf("%q: %w", aesKey.Name, ErrEncryptAESNoKey)
	}

	var decErr error
	if a.key, decErr = base64.StdEncoding.DecodeString(aesKey.Key.String()); decErr != nil {
		return fmt.Errorf("failed to decode the AES key: %w", decErr)
	}

	return nil
}

type encryptAES struct {
	aesKeyParam
	KeepOriginal boolStr    `json:"keepOriginal"`
	OutputFile   string     `json:"outputFile"`
	Mode         cipherMode `json:"mode"`
}

func (e *encryptAES) ValidateDB(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, e); err != nil {
		return fmt.Errorf("failed to parse the AES encryption parameters: %w", err)
	}

	return e.validateDB(db)
}

func (e *encryptAES) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := e.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	if err := encryptFile(logger, transCtx, bool(e.KeepOriginal), e.OutputFile,
		e.encrypt); err != nil {
		return err
	}

	return nil
}

func (e *encryptAES) encrypt(src io.Reader, dst io.Writer) error {
	block, aesErr := aes.NewCipher(e.key)
	if aesErr != nil {
		return fmt.Errorf("failed to create AES cipher: %w", aesErr)
	}

	switch e.Mode {
	case encryptModeCTR:
		return encryptStream(src, dst, block, cipher.NewCTR)
	case encryptModeCFB:
		return encryptStream(src, dst, block, cipher.NewCFBEncrypter)
	case encryptModeOFB:
		return encryptStream(src, dst, block, cipher.NewOFB)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCipherMode, e.Mode)
	}
}

func isAESKey(key *model.CryptoKey) bool {
	return key.Type == model.CryptoKeyTypeAES
}
