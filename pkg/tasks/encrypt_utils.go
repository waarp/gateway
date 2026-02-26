package tasks

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:dupl //similar to decryptFile, but best keep them separate
func encryptFile(logger *log.Logger, transCtx *model.TransferContext,
	keepOriginal bool, outputFile string,
	encryptFunc func(src io.Reader, dst io.Writer) error,
) error {
	plainFilepath := transCtx.Transfer.LocalPath
	cryptFilepath := plainFilepath + ".crypt"

	if outputFile != "" {
		cryptFilepath = outputFile
	}

	if err := doEncryptFile(logger, plainFilepath, cryptFilepath,
		encryptFunc); err != nil {
		if rmErr := fs.Remove(cryptFilepath); rmErr != nil {
			logger.Warningf("Failed to delete partial encrypted file %q: %v",
				cryptFilepath, rmErr)
		}

		return err
	}

	if !keepOriginal {
		if err := fs.Remove(plainFilepath); err != nil {
			return fmt.Errorf("failed to delete plaintext file %q: %w", plainFilepath, err)
		}
	}

	transCtx.Transfer.LocalPath = cryptFilepath

	return nil
}

func doEncryptFile(logger *log.Logger, plainPath, cryptPath string,
	encryptFunc func(src io.Reader, dst io.Writer) error,
) error {
	plainFile, err1 := fs.Open(plainPath)
	if err1 != nil {
		logger.Errorf("Failed to open plaintext file %q: %v", plainPath, err1)

		return fmt.Errorf("failed to open plaintext file %q: %w", plainPath, err1)
	}

	defer func() {
		if closeErr := plainFile.Close(); closeErr != nil && !errors.Is(closeErr, fs.ErrClosed) {
			logger.Warningf("Failed to close plaintext file %q: %v", plainPath, closeErr)
		}
	}()

	cryptFile, err2 := fs.Create(cryptPath)
	if err2 != nil {
		logger.Errorf("Failed to create encrypted file %q: %v", plainPath, err2)

		return fmt.Errorf("failed to create encrypted file %q: %w", cryptPath, err2)
	}

	defer func() {
		if closeErr := cryptFile.Close(); closeErr != nil && !errors.Is(closeErr, fs.ErrClosed) {
			logger.Warningf("Failed to close encrypted file %q: %v", cryptPath, closeErr)
		}
	}()

	wCryptFile, canWrite := cryptFile.(io.Writer)
	if !canWrite {
		logger.Errorf("Encrypted file %q cannot be written to", cryptPath)

		return fmt.Errorf("cannot write to encrypted file %q: %w", cryptPath, fs.ErrNotImplemented)
	}

	if err := encryptFunc(plainFile, wCryptFile); err != nil {
		logger.Errorf("Failed to encrypt file %q: %v", plainPath, err)

		return fmt.Errorf("failed to encrypt file %q: %w", plainPath, err)
	}

	if closeErr1 := cryptFile.Close(); closeErr1 != nil {
		logger.Errorf("Failed to close encrypted file %q: %v", cryptPath, closeErr1)

		return fmt.Errorf("failed to close encrypted file: %w", closeErr1)
	}

	if closeErr2 := plainFile.Close(); closeErr2 != nil {
		logger.Errorf("Failed to close plaintext file %q: %v", plainPath, closeErr2)

		return fmt.Errorf("failed to close plaintext file: %w", closeErr2)
	}

	return nil
}

func encryptStream(src io.Reader, dst io.Writer, block cipher.Block,
	mkStream func(cipher.Block, []byte) cipher.Stream,
) error {
	iv := make([]byte, block.BlockSize())
	if _, err := rand.Read(iv); err != nil {
		return fmt.Errorf("failed to generate IV: %w", err)
	}

	stream := mkStream(block, iv)

	if _, err := dst.Write(iv); err != nil {
		return fmt.Errorf("failed to write the IV to the encrypted file: %w", err)
	}

	streamWriter := cipher.StreamWriter{S: stream, W: dst}

	if _, err := io.Copy(streamWriter, src); err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	return nil
}
