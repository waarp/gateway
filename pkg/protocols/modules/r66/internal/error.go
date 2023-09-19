package internal

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// NewR66Error returns a new r66.Error from the given code and message. The
// message accepts formats and arguments, similarly to the fmt package.
func NewR66Error(code rune, details string, args ...interface{}) *r66.Error {
	return &r66.Error{Code: code, Detail: fmt.Sprintf(details, args...)}
}

// ToR66Error takes an error (preferably a types.TransferError) and returns the
// equivalent r66.Error.
//
//nolint:funlen,gocyclo,cyclop // splitting the function would add complexity
func ToR66Error(err error) *r66.Error {
	var rErr *r66.Error
	if errors.As(err, &rErr) {
		return rErr
	}

	var tErr *types.TransferError
	if !errors.As(err, &tErr) {
		return NewR66Error(r66.Unknown, err.Error())
	}

	switch tErr.Code {
	case types.TeOk:
		return NewR66Error(r66.CompleteOk, "")
	case types.TeUnknown:
		return NewR66Error(r66.Unknown, tErr.Details)
	case types.TeInternal:
		return NewR66Error(r66.Internal, tErr.Details)
	case types.TeUnimplemented:
		return NewR66Error(r66.Unimplemented, tErr.Details)
	case types.TeConnection:
		return NewR66Error(r66.ConnectionImpossible, tErr.Details)
	case types.TeConnectionReset:
		return NewR66Error(r66.Disconnection, tErr.Details)
	case types.TeUnknownRemote:
		return NewR66Error(r66.QueryRemotelyUnknown, tErr.Details)
	case types.TeExceededLimit:
		return NewR66Error(r66.ServerOverloaded, tErr.Details)
	case types.TeBadAuthentication:
		return NewR66Error(r66.BadAuthent, tErr.Details)
	case types.TeDataTransfer:
		return NewR66Error(r66.TransferError, tErr.Details)
	case types.TeIntegrity:
		return NewR66Error(r66.FinalOp, tErr.Details)
	case types.TeFinalization:
		return NewR66Error(r66.FinalOp, tErr.Details)
	case types.TeExternalOperation:
		return NewR66Error(r66.ExternalOperation, tErr.Details)
	case types.TeWarning:
		return NewR66Error(r66.Warning, tErr.Details)
	case types.TeStopped:
		return NewR66Error(r66.StoppedTransfer, tErr.Details)
	case types.TeCanceled:
		return NewR66Error(r66.CanceledTransfer, tErr.Details)
	case types.TeFileNotFound:
		return NewR66Error(r66.FileNotFound, tErr.Details)
	case types.TeForbidden:
		return NewR66Error(r66.FileNotAllowed, tErr.Details)
	case types.TeBadSize:
		return NewR66Error(r66.SizeNotAllowed, tErr.Details)
	case types.TeShuttingDown:
		return NewR66Error(r66.Shutdown, tErr.Details)
	default:
		return NewR66Error(r66.Unknown, tErr.Details)
	}
}

// FromR66Error takes an R66 error (most likely of type r66.Error) and returns
// the corresponding types.TransferError.
//
//nolint:funlen,gocyclo,cyclop // splitting the function would add complexity
func FromR66Error(err error, pip *pipeline.Pipeline) *types.TransferError {
	var rErr *r66.Error
	if !errors.As(err, &rErr) {
		return types.NewTransferError(types.TeUnknownRemote, err.Error())
	}

	switch rErr.Code {
	case r66.InitOk, r66.PreProcessingOk, r66.TransferOk, r66.PostProcessingOk,
		r66.CompleteOk:
		return nil
	case r66.ConnectionImpossible:
		return types.NewTransferError(types.TeConnection, rErr.Detail)
	case r66.ServerOverloaded:
		return types.NewTransferError(types.TeExceededLimit, rErr.Detail)
	case r66.BadAuthent:
		return types.NewTransferError(types.TeBadAuthentication, rErr.Detail)
	case r66.ExternalOperation:
		return types.NewTransferError(types.TeExternalOperation, rErr.Detail)
	case r66.TransferError:
		return types.NewTransferError(types.TeDataTransfer, rErr.Detail)
	case r66.MD5Error:
		return types.NewTransferError(types.TeIntegrity, rErr.Detail)
	case r66.Disconnection:
		return types.NewTransferError(types.TeConnectionReset, rErr.Detail)
	case r66.RemoteShutdown:
		return types.NewTransferError(types.TeShuttingDown, rErr.Detail)
	case r66.FinalOp:
		return types.NewTransferError(types.TeFinalization, rErr.Detail)
	case r66.Unimplemented:
		return types.NewTransferError(types.TeUnimplemented, rErr.Detail)
	case r66.Shutdown:
		return types.NewTransferError(types.TeShuttingDown, rErr.Detail)
	case r66.RemoteError:
		return types.NewTransferError(types.TeUnknownRemote, rErr.Detail)
	case r66.Internal:
		return types.NewTransferError(types.TeInternal, rErr.Detail)
	case r66.Warning:
		return types.NewTransferError(types.TeWarning, rErr.Detail)
	case r66.Unknown:
		return types.NewTransferError(types.TeUnknownRemote, rErr.Detail)
	case r66.FileNotFound:
		return types.NewTransferError(types.TeFileNotFound, rErr.Detail)
	case r66.CommandNotFound:
		return types.NewTransferError(types.TeUnimplemented, rErr.Detail)
	case r66.IncorrectCommand:
		return types.NewTransferError(types.TeUnimplemented, rErr.Detail)
	case r66.FileNotAllowed:
		return types.NewTransferError(types.TeForbidden, rErr.Detail)
	case r66.SizeNotAllowed:
		return types.NewTransferError(types.TeForbidden, rErr.Detail)
	case r66.StoppedTransfer:
		if err := pip.Pause(context.Background()); err != nil {
			return AsTransferError(types.TeStopped, err)
		}

		return nil
	case r66.CanceledTransfer:
		if err := pip.Cancel(context.Background()); err != nil {
			return AsTransferError(types.TeCanceled, err)
		}

		return nil
	default:
		return types.NewTransferError(types.TeUnknownRemote, rErr.Detail)
	}
}

func AsTransferError(defaultCode types.TransferErrorCode, err error) *types.TransferError {
	tErr := types.NewTransferError(defaultCode, err.Error())
	errors.As(err, &tErr)

	return tErr
}
