package sftp

import (
	"fmt"
	"io"
	"os"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/pkg/sftp"
)

var (
	errDatabase = fmt.Errorf("database error")
)

// modelToSFTP converts the given error into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func modelToSFTP(err error) error {
	if tErr, ok := err.(types.TransferError); ok {
		switch tErr.Code {
		case types.TeOk:
			return sftp.ErrSSHFxOk
		case types.TeUnimplemented:
			return sftp.ErrSSHFxOpUnsupported
		case types.TeIntegrity:
			return sftp.ErrSSHFxBadMessage
		case types.TeFileNotFound:
			return sftp.ErrSSHFxNoSuchFile
		case types.TeForbidden:
			return sftp.ErrSSHFxPermissionDenied
		default:
			return fmt.Errorf(tErr.Details)
		}
	}
	return err
}

func sftpToCode(err error, def types.TransferErrorCode) types.TransferErrorCode {
	switch err {
	case sftp.ErrSSHFxNoSuchFile, os.ErrNotExist:
		return types.TeFileNotFound
	case sftp.ErrSSHFxBadMessage, sftp.ErrSSHFxOpUnsupported:
		return types.TeUnimplemented
	case sftp.ErrSSHFxConnectionLost:
		return types.TeConnectionReset
	case sftp.ErrSSHFxPermissionDenied:
		return types.TeForbidden
	case sftp.ErrSSHFxNoConnection:
		return types.TeConnection
	case sftp.ErrSSHFxEOF, io.EOF:
		return types.TeDataTransfer
	default:
		return def
	}
}
