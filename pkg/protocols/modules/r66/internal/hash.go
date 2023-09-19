package internal

import (
	"context"
	"crypto/sha256"
	"io"
	"os"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// MakeHash takes a file path and returns the sha256 checksum of the file.
func MakeHash(ctx context.Context, filesys fs.FS, logger *log.Logger, path *types.URL,
) ([]byte, *types.TransferError) {
	hasher := sha256.New()

	file, opErr := fs.OpenFile(filesys, path, os.O_RDONLY, 0o600)
	if opErr != nil {
		logger.Error("Failed to open file for hash calculation: %s", opErr)

		return nil, types.NewTransferError(types.TeInternal, "failed to open file")
	}

	defer func() {
		if fErr := file.Close(); fErr != nil {
			logger.Warning("Failed to close file: %s", fErr)
		}
	}()

	if err := utils.RunWithCtx(ctx, func() error {
		if _, err := io.Copy(hasher, file); err != nil {
			logger.Error("Failed to read file content to hash: %s", err)

			return types.NewTransferError(types.TeInternal, "failed to read file")
		}

		return nil
	}); err != nil {
		//nolint:errorlint,forcetypeassert //assertion always succeeds
		return nil, err.(*types.TransferError)
	}

	return hasher.Sum(nil), nil
}
