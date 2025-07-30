package pesit

import (
	"context"
	"errors"
	"io"
	"time"

	"code.waarp.fr/lib/pesit"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const stopTimeout = 5 * time.Second

func stopReceived(pip *pipeline.Pipeline) func(pesit.StopCause, error) {
	return func(cause pesit.StopCause, err error) {
		var pErr pesit.Diagnostic
		errors.As(err, &pErr)

		if pErr.GetMessage() != "" {
			pip.Logger.Infof("Transfer interrupted by partner: %s", pErr.GetMessage())
		}

		ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
		defer cancel()

		switch cause {
		case pesit.StopSuspend:
			if pErr := pip.Pause(ctx); pErr != nil {
				pip.Logger.Errorf("Failed to pause transfer: %v", pErr)
			}
		case pesit.StopCancel:
			if pErr.IsSuccess() {
				if cErr := pip.Cancel(ctx); cErr != nil {
					pip.Logger.Errorf("Failed to cancel transfer: %v", cErr)
				}

				return
			}

			fallthrough
		default:
			pipErr := pesitErrToPipErr("error on remote partner", pErr)
			pip.SetError(pipErr.Code(), pipErr.Details())
		}
	}
}

func connectionAborted(pip *pipeline.Pipeline) func(error) {
	return func(err error) {
		var pErr pesit.Diagnostic
		errors.As(err, &pErr)

		pip.SetError(types.TeConnectionReset, pErr.GetMessage())
	}
}

func restartReceived(pip *pipeline.Pipeline) func(uint32, error) uint32 {
	return func(checkpoint uint32, _ error) uint32 {
		if pip.Stream == nil {
			return 0 // data transfer hasn't started yet
		}

		const checkpointSize = 1 // TODO: replace with real value once obtainable
		offset := checkpointSize * checkpoint

		newOff, err := pip.Stream.Seek(int64(offset), io.SeekStart)
		if err != nil {
			pip.Logger.Errorf("Restart request failed: %v", err)
		}

		return uint32(newOff)
	}
}

func checkpointRequestReceived(pip *pipeline.Pipeline) func(uint32) bool {
	return func(uint32) bool {
		if pip.Stream == nil {
			return false
		}

		if err := pip.Stream.Sync(); err != nil {
			pip.Logger.Errorf("Checkpoint validation failed: %v", err)

			return false
		}

		return true
	}
}
