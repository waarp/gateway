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

var errReadClosed = errors.New("read of closed body")

type uploadHandler struct {
	pip     *pipeline.Pipeline
	req     *http.Request
	reqBody io.ReadCloser
	resp    http.ResponseWriter
	reply   sync.Once
}

func (u *uploadHandler) Pause(ctx context.Context) error {
	return utils.RunWithCtx(ctx, func() error {
		u.reply.Do(func() {
			_ = u.reqBody.Close() //nolint:errcheck // error is irrelevant at this point
			u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusPaused))
			u.resp.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(u.resp, "transfer paused by user")
		})

		return nil
	})
}

func (u *uploadHandler) Interrupt(ctx context.Context) error {
	return utils.RunWithCtx(ctx, func() error {
		u.reply.Do(func() {
			_ = u.reqBody.Close() //nolint:errcheck // error is irrelevant at this point
			u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusInterrupted))
			u.resp.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(u.resp, "transfer interrupted by a server shutdown")
		})

		return nil
	})
}

func (u *uploadHandler) Cancel(ctx context.Context) error {
	return utils.RunWithCtx(ctx, func() error {
		u.reply.Do(func() {
			_ = u.reqBody.Close() //nolint:errcheck // error is irrelevant at this point
			u.resp.Header().Set(httpconst.TransferStatus, string(types.StatusCancelled))
			u.resp.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(u.resp, "transfer canceled by user")
		})

		return nil
	})
}

func (u *uploadHandler) sendError(status int, err *pipeline.Error) {
	sendServerError(u.pip, u.req, u.resp, &u.reply, status, err)
}

func (u *uploadHandler) handleError(err *pipeline.Error) bool {
	if err == nil {
		return false
	}

	u.sendError(http.StatusInternalServerError, err)

	return true
}

func (u *uploadHandler) run() {
	if !setServerTransferInfo(u.pip, u.req.Header, u.sendError) {
		return
	}

	// if !setServerFileInfo(u.pip, u.req.Header, u.sendError) {
	// 	return
	// }

	if pErr := u.pip.PreTasks(); u.handleError(pErr) {
		return
	}

	file, fErr := u.pip.StartData()
	if u.handleError(fErr) {
		return
	}

	if _, err := io.Copy(file, u.reqBody); err != nil {
		var cErr *pipeline.Error
		if !errors.As(err, &cErr) {
			cErr = pipeline.NewErrorWith(types.TeDataTransfer, "failed to copy data", err)
		}

		u.handleError(cErr)

		return
	}

	if err := getRemoteStatus(u.req.Trailer, nil, u.pip); err != nil {
		u.sendError(http.StatusBadRequest, err)

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

func (b *postBody) Read(p []byte) (int, error) {
	done := make(chan struct{})

	var (
		n   int
		err error
	)

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

	if err := b.src.Close(); err != nil {
		return fmt.Errorf("failed to close request body: %w", err)
	}

	return nil
}
