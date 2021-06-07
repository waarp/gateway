package internal

import (
	"crypto/sha256"
	"io"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

func MakeHash(logger *log.Logger, path string) ([]byte, *types.TransferError) {
	hasher := sha256.New()
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		logger.Errorf("Failed to open file for hash calculation: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to open file")
	}
	defer func() { _ = file.Close() }()

	_, err = io.Copy(hasher, file)
	if err != nil {
		logger.Errorf("Failed to read file content to hash: %s", err)
		return nil, types.NewTransferError(types.TeInternal, "failed to read file")
	}

	return hasher.Sum(nil), nil
}
