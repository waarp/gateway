package internal

import (
	"context"
	"crypto/sha256"
	"io"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

// MakeHash takes a file path and returns the sha256 checksum of the file.
func MakeHash(ctx context.Context, logger *log.Logger, path string) ([]byte, *types.TransferError) {
	hasher := sha256.New()
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		logger.Errorf("Failed to open file for hash calculation: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to open file")
	}
	defer func() { _ = file.Close() }()

	res := make(chan *types.TransferError)
	go func() {
		defer close(res)
		_, err = io.Copy(hasher, file)
		if err != nil {
			logger.Errorf("Failed to read file content to hash: %s", err)
			res <- types.NewTransferError(types.TeInternal, "failed to read file")
		}
	}()
	select {
	case <-ctx.Done():
		return nil, types.NewTransferError(types.TeInternal, "hash calculation stopped")
	case err := <-res:
		if err != nil {
			return nil, err
		}
		return hasher.Sum(nil), nil
	}
}
