package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type downloadHandler struct {
	pip   *pipeline.Pipeline
	req   *http.Request
	resp  http.ResponseWriter
	reply sync.Once
}

//nolint:dupl // factorizing would hurt readability
func (d *downloadHandler) Pause(ctx context.Context) error {
	return utils.RunWithCtx(ctx, func() error {
		d.reply.Do(func() {
			d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusPaused))
			d.resp.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(d.resp, "transfer paused by user")
		})

		return nil
	})
}

//nolint:dupl // factorizing would hurt readability
func (d *downloadHandler) Interrupt(ctx context.Context) error {
	return utils.RunWithCtx(ctx, func() error {
		d.reply.Do(func() {
			d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusInterrupted))
			d.resp.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(d.resp, "transfer interrupted by a server shutdown")
		})

		return nil
	})
}

//nolint:dupl // factorizing would hurt readability
func (d *downloadHandler) Cancel(ctx context.Context) error {
	return utils.RunWithCtx(ctx, func() error {
		d.reply.Do(func() {
			d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusCancelled))
			d.resp.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(d.resp, "transfer canceled by user")
		})

		return nil
	})
}

func (d *downloadHandler) sendEarlyError(status int, err error) {
	sendServerError(d.pip, d.req, d.resp, &d.reply, status, err)
}

func (d *downloadHandler) handleEarlyError(err error) bool {
	if err == nil {
		return false
	}

	d.sendEarlyError(http.StatusInternalServerError, err)

	return true
}

func (d *downloadHandler) handleLateError(err error) bool {
	if err == nil {
		return false
	}

	d.reply.Do(func() {
		select {
		case <-d.req.Context().Done():
			err = types.NewTransferError(types.TeConnectionReset, "connection closed by remote host")
		default:
		}

		tErr := asTransferError(err)

		d.pip.SetError(tErr)
		d.resp.Header().Set(httpconst.TransferStatus, string(types.StatusError))
		d.resp.Header().Set(httpconst.ErrorCode, tErr.Code.String())
		d.resp.Header().Set(httpconst.ErrorMessage, tErr.Details)
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
