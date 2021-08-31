package internal

import (
	"errors"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/pkg/sftp"
)

// ToSFTPErr converts the given error into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func ToSFTPErr(err error) error {
	var tErr *types.TransferError
	if !errors.As(err, &tErr) {
		return errors.New(err.Error())
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
		return errors.New(err.Error())
	}
}
