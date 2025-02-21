package tasks

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type decryptAES struct {
	aesKeyParam
	KeepOriginal boolStr    `json:"keepOriginal"`
	OutputFile   string     `json:"outputFile"`
	Mode         cipherMode `json:"mode"`
}

func (d *decryptAES) ValidateDB(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, d); err != nil {
		return fmt.Errorf("failed to parse the AES decryption parameters: %w", err)
	}

	return d.validateDB(db)
}

func (d *decryptAES) Run(_ context.Context, params map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := d.ValidateDB(db, params); err != nil {
		logger.Error(err.Error())

		return err
	}

	if err := decryptFile(logger, transCtx, bool(d.KeepOriginal), d.OutputFile,
		d.decrypt); err != nil {
		return err
	}

	return nil
}

func (d *decryptAES) decrypt(src io.Reader, dst io.Writer) error {
	block, aesErr := aes.NewCipher(d.key)
	if aesErr != nil {
		return fmt.Errorf("failed to create AES cipher: %w", aesErr)
	}

	switch d.Mode {
	case encryptModeCTR:
		return decryptStream(src, dst, block, cipher.NewCTR)
	case encryptModeCFB:
		return decryptStream(src, dst, block, cipher.NewCFBDecrypter)
	case encryptModeOFB:
		return decryptStream(src, dst, block, cipher.NewOFB)
	default:
		return fmt.Errorf("%w: %s", ErrInvalidCipherMode, d.Mode)
	}
}
