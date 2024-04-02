package sftp

import (
	"context"
	"errors"
	"io"
	"regexp"
	"time"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

var (
	ErrFileSystem = errors.New("file system error")
	ErrInternal   = errors.New("internal error")
	ErrDatabase   = errors.New("database error")
	ErrAuthFailed = errors.New("authentication failed")
)

// toSFTPErr converts the given error into its closest equivalent
// SFTP error code. Since SFTP v3 only supports 8 error codes (9 with code Ok),
// most errors will be converted to the generic code SSH_FX_FAILURE.
func toSFTPErr(err *pipeline.Error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, io.EOF) {
		return err
	}

	var tErr *pipeline.Error
	if !errors.As(err, &tErr) {
		return err
	}

	switch tErr.Code() {
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

const stopTimeout = 5 * time.Second

func fromSFTPErr(origErr error, defaults types.TransferErrorCode, pip *pipeline.Pipeline) *pipeline.Error {
	if err := checkTransferErrorString(origErr.Error(), pip); err != nil {
		return err
	}

	code := defaults
	msg := origErr.Error()

	var sErr *sftp.StatusError
	if !errors.As(origErr, &sErr) {
		return pipeline.NewError(code, "Error on remote partner: %s", msg)
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

	s2 := regex2.FindStringSubmatch(origErr.Error())
	if len(s2) >= 1 {
		msg = s2[1]
	}

	return pipeline.NewError(code, "Error on remote partner: %s", msg)
}

func checkTransferErrorString(errMsg string, pip *pipeline.Pipeline) *pipeline.Error {
	const (
		groupNB     = 3
		codeGroupNB = 1
		msgGroupNB  = 2
	)

	regex := regexp.MustCompile(`sftp: "TransferError\((Te\w*)\): (.*)" \(.*\)`)
	s := regex.FindStringSubmatch(errMsg)

	if len(s) < groupNB {
		return nil // not a transfer error string
	}

	msg := s[msgGroupNB]
	code := types.TecFromString(s[codeGroupNB])

	switch code {
	case types.TeStopped:
		ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
		defer cancel()

		if err := pip.Pause(ctx); err != nil {
			return pipeline.NewErrorWith(types.TeInternal, "failed to pause transfer", err)
		}
	case types.TeCanceled:
		ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
		defer cancel()

		if err := pip.Cancel(ctx); err != nil {
			return pipeline.NewErrorWith(types.TeInternal, "failed to cancel transfer", err)
		}

	default:
	}

	return pipeline.NewError(code, "Error on remote partner: %s", msg)
}
