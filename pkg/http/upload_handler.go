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

var errReadClosed = errors.New("read of closed body")

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
		_ = u.reqBody.Close() //nolint:errcheck // error is irrelevant at this point
		u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusPaused))
		u.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(u.resp, "transfer paused by user")
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

func (u *uploadHandler) Interrupt(ctx context.Context) error {
	u.reply.Do(func() {
		u.pip.Interrupt()
		_ = u.reqBody.Close() //nolint:errcheck // error is irrelevant at this point
		u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusInterrupted))
		u.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(u.resp, "transfer interrupted by a server shutdown")
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
}

func (u *uploadHandler) Cancel(ctx context.Context) error {
	u.reply.Do(func() {
		u.pip.Cancel()
		_ = u.reqBody.Close() //nolint:errcheck // error is irrelevant at this point
		u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusCancelled))
		u.resp.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(u.resp, "transfer canceled by user")
	})

	if err := ctx.Err(); err != nil {
		return context.Canceled
	}

	return nil
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
		if err != nil {
			if errors.Is(err, io.EOF) {
				return n, io.EOF
			}

			return n, fmt.Errorf("error while reading request body: %w", err)
		}

		return n, nil
	case <-b.closed:
		return 0, errReadClosed
	}
}

func (b *postBody) Close() error {
	close(b.closed)

	return nil
}
