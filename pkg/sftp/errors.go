package sftp

import (
	"errors"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var (
	errDatabase        = errors.New("database error")
	errFileSystem      = errors.New("file system error")
	errInvalidHandle   = errors.New("file handle is no longer valid")
	errAuthFailed      = errors.New("authentication failed")
	errFilepathParsing = errors.New("failed to parse file path")
)

// toSFTPErr converts the given error into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func toSFTPErr(err error) error {
	var tErr *types.TransferError

	if !errors.As(err, &tErr) {
		return err
	}

	switch tErr.Code {
	case types.TeOk:
		return sftp.ErrSSHFxOk
	case types.TeUnimplemented:
		return sftp.ErrSSHFxOpUnsupported
	case types.TeFileNotFound:
		return sftp.ErrSSHFxNoSuchFile
	case types.TeForbidden:
		return sftp.ErrSSHFxPermissionDenied
	default:
		return err
	}
}
