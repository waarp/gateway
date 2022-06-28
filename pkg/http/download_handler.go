package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

type downloadHandler struct {
	pip   *pipeline.Pipeline
	req   *http.Request
	resp  http.ResponseWriter
	reply sync.Once
}

//nolint:dupl // factorizing would hurt readability
func (d *downloadHandler) Pause(ctx context.Context) error {
	d.reply.Do(func() {
		d.pip.Pause()
		_ = d.req.Body.Close() //nolint:errcheck // error is irrelevant at this point
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusPaused))
		d.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(d.resp, "transfer paused by user")
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

//nolint:dupl // factorizing would hurt readability
func (d *downloadHandler) Interrupt(ctx context.Context) error {
	d.reply.Do(func() {
		d.pip.Interrupt()
		_ = d.req.Body.Close() //nolint:errcheck // error is irrelevant at this point
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusInterrupted))
		d.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(d.resp, "transfer interrupted by a server shutdown")
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

//nolint:dupl // factorizing would hurt readability
func (d *downloadHandler) Cancel(ctx context.Context) error {
	d.reply.Do(func() {
		d.pip.Cancel()
		_ = d.req.Body.Close() //nolint:errcheck // error is irrelevant at this point
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusCancelled))
		d.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(d.resp, "transfer canceled by user")
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

func runDownload(r *http.Request, w http.ResponseWriter, running *service.TransferMap,
	pip *pipeline.Pipeline,
) {
	down := &downloadHandler{
		pip:  pip,
		req:  r,
		resp: w,
	}

	running.Add(pip.TransCtx.Transfer.ID, down)
	defer running.Delete(pip.TransCtx.Transfer.ID)

	down.run()
}

func (d *downloadHandler) sendEarlyError(status int, err *types.TransferError) {
	sendServerError(d.pip, d.req, d.resp, &d.reply, status, err)
}

func (d *downloadHandler) handleEarlyError(err *types.TransferError) bool {
	if err == nil {
		return false
	}

	d.sendEarlyError(http.StatusInternalServerError, err)

	return true
}

func (d *downloadHandler) handleLateError(err *types.TransferError) bool {
	if err == nil {
		return false
	}

	d.reply.Do(func() {
		select {
		case <-d.req.Context().Done():
			err = types.NewTransferError(types.TeConnectionReset, "connection closed by remote host")
		default:
		}

		d.pip.SetError(err)
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusError))
		d.resp.Header().Set(httpconst.ErrorCode, err.Code.String())
		d.resp.Header().Set(httpconst.ErrorMessage, err.Details)
	})

	return true
}

func (d *downloadHandler) makeHeaders() {
	d.resp.Header().Set(httpconst.TransferID, d.pip.TransCtx.Transfer.RemoteTransferID)
	d.resp.Header().Set(httpconst.RuleName, d.pip.TransCtx.Rule.Name)
	head := d.resp.Header()
	makeContentRange(head, d.pip.TransCtx.Transfer)
	// _ = makeFileInfo(head, d.pip)

	d.resp.Header().Add("Trailer", httpconst.TransferStatus)
	d.resp.Header().Add("Trailer", httpconst.ErrorCode)
	d.resp.Header().Add("Trailer", httpconst.ErrorMessage)

	if d.pip.TransCtx.Transfer.Progress != 0 {
		d.resp.WriteHeader(http.StatusPartialContent)
	} else {
		d.resp.WriteHeader(http.StatusOK)
	}
}

func (d *downloadHandler) run() {
	if !setServerTransferInfo(d.pip, d.req.Header, d.sendEarlyError) {
		return
	}

	if pErr := d.pip.PreTasks(); d.handleEarlyError(pErr) {
		return
	}

	file, fErr := d.pip.StartData()
	if d.handleEarlyError(fErr) {
		return
	}

	d.makeHeaders()

	if _, err := io.Copy(d.resp, file); err != nil {
		var tErr *types.TransferError
		if !errors.As(err, &tErr) {
			tErr = types.NewTransferError(types.TeDataTransfer, "failed to copy data")
		}

		d.handleLateError(tErr)

		return
	}

	if err := d.req.Body.Close(); err != nil {
		d.pip.Logger.Warning("Error while closing request body: %v", err)
	}

	if dErr := d.pip.EndData(); d.handleLateError(dErr) {
		return
	}

	if pErr := d.pip.PostTasks(); d.handleLateError(pErr) {
		return
	}

	if tErr := d.pip.EndTransfer(); d.handleLateError(tErr) {
		return
	}

	d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusDone))
}
