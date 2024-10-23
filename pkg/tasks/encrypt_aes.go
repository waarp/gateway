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
	ErrAESKeyLength   = errors.New("invalid AES key length")
	ErrAESInvalidMode = errors.New("invalid cipher mode")
)

type encryptAES struct {
	KeepOriginal boolStr    `json:"keepOriginal"`
	OutputFile   string     `json:"outputFile"`
	Key          string     `json:"key"`
	Mode         cipherMode `json:"mode"`
}

func (e *encryptAES) ToDB(params map[string]string) error {
	if err := e.parseParams(params); err != nil {
		return err
	}

	var cryptErr error
	if e.Key, cryptErr = utils.AESCrypt(database.GCM, e.Key); cryptErr != nil {
		return fmt.Errorf("failed to encrypt the AES key: %w", cryptErr)
	}

	if err := utils.JSONConvert(e, &params); err != nil {
		return fmt.Errorf("failed to serialize the AES encryp parameters: %w", err)
	}

	return nil
}

func (e *encryptAES) FromDB(params map[string]string) error {
	if err := e.parseParams(params); err != nil && !errors.Is(err, ErrAESKeyLength) {
		return err
	}

	var decryptErr error
	if e.Key, decryptErr = utils.AESDecrypt(database.GCM, e.Key); decryptErr != nil {
		return fmt.Errorf("failed to decrypt the AES key: %w", decryptErr)
	}

	if err := utils.JSONConvert(e, &params); err != nil {
		return fmt.Errorf("failed to serialize the AES encryp parameters: %w", err)
	}

	return nil
}

func (e *encryptAES) parseParams(params map[string]string) error {
	if err := utils.JSONConvert(params, e); err != nil {
		return fmt.Errorf("failed to parse the AES encryption parameters: %w", err)
	}

	switch length := len(e.Key); length {
	case 16, 24, 32: //nolint:mnd //too specific
	default:
		return fmt.Errorf("%w: %d", ErrAESKeyLength, length)
	}

	return nil
}

func (e *encryptAES) Validate(params map[string]string) error {
	return e.parseParams(params)
}

func (e *encryptAES) Run(_ context.Context, params map[string]string,
	_ *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := e.parseParams(params); err != nil {
		return err
	}

	if err := encryptFile(logger, transCtx, bool(e.KeepOriginal), e.OutputFile,
		e.encrypt); err != nil {
		return err
	}

	return nil
}

func (e *encryptAES) encrypt(src io.Reader, dst io.Writer) error {
	key, decErr := base64.StdEncoding.DecodeString(e.Key)
	if decErr != nil {
		return fmt.Errorf("failed to decode the AES key: %w", decErr)
	}

	block, aesErr := aes.NewCipher(key)
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
		return fmt.Errorf("%w: %s", ErrAESInvalidMode, e.Mode)
	}
}
