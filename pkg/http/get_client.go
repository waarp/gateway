package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/http/httpconst"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

type getClient struct {
	pip       *pipeline.Pipeline
	transport *http.Transport

	resp   *http.Response
	ctx    context.Context
	cancel context.CancelFunc
}

func (g *getClient) Request() *types.TransferError {
	g.ctx, g.cancel = context.WithCancel(context.Background())

	scheme := "http://"
	if g.transport.TLSClientConfig != nil {
		scheme = "https://"
	}

	addr, err := conf.GetRealAddress(g.pip.TransCtx.RemoteAgent.Address)
	if err != nil {
		g.pip.Logger.Errorf("Failed to retrieve HTTP address: %s", err)
		return types.NewTransferError(types.TeInternal, "failed to retrieve HTTP address")
	}

	url := scheme + path.Join(addr, g.pip.TransCtx.Transfer.RemotePath)
	req, err := http.NewRequestWithContext(g.ctx, http.MethodGet, url, nil)
	if err != nil {
		g.pip.Logger.Errorf("Failed to make HTTP request: %s", err)
		return types.NewTransferError(types.TeInternal,
			"failed to make HTTP request")
	}
	req.SetBasicAuth(g.pip.TransCtx.RemoteAccount.Login, string(g.pip.TransCtx.RemoteAccount.Password))

	req.Header.Set(httpconst.TransferID, fmt.Sprint(g.pip.TransCtx.Transfer.ID))
	req.Header.Set(httpconst.RuleName, g.pip.TransCtx.Rule.Name)
	makeRange(req, g.pip.TransCtx.Transfer)
	req.Trailer = make(http.Header)
	req.Trailer.Set(httpconst.TransferStatus, "")

	client := &http.Client{Transport: g.transport}
	g.resp, err = client.Do(req) //nolint:bodyclose //body is closed in another function
	if err != nil {
		g.pip.Logger.Errorf("Failed to connect to remote host: %s", err)
		return types.NewTransferError(types.TeConnection, "failed to connect to remote host")
	}
	switch g.resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		return g.getSizeProgress()
	default:
		return getRemoteStatus(g.resp.Header, g.pip)
	}
}

func (g *getClient) getSizeProgress() *types.TransferError {
	cols := []string{"progression"}
	trans := g.pip.TransCtx.Transfer

	progress, filesize, err := getContentRange(g.resp.Header)
	if err != nil {
		g.pip.Logger.Errorf("Failed to parse transfer progress/filesize: %s", err)
		return types.NewTransferError(types.TeBadSize, "failed to parse transfer info")
	}
	if trans.Filesize < 0 {
		cols = append(cols, "filesize")
		trans.Filesize = filesize
	}
	trans.Progress = progress

	if err := g.pip.DB.Update(trans).Cols(cols...).Run(); err != nil {
		g.pip.Logger.Errorf("Failed to update transfer file attributes: %s", err)
		return types.NewTransferError(types.TeInternal, "database error")
	}
	return nil
}

func (g *getClient) Data(stream pipeline.DataStream) *types.TransferError {
	defer g.resp.Body.Close()
	if _, err := io.Copy(stream, g.resp.Body); err != nil {
		g.pip.Logger.Errorf("Failed to read from remote HTTP file: %s", err)
		return types.NewTransferError(types.TeDataTransfer,
			"failed to read from remote HTTP file")
	}

	if err := g.resp.Body.Close(); err != nil {
		g.pip.Logger.Errorf("Failed to close remote HTTP file: %s", err)
		return types.NewTransferError(types.TeDataTransfer,
			"failed to close remote HTTP file")
	}

	return getRemoteStatus(g.resp.Trailer, g.pip)
}

func (g *getClient) EndTransfer() *types.TransferError {
	if g.resp != nil {
		_ = g.resp.Body.Close()
	}
	// nothing more to do at this point
	return nil
}

func (g *getClient) SendError(*types.TransferError) {
	if g.resp != nil {
		_ = g.resp.Body.Close()
	}
	g.cancel()
}
