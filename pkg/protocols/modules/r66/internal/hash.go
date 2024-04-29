package internal

//nolint:gosec // MD5 is needed for compatibility with WaarpR66
//nolint:gosec // SHA1 is needed for compatibility with WaarpR66
import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"hash"
	"hash/adler32"
	"io"
	"os"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errUnknownHashAlgo error = errors.New("unsuported hash algorithm")

func GetHasher(h string) (hash.Hash, error) {
	switch h {
	case "ADLER32":
		return adler32.New(), nil
	// case "CRC32": // FIXME Add with specific table ?
	// case "MD2": // FIXME Add despite security concerns ?
	case "MD5":
		return md5.New(), nil //nolint:gosec // MD5 is needed for compatibility with WaarpR66
	case "SHA-1":
		return sha1.New(), nil //nolint:gosec // SHA1 is needed for compatibility with WaarpR66
	case "SHA-256":
		return sha256.New(), nil
	case "SHA-384":
		return sha512.New384(), nil
	case "SHA-512":
		return sha512.New(), nil
	default:
		return nil, errUnknownHashAlgo
	}
}

// MakeHash takes a file path and returns the sha256 checksum of the file.
func MakeHash(ctx context.Context, hashAlgo string, filesys fs.FS, logger *log.Logger, path *types.URL,
) ([]byte, *pipeline.Error) {
	hasher, err := GetHasher(hashAlgo)
	if err != nil {
		return nil, pipeline.NewErrorWith(types.TeInternal, "unknown hash algorithm", err)
	}

	file, opErr := fs.OpenFile(filesys, path, os.O_RDONLY, 0o600)
	if opErr != nil {
		logger.Error("Failed to open file for hash calculation: %s", opErr)

		return nil, pipeline.NewErrorWith(types.TeInternal, "failed to open file", opErr)
	}

	defer func() {
		if fErr := file.Close(); fErr != nil {
			logger.Warning("Failed to close file: %s", fErr)
		}
	}()

	if err := utils.RunWithCtx(ctx, func() error {
		if _, err := io.Copy(hasher, file); err != nil {
			logger.Error("Failed to read file content to hash: %s", err)

			return pipeline.NewErrorWith(types.TeInternal, "failed to read file", err)
		}

		return nil
	}); err != nil {
		var pErr *pipeline.Error
		if errors.As(err, &pErr) {
			return nil, pErr
		}

		return nil, pipeline.NewError(types.TeStopped, "transfer interrupted")
	}

	return hasher.Sum(nil), nil
}
