package internal

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// NewR66Error returns a new r66.Error from the given code and message.
func NewR66Error(code rune, details string) *r66.Error {
	return &r66.Error{Code: code, Detail: details}
}

// NewR66Errorf returns a new r66.Error from the given code and message. The
// message accepts formats and arguments, similarly to the fmt package.
func NewR66Errorf(code rune, details string, args ...any) *r66.Error {
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

	var tErr *pipeline.Error
	if !errors.As(err, &tErr) {
		return NewR66Error(r66.Unknown, err.Error())
	}

	switch tErr.Code() {
	case types.TeOk:
		return NewR66Error(r66.CompleteOk, "")
	case types.TeUnknown:
		return NewR66Error(r66.Unknown, tErr.Redacted())
	case types.TeInternal:
		return NewR66Error(r66.Internal, tErr.Redacted())
	case types.TeUnimplemented:
		return NewR66Error(r66.Unimplemented, tErr.Redacted())
	case types.TeConnection:
		return NewR66Error(r66.ConnectionImpossible, tErr.Redacted())
	case types.TeConnectionReset:
		return NewR66Error(r66.Disconnection, tErr.Redacted())
	case types.TeUnknownRemote:
		return NewR66Error(r66.QueryRemotelyUnknown, tErr.Redacted())
	case types.TeExceededLimit:
		return NewR66Error(r66.ServerOverloaded, tErr.Redacted())
	case types.TeBadAuthentication:
		return NewR66Error(r66.BadAuthent, tErr.Redacted())
	case types.TeDataTransfer:
		return NewR66Error(r66.TransferError, tErr.Redacted())
	case types.TeIntegrity:
		return NewR66Error(r66.FinalOp, tErr.Redacted())
	case types.TeFinalization:
		return NewR66Error(r66.FinalOp, tErr.Redacted())
	case types.TeExternalOperation:
		return NewR66Error(r66.ExternalOperation, tErr.Redacted())
	case types.TeWarning:
		return NewR66Error(r66.Warning, tErr.Redacted())
	case types.TeStopped:
		return NewR66Error(r66.StoppedTransfer, tErr.Redacted())
	case types.TeCanceled:
		return NewR66Error(r66.CanceledTransfer, tErr.Redacted())
	case types.TeFileNotFound:
		return NewR66Error(r66.FileNotFound, tErr.Redacted())
	case types.TeForbidden:
		return NewR66Error(r66.FileNotAllowed, tErr.Redacted())
	case types.TeBadSize:
		return NewR66Error(r66.SizeNotAllowed, tErr.Redacted())
	case types.TeShuttingDown:
		return NewR66Error(r66.Shutdown, tErr.Redacted())
	default:
		return NewR66Error(r66.Unknown, tErr.Redacted())
	}
}

// FromR66Error takes an R66 error (most likely of type r66.Error) and returns
// the corresponding types.TransferError.
//
//nolint:funlen,gocyclo,cyclop // splitting the function would add complexity
func FromR66Error(err error, pip *pipeline.Pipeline) *pipeline.Error {
	var rErr *r66.Error
	if !errors.As(err, &rErr) {
		return pipeline.NewError(types.TeUnknownRemote,
			fmt.Sprintf("Error on remote partner: %v", err))
	}

	details := fmt.Sprintf("Error on remote partner: %v", rErr.Detail)

	switch rErr.Code {
	case r66.InitOk, r66.PreProcessingOk, r66.TransferOk, r66.PostProcessingOk,
		r66.CompleteOk:
		return nil
	case r66.ConnectionImpossible:
		return pipeline.NewError(types.TeConnection, details)
	case r66.ServerOverloaded:
		return pipeline.NewError(types.TeExceededLimit, details)
	case r66.BadAuthent:
		return pipeline.NewError(types.TeBadAuthentication, details)
	case r66.ExternalOperation:
		return pipeline.NewError(types.TeExternalOperation, details)
	case r66.TransferError:
		return pipeline.NewError(types.TeDataTransfer, details)
	case r66.MD5Error:
		return pipeline.NewError(types.TeIntegrity, details)
	case r66.Disconnection:
		return pipeline.NewError(types.TeConnectionReset, details)
	case r66.RemoteShutdown:
		return pipeline.NewError(types.TeShuttingDown, details)
	case r66.FinalOp:
		return pipeline.NewError(types.TeFinalization, details)
	case r66.Unimplemented:
		return pipeline.NewError(types.TeUnimplemented, details)
	case r66.Shutdown:
		return pipeline.NewError(types.TeShuttingDown, details)
	case r66.RemoteError:
		return pipeline.NewError(types.TeUnknownRemote, details)
	case r66.Internal:
		return pipeline.NewError(types.TeInternal, details)
	case r66.Warning:
		return pipeline.NewError(types.TeWarning, details)
	case r66.Unknown:
		return pipeline.NewError(types.TeUnknownRemote, details)
	case r66.FileNotFound:
		return pipeline.NewError(types.TeFileNotFound, details)
	case r66.CommandNotFound:
		return pipeline.NewError(types.TeUnimplemented, details)
	case r66.IncorrectCommand:
		return pipeline.NewError(types.TeUnimplemented, details)
	case r66.FileNotAllowed:
		return pipeline.NewError(types.TeForbidden, details)
	case r66.SizeNotAllowed:
		return pipeline.NewError(types.TeForbidden, details)
	case r66.StoppedTransfer:
		if pErr := pip.Pause(context.Background()); pErr != nil {
			return pipeline.NewErrorWith(types.TeInternal, "failed to pause transfer", pErr)
		}

		return nil
	case r66.CanceledTransfer:
		if cErr := pip.Cancel(context.Background()); cErr != nil {
			return pipeline.NewErrorWith(types.TeInternal, "failed to cancel transfer", cErr)
		}

		return nil
	default:
		return pipeline.NewError(types.TeUnknownRemote, details)
	}
}
