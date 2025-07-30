package pipeline

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/statemachine"
)

func (p *Pipeline) stateErr(fun string, currentState statemachine.State) *Error {
	return p.internalErrorWithMsg(types.TeInternal,
		fmt.Sprintf("cannot call %q while in state %q", fun, currentState),
		"internal logic error", nil)
}

func (p *Pipeline) internalError(code types.TransferErrorCode, msg string,
	cause error,
) *Error {
	return p.internalErrorWithMsg(code, msg, msg, cause)
}

func (p *Pipeline) internalErrorWithMsg(code types.TransferErrorCode, msg, extMsg string,
	cause error,
) *Error {
	p.errOnce.Do(func() {
		p.processError(code, msg, extMsg, cause)
	})

	return p.storedErr
}

func (p *Pipeline) externalError(code types.TransferErrorCode, detail string) {
	p.errOnce.Do(func() {
		p.processError(code, detail, detail, nil)
	})
}

func (p *Pipeline) processError(code types.TransferErrorCode, msg, extMsg string,
	cause error,
) {
	if err := p.machine.Transition(stateError); err != nil {
		p.Logger.Warningf("Failed to transition to %q state: %v", stateError, err)
	}

	err := NewErrorWith(code, msg, cause)
	p.storedErr = NewError(code, extMsg)

	fullMsg := msg
	if cause != nil {
		fullMsg = fmt.Sprintf("%s: %s", msg, cause)
	}

	p.Logger.Error(fullMsg)
	p.stop()

	if p.Trace.OnError != nil {
		p.Trace.OnError(err)
	}

	p.TransCtx.Transfer.ErrCode = code
	p.TransCtx.Transfer.ErrDetails = fullMsg

	if dbErr := p.UpdateTrans(); dbErr != nil {
		p.Logger.Errorf("Failed to update transfer error: %s", dbErr)
	}

	p.errorTasks()
	p.doneErr(types.StatusError)
}

func (f *FileStream) stateErr(fun string, currentState statemachine.State) *Error {
	return f.internalErrorWithMsg(types.TeInternal,
		fmt.Sprintf("cannot call %s while in state %s", fun, currentState),
		"internal logic error", nil)
}

func (f *FileStream) internalError(code types.TransferErrorCode, msg string,
	cause error,
) *Error {
	return f.internalErrorWithMsg(code, msg, msg, cause)
}

func (f *FileStream) internalErrorWithMsg(code types.TransferErrorCode, msg, extMsg string,
	cause error,
) *Error {
	f.errOnce.Do(func() {
		if err := f.file.Close(); err != nil {
			f.Logger.Warningf("Failed to close file: %v", err)
		}

		f.processError(code, msg, extMsg, cause)
	})

	return f.storedErr
}
