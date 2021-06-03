package sftp

import (
	"errors"
	"fmt"
	"regexp"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"github.com/pkg/sftp"
)

var (
	errDatabase = fmt.Errorf("database error")
)

// modelToSFTP converts the given error into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func modelToSFTP(err *types.TransferError) error {
	switch err.Code {
	case types.TeOk:
		return sftp.ErrSSHFxOk
	case types.TeUnimplemented:
		return sftp.ErrSSHFxOpUnsupported
	case types.TeFileNotFound:
		return sftp.ErrSSHFxNoSuchFile
	case types.TeForbidden:
		return sftp.ErrSSHFxPermissionDenied
	default:
		return fmt.Errorf(err.Error())
	}
}

func sftpToModel(err error, defaults types.TransferErrorCode) *types.TransferError {
	code := defaults
	msg := err.Error()

	var sErr *sftp.StatusError
	if !errors.As(err, &sErr) {
		return types.NewTransferError(code, msg)
	}

	regex, _ := regexp.Compile(`sftp: "TransferError\((Te\w*)\): (.*)" \(.*\)`)
	s := regex.FindStringSubmatch(err.Error())
	if len(s) >= 3 {
		code = types.TecFromString(s[1])
		msg = s[2]
		return types.NewTransferError(code, msg)
	}

	switch sErr.FxCode() {
	case sftp.ErrSSHFxOk, sftp.ErrSSHFxEOF:
		return nil
	case sftp.ErrSSHFxNoSuchFile:
		code = types.TeFileNotFound
	case sftp.ErrSSHFxPermissionDenied:
		code = types.TeForbidden
	case sftp.ErrSSHFxFailure:
		code = types.TeUnknownRemote
	case sftp.ErrSSHFxBadMessage:
		code = types.TeUnimplemented
	case sftp.ErrSSHFxNoConnection:
		code = types.TeConnection
	case sftp.ErrSSHFxConnectionLost:
		code = types.TeConnectionReset
	case sftp.ErrSSHFxOpUnsupported:
		code = types.TeUnimplemented
	}

	regex2, _ := regexp.Compile(`sftp: "(.*)" \(.*\)`)
	s2 := regex2.FindStringSubmatch(err.Error())
	if len(s2) >= 1 {
		msg = s2[1]
	}
	return types.NewTransferError(code, msg)
}
