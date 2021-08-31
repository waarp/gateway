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

type uploadHandler struct {
	pip     *pipeline.Pipeline
	req     *http.Request
	reqBody io.ReadCloser
	resp    http.ResponseWriter
	reply   sync.Once
}

func (u *uploadHandler) Pause(ctx context.Context) error {
	u.reply.Do(func() {
		u.pip.Pause()
		_ = u.reqBody.Close()
		u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusPaused))
		u.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(u.resp, "transfer paused by user")
	})
	return ctx.Err()
}

func (u *uploadHandler) Interrupt(ctx context.Context) error {
	u.reply.Do(func() {
		u.pip.Interrupt()
		_ = u.reqBody.Close()
		u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusInterrupted))
		u.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(u.resp, "transfer interrupted by a server shutdown")
	})
	return ctx.Err()
}

func (u *uploadHandler) Cancel(ctx context.Context) error {
	u.reply.Do(func() {
		u.pip.Cancel()
		_ = u.reqBody.Close()
		u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusCancelled))
		u.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(u.resp, "transfer cancelled by user")
	})
	return ctx.Err()
}

func runUpload(r *http.Request, w http.ResponseWriter, running *service.TransferMap,
	pip *pipeline.Pipeline) {
	up := &uploadHandler{
		pip:     pip,
		req:     r,
		reqBody: &postBody{src: r.Body, closed: make(chan struct{})},
		resp:    w,
	}
	running.Add(pip.TransCtx.Transfer.ID, up)
	defer running.Delete(pip.TransCtx.Transfer.ID)

	up.run()
}

func (u *uploadHandler) sendError(code types.TransferErrorCode, msg string, status int) {
	u.reply.Do(func() {
		select {
		case <-u.req.Context().Done():
			code = types.TeConnectionReset
			msg = "connection closed by remote host"
		default:
		}

		u.pip.SetError(types.NewTransferError(code, msg))
		u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusError))
		u.resp.Header().Set(httpconst.ErrorCode, code.String())
		u.resp.Header().Set(httpconst.ErrorMessage, msg)
		u.resp.WriteHeader(status)
	})
}

func (u *uploadHandler) handleError(err *types.TransferError) bool {
	if err == nil {
		return false
	}
	u.sendError(err.Code, err.Details, http.StatusInternalServerError)

	return true
}

func (u *uploadHandler) run() {
	if pErr := u.pip.PreTasks(); u.handleError(pErr) {
		return
	}

	file, fErr := u.pip.StartData()
	if u.handleError(fErr) {
		return
	}

	if _, err := io.Copy(file, u.reqBody); err != nil {
		u.pip.Logger.Errorf("Failed to copy data: %s", err.Error())
		cErr := types.NewTransferError(types.TeDataTransfer, "failed to copy data")
		u.handleError(cErr)
		return
	}
	if err := getRemoteStatus(u.req.Trailer, u.pip); err != nil {
		u.sendError(err.Code, err.Details, http.StatusBadRequest)
		return
	}

	if dErr := u.pip.EndData(); u.handleError(dErr) {
		return
	}

	if pErr := u.pip.PostTasks(); u.handleError(pErr) {
		return
	}

	if tErr := u.pip.EndTransfer(); u.handleError(tErr) {
		return
	}

	u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusDone))
	u.resp.WriteHeader(http.StatusCreated)
}

type postBody struct {
	src    io.ReadCloser
	closed chan struct{}
}

func (b *postBody) Read(p []byte) (n int, err error) {
	done := make(chan struct{})
	go func() {
		n, err = b.src.Read(p)
		close(done)
	}()
	select {
	case <-done:
		return n, err
	case <-b.closed:
		return 0, fmt.Errorf("read of closed body")
	}
}

func (b *postBody) Close() error {
	close(b.closed)
	return nil
}
