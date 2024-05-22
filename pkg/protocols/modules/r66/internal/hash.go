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

const (
	HashADLER32 = "ADLER32"
	HashMD5     = "MD5"
	HashSHA1    = "SHA-1"
	HashSHA256  = "SHA-256"
	HashSHA384  = "SHA-384"
	HashSHA512  = "SHA-512"
)

var errUnknownHashAlgo error = errors.New("unsuported hash algorithm")

func GetHasher(h string) (hash.Hash, error) {
	switch h {
	case HashADLER32:
		return adler32.New(), nil
	// case "CRC32": // FIXME Add with specific table ?
	// case "MD2": // FIXME Add despite security concerns ?
	case HashMD5:
		return md5.New(), nil //nolint:gosec // MD5 is needed for compatibility with WaarpR66
	case HashSHA1:
		return sha1.New(), nil //nolint:gosec // SHA1 is needed for compatibility with WaarpR66
	case HashSHA256:
		return sha256.New(), nil
	case HashSHA384:
		return sha512.New384(), nil
	case HashSHA512:
		return sha512.New(), nil
	default:
		return nil, errUnknownHashAlgo
	}
}

// MakeHash takes a file path and returns the sha256 checksum of the file.
func MakeHash(ctx context.Context, hashAlgo string, filesys fs.FS, logger *log.Logger, path *types.URL,
) ([]byte, *pipeline.Error) {
	hasher, hashErr := GetHasher(hashAlgo)
	if hashErr != nil {
		return nil, pipeline.NewErrorWith(types.TeInternal, "unknown hash algorithm", hashErr)
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
