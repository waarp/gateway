package internal

import (
	"errors"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-r66/r66"
)

// ToR66Error takes an error (preferably a types.TransferError) and returns the
// equivalent r66.Error.
func ToR66Error(err error) *r66.Error {
	var tErr *types.TransferError
	if !errors.As(err, &tErr) {
		return &r66.Error{Code: r66.Unknown, Detail: err.Error()}
	}

	switch tErr.Code {
	case types.TeOk:
		return &r66.Error{Code: r66.CompleteOk, Detail: ""}
	case types.TeUnknown:
		return &r66.Error{Code: r66.Unknown, Detail: tErr.Details}
	case types.TeInternal:
		return &r66.Error{Code: r66.Internal, Detail: tErr.Details}
	case types.TeUnimplemented:
		return &r66.Error{Code: r66.Unimplemented, Detail: tErr.Details}
	case types.TeConnection:
		return &r66.Error{Code: r66.ConnectionImpossible, Detail: tErr.Details}
	case types.TeConnectionReset:
		return &r66.Error{Code: r66.Disconnection, Detail: tErr.Details}
	case types.TeUnknownRemote:
		return &r66.Error{Code: r66.QueryRemotelyUnknown, Detail: tErr.Details}
	case types.TeExceededLimit:
		return &r66.Error{Code: r66.ServerOverloaded, Detail: tErr.Details}
	case types.TeBadAuthentication:
		return &r66.Error{Code: r66.BadAuthent, Detail: tErr.Details}
	case types.TeDataTransfer:
		return &r66.Error{Code: r66.TransferError, Detail: tErr.Details}
	case types.TeIntegrity:
		return &r66.Error{Code: r66.FinalOp, Detail: tErr.Details}
	case types.TeFinalization:
		return &r66.Error{Code: r66.FinalOp, Detail: tErr.Details}
	case types.TeExternalOperation:
		return &r66.Error{Code: r66.ExternalOperation, Detail: tErr.Details}
	case types.TeWarning:
		return &r66.Error{Code: r66.Warning, Detail: tErr.Details}
	case types.TeStopped:
		return &r66.Error{Code: r66.StoppedTransfer, Detail: tErr.Details}
	case types.TeCanceled:
		return &r66.Error{Code: r66.CanceledTransfer, Detail: tErr.Details}
	case types.TeFileNotFound:
		return &r66.Error{Code: r66.FileNotFound, Detail: tErr.Details}
	case types.TeForbidden:
		return &r66.Error{Code: r66.FileNotAllowed, Detail: tErr.Details}
	case types.TeBadSize:
		return &r66.Error{Code: r66.SizeNotAllowed, Detail: tErr.Details}
	case types.TeShuttingDown:
		return &r66.Error{Code: r66.Shutdown, Detail: tErr.Details}
	default:
		return &r66.Error{Code: r66.Unknown, Detail: tErr.Details}
	}
}

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
		pip.Pause()
		return nil
	case r66.CanceledTransfer:
		pip.Cancel()
		return nil

	default:
		return types.NewTransferError(types.TeUnknownRemote, rErr.Detail)
	}
}
