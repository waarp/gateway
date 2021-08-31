package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/http/httpconst"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

type downloadHandler struct {
	pip   *pipeline.Pipeline
	req   *http.Request
	resp  http.ResponseWriter
	reply sync.Once
}

func (d *downloadHandler) Pause(ctx context.Context) error {
	d.reply.Do(func() {
		d.pip.Pause()
		_ = d.req.Body.Close()
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusPaused))
		d.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(d.resp, "transfer paused by user")
	})
	return ctx.Err()
}

func (d *downloadHandler) Interrupt(ctx context.Context) error {
	d.reply.Do(func() {
		d.pip.Interrupt()
		_ = d.req.Body.Close()
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusInterrupted))
		d.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(d.resp, "transfer interrupted by a server shutdown")
	})
	return ctx.Err()
}

func (d *downloadHandler) Cancel(ctx context.Context) error {
	d.reply.Do(func() {
		d.pip.Cancel()
		_ = d.req.Body.Close()
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusCancelled))
		d.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(d.resp, "transfer cancelled by user")
	})
	return ctx.Err()
}

func runDownload(r *http.Request, w http.ResponseWriter, running *service.TransferMap,
	pip *pipeline.Pipeline) {
	down := &downloadHandler{
		pip:  pip,
		req:  r,
		resp: w,
	}
	running.Add(pip.TransCtx.Transfer.ID, down)
	defer running.Delete(pip.TransCtx.Transfer.ID)

	down.run()
}

func (d *downloadHandler) handleEarlyError(err *types.TransferError) bool {
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
		d.resp.WriteHeader(http.StatusInternalServerError)
	})
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
	makeContentRange(d.resp.Header(), d.pip.TransCtx.Transfer)

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
	if pErr := d.pip.PreTasks(); d.handleEarlyError(pErr) {
		return
	}

	file, fErr := d.pip.StartData()
	if d.handleEarlyError(fErr) {
		return
	}

	d.makeHeaders()

	if _, err := io.Copy(d.resp, file); err != nil {
		cErr := types.NewTransferError(types.TeDataTransfer, "failed to copy data")
		d.handleLateError(cErr)
		return
	}
	_ = d.req.Body.Close()

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
