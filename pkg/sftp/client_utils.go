package sftp

import (
	"errors"
	"regexp"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func (c *client) fromSFTPErr(err error, defaults types.TransferErrorCode) *types.TransferError {
	code := defaults
	msg := err.Error()

	var sErr *sftp.StatusError
	if !errors.As(err, &sErr) {
		return types.NewTransferError(code, msg)
	}

	regex := regexp.MustCompile(`sftp: "TransferError\((Te\w*)\): (.*)" \(.*\)`)

	s := regex.FindStringSubmatch(err.Error())
	if len(s) >= 3 { //nolint:gomnd // using a const is unnecessary
		code = types.TecFromString(s[1])
		switch code {
		case types.TeStopped:
			c.pip.Pause()

		case types.TeCanceled:
			c.pip.Cancel()

		default:
		}

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

	regex2 := regexp.MustCompile(`sftp: "(.*)" \(.*\)`)

	s2 := regex2.FindStringSubmatch(err.Error())
	if len(s2) >= 1 {
		msg = s2[1]
	}

	return types.NewTransferError(code, msg)
}
