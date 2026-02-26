package tasks

import (
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:dupl //similar to encryptFile, but best keep them separate
func decryptFile(logger *log.Logger, transCtx *model.TransferContext,
	keepOriginal bool, outputFile string,
	decryptFunc func(src io.Reader, dst io.Writer) error,
) error {
	cryptFilepath := transCtx.Transfer.LocalPath
	plainFilepath := outputFile

	if plainFilepath == "" {
		if ext := path.Ext(cryptFilepath); ext == ".crypt" {
			plainFilepath = strings.TrimSuffix(cryptFilepath, ext)
		} else {
			plainFilepath = cryptFilepath + ".plain"
		}
	}

	if err := doDecryptFile(logger, cryptFilepath, plainFilepath,
		decryptFunc); err != nil {
		if rmErr := fs.Remove(plainFilepath); rmErr != nil {
			logger.Warningf("Failed to delete partial decrypted file %q: %v",
				plainFilepath, rmErr)
		}

		return err
	}

	if !keepOriginal {
		if err := fs.Remove(cryptFilepath); err != nil {
			return fmt.Errorf("failed to delete encrypted file %q: %w", cryptFilepath, err)
		}
	}

	transCtx.Transfer.LocalPath = plainFilepath

	return nil
}

func doDecryptFile(logger *log.Logger, cryptPath, plainPath string,
	decryptFunc func(src io.Reader, dst io.Writer) error,
) error {
	cryptFile, openErr := fs.Open(cryptPath)
	if openErr != nil {
		logger.Errorf("Failed to open encrypted file %q: %v", cryptPath, openErr)

		return fmt.Errorf("failed to open encrypted file %q: %w", cryptPath, openErr)
	}

	defer func() {
		if closeErr := cryptFile.Close(); closeErr != nil && !errors.Is(closeErr, fs.ErrClosed) {
			logger.Warningf("Failed to close source file %q: %v", cryptPath, closeErr)
		}
	}()

	plainFile, createErr := fs.Create(plainPath)
	if createErr != nil {
		logger.Errorf("Failed to create plaintext file %q: %v", plainPath, createErr)

		return fmt.Errorf("failed to create plaintext file %q: %w", plainPath, createErr)
	}

	defer func() {
		if closeErr := plainFile.Close(); closeErr != nil && !errors.Is(closeErr, fs.ErrClosed) {
			logger.Warningf("Failed to close plaintext file %q: %v", plainPath, closeErr)
		}
	}()

	wPlainFile, canWrite := plainFile.(io.Writer)
	if !canWrite {
		return fmt.Errorf("cannot write to plaintext file %q: %w", plainPath, fs.ErrNotImplemented)
	}

	if cryptErr := decryptFunc(cryptFile, wPlainFile); cryptErr != nil {
		return cryptErr
	}

	if closeErr1 := cryptFile.Close(); closeErr1 != nil {
		logger.Errorf("Failed to close encrypted file %q: %v", cryptPath, closeErr1)

		return fmt.Errorf("failed to close encrypted file: %w", closeErr1)
	}

	if closeErr2 := plainFile.Close(); closeErr2 != nil {
		logger.Errorf("Failed to close decrypted file %q: %v", plainPath, closeErr2)

		return fmt.Errorf("failed to close decrypted file: %w", closeErr2)
	}

	return nil
}

func decryptStream(src io.Reader, dst io.Writer, block cipher.Block,
	mkStream func(cipher.Block, []byte) cipher.Stream,
) error {
	iv := make([]byte, block.BlockSize())
	if _, err := src.Read(iv); err != nil {
		return fmt.Errorf("failed to read the IV: %w", err)
	}

	stream := mkStream(block, iv)
	streamReader := cipher.StreamReader{S: stream, R: src}

	if _, err := io.Copy(dst, streamReader); err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	return nil
}
