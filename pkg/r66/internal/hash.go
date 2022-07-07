package internal

import (
	"context"
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// MakeHash takes a file path and returns the sha256 checksum of the file.
func MakeHash(ctx context.Context, logger *log.Logger, path string) ([]byte, *types.TransferError) {
	hasher := sha256.New()

	file, err := os.OpenFile(filepath.Clean(path), os.O_RDONLY, 0o600)
	if err != nil {
		logger.Error("Failed to open file for hash calculation: %s", err)

		return nil, types.NewTransferError(types.TeInternal, "failed to open file")
	}

	errorChan := make(chan *types.TransferError)

	go func() {
		defer close(errorChan)

		_, err = io.Copy(hasher, file)
		if err != nil {
			logger.Error("Failed to read file content to hash: %s", err)
			errorChan <- types.NewTransferError(types.TeInternal, "failed to read file")
		}
	}()
	select {
	case <-ctx.Done():
		_ = file.Close() //nolint:errcheck // this error is irrelevant here

		return nil, types.NewTransferError(types.TeInternal, "hash calculation stopped")
	case cErr := <-errorChan:
		if cErr != nil {
			return nil, cErr
		}

		if fErr := file.Close(); fErr != nil {
			logger.Warning("Failed to close file: %s", fErr)
		}

		return hasher.Sum(nil), nil
	}
}
