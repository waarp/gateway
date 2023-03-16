package http

import (
	"context"
	"io"
	"net/http"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type getClient struct {
	pip       *pipeline.Pipeline
	transport *http.Transport
	isHTTPS   bool

	resp   *http.Response
	ctx    context.Context //nolint:containedctx //FIXME move the context to a function parameter
	cancel context.CancelFunc
}

func (g *getClient) Request() (tErr *types.TransferError) {
	g.ctx, g.cancel = context.WithCancel(context.Background())

	scheme := "http://"
	if g.isHTTPS {
		scheme = "https://"
	}

	addr := g.pip.TransCtx.RemoteAgent.Address
	url := scheme + path.Join(addr, g.pip.TransCtx.Transfer.RemotePath)

	req, err := http.NewRequestWithContext(g.ctx, http.MethodGet, url, nil)
	if err != nil {
		g.pip.Logger.Error("Failed to make HTTP request: %s", err)

		return types.NewTransferError(types.TeInternal,
			"failed to make HTTP request")
	}

	req.SetBasicAuth(g.pip.TransCtx.RemoteAccount.Login, string(g.pip.TransCtx.RemoteAccount.Password))

	if err := makeTransferInfo(req.Header, g.pip); err != nil {
		return err
	}

	req.Header.Set(httpconst.TransferID, g.pip.TransCtx.Transfer.RemoteTransferID)
	req.Header.Set(httpconst.RuleName, g.pip.TransCtx.Rule.Name)
	makeRange(req, g.pip.TransCtx.Transfer)
	req.Trailer = make(http.Header)
	req.Trailer.Set(httpconst.TransferStatus, "")

	client := &http.Client{Transport: g.transport}

	g.resp, err = client.Do(req) //nolint:bodyclose //body is closed in another function
	if err != nil {
		g.pip.Logger.Error("Failed to connect to remote host: %s", err)

		return types.NewTransferError(types.TeConnection, "failed to connect to remote host")
	}

	defer func() {
		if tErr != nil {
			g.SendError(tErr)
		}
	}()

	switch g.resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		/* if err := setFileInfo(g.pip, g.resp.Header); err != nil {
		return err
		} */
		return g.getSizeProgress()
	default:
		return getRemoteStatus(g.resp.Header, g.resp.Body, g.pip)
	}
}

func (g *getClient) getSizeProgress() *types.TransferError {
	cols := []string{"progress"}
	trans := g.pip.TransCtx.Transfer

	progress, filesize, err := getContentRange(g.resp.Header)
	if err != nil {
		g.pip.Logger.Error("Failed to parse transfer progress/filesize: %s", err)

		return types.NewTransferError(types.TeBadSize, "failed to parse transfer info")
	}

	if trans.Filesize < 0 {
		cols = append(cols, "filesize")
		trans.Filesize = filesize
	}

	trans.Progress = progress

	if err := g.pip.DB.Update(trans).Cols(cols...).Run(); err != nil {
		g.pip.Logger.Error("Failed to update transfer file attributes: %s", err)

		return types.NewTransferError(types.TeInternal, "database error")
	}

	/* if err := setFileInfo(g.pip, g.resp.Header); err != nil {
		return err
	} */

	return nil
}

func (g *getClient) Data(stream pipeline.DataStream) (tErr *types.TransferError) {
	defer func() {
		if tErr != nil {
			g.SendError(tErr)
		}
	}()

	if _, err := io.Copy(stream, g.resp.Body); err != nil {
		g.pip.Logger.Error("Failed to read from remote HTTP file: %s", err)

		return types.NewTransferError(types.TeDataTransfer,
			"failed to read from remote HTTP file")
	}

	if err := g.resp.Body.Close(); err != nil {
		g.pip.Logger.Error("Failed to close remote HTTP file: %s", err)

		return types.NewTransferError(types.TeDataTransfer,
			"failed to close remote HTTP file")
	}

	return getRemoteStatus(g.resp.Trailer, g.resp.Body, g.pip)
}

func (g *getClient) EndTransfer() *types.TransferError {
	if g.resp != nil {
		if err := g.resp.Body.Close(); err != nil {
			g.pip.Logger.Warning("Error while closing the response body: %v", err)
		}
	}
	// nothing more to do at this point
	return nil
}

func (g *getClient) SendError(*types.TransferError) {
	if g.resp != nil {
		_ = g.resp.Body.Close() //nolint:errcheck // error is irrelevant at this point
	}

	if g.cancel != nil {
		g.cancel()
	}
}
