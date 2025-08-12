package pesit

import (
	"errors"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func toPipErr(defaultCode types.TransferErrorCode, msg string, err error,
) *pipeline.Error {
	var pErr pesit.Diagnostic
	if !errors.As(err, &pErr) {
		return pipeline.NewErrorWith(defaultCode, msg, err)
	}

	return pesitErrToPipErr(msg, pErr)
}

func pesitErrToPipErr(msg string, pErr pesit.Diagnostic) *pipeline.Error {
	var code types.TransferErrorCode

	switch pErr.GetCode() {
	case pesit.CodeTransmissionError:
		code = types.TeDataTransfer
	case pesit.CodeSystemResourcesInsufficient, pesit.CodeUserResourcesInsufficient,
		pesit.CodeTooManyConnections, pesit.CodeTooManyRetries, pesit.CodeTooManyAcknowledgedCheckpoints:
		code = types.TeExceededLimit
	case pesit.CodeFileAlreadyExists, pesit.CodeFileBusy:
		code = types.TeForbidden
	case pesit.CodeFileNotExists:
		code = types.TeFileNotFound
	case pesit.CodeInsufficientDiskSpace, pesit.CodeFileSpaceOverflow,
		pesit.CodeExcedeedLength, pesit.CodeFileSizeExceeded:
		code = types.TeBadSize
	case pesit.CodeInternalError:
		code = types.TeInternal
	case pesit.CodeUnknownIdentification, pesit.CodeUnauthorizedCaller:
		code = types.TeBadAuthentication
	case pesit.CodeUnsupportedVersion, pesit.CodeMessageTypeRefused:
		code = types.TeUnimplemented
	case pesit.CodeNetworkError, pesit.CodeNetworkSaturation,
		pesit.CodeRemoteNetworkSaturation, pesit.CodeOtherConnectionError:
		code = types.TeConnection
	case pesit.CodeTimeoutExpired, pesit.CodeConnectedTimeoutExpired, pesit.CodeUserServiceTermination:
		code = types.TeConnectionReset
	case pesit.CodeVolontaryTermination:
		code = types.TeShuttingDown
	case pesit.CodeFileCloseError:
		code = types.TeFinalization
	case pesit.CodeOtherTransferError:
		code = types.TeExternalOperation
	default:
		return pipeline.NewErrorf(types.TeUnknownRemote, "%s: (%s) %s",
			msg, pErr.GetCode().String(), pErr.GetMessage())
	}

	return pipeline.NewErrorf(code, "%s: %s", msg, pErr.GetMessage())
}

//nolint:unparam //leave the default code parameter, we might need it later
func toPesitErr(defaultCode pesit.DiagnosticCode, err error) pesit.Diagnostic {
	var pErr *pipeline.Error
	if errors.As(err, &pErr) {
		return transErrToPesitErr(pErr)
	}

	return pesit.NewDiagnostic(defaultCode, err.Error())
}

func transErrToPesitErr(pErr *pipeline.Error) pesit.Diagnostic {
	var pCode pesit.DiagnosticCode

	switch pErr.Code() {
	case types.TeInternal:
		pCode = pesit.CodeInternalError
	case types.TeUnimplemented:
		pCode = pesit.CodeMessageTypeRefused
	case types.TeConnection:
		pCode = pesit.CodeNetworkError
	case types.TeConnectionReset:
		pCode = pesit.CodeAdminRequest
	case types.TeExceededLimit:
		pCode = pesit.CodeUserResourcesInsufficient
	case types.TeBadAuthentication:
		pCode = pesit.CodeUnknownIdentification
	case types.TeDataTransfer:
		pCode = pesit.CodeTransmissionError
	case types.TeFinalization:
		pCode = pesit.CodeFileCloseError
	case types.TeExternalOperation:
		pCode = pesit.CodeOtherTransferError
	case types.TeFileNotFound:
		pCode = pesit.CodeFileNotExists
	case types.TeForbidden:
		pCode = pesit.CodeUnauthorizedCaller
	case types.TeBadSize:
		pCode = pesit.CodeExcedeedLength
	case types.TeShuttingDown:
		pCode = pesit.CodeUserServiceTermination
	default:
		pCode = pesit.CodeOtherConnectionError
	}

	return pesit.NewDiagnostic(pCode, pErr.Details())
}
