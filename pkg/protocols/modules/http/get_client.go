package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type getClient struct {
	pip       *pipeline.Pipeline
	transport *http.Transport
	isHTTPS   bool

	resp   *http.Response
	ctx    context.Context
	cancel context.CancelFunc
}

//nolint:funlen //it's fine as is (for now)
func (g *getClient) Request() *pipeline.Error {
	g.ctx, g.cancel = context.WithCancel(context.Background())

	scheme := schemeHTTP
	if g.isHTTPS {
		scheme = schemeHTTPS
	}

	addr := conf.GetRealAddress(g.pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(g.pip.TransCtx.RemoteAgent.Address.Port))
	url := scheme + path.Join(addr, g.pip.TransCtx.Transfer.RemotePath)

	req, reqErr := http.NewRequestWithContext(g.ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		g.pip.Logger.Error("Failed to make HTTP request: %s", reqErr)

		return pipeline.NewErrorWith(types.TeInternal,
			"failed to make HTTP request", reqErr)
	}

	var pwd string

	for _, a := range g.pip.TransCtx.RemoteAccountCreds {
		if a.Type == auth.Password {
			pwd = a.Value
		}
	}

	req.SetBasicAuth(g.pip.TransCtx.RemoteAccount.Login, pwd)

	if err := makeTransferInfo(req.Header, g.pip); err != nil {
		return err
	}

	req.Header.Set(httpconst.TransferID, g.pip.TransCtx.Transfer.RemoteTransferID)
	req.Header.Set(httpconst.RuleName, g.pip.TransCtx.Rule.Name)
	makeRange(req, g.pip.TransCtx.Transfer)
	req.Trailer = make(http.Header)
	req.Trailer.Set(httpconst.TransferStatus, "")

	client := &http.Client{Transport: g.transport}

	g.resp, reqErr = client.Do(req) //nolint:bodyclose //body is closed in another function
	if reqErr != nil {
		g.pip.Logger.Error("Failed to connect to remote host: %s", reqErr)

		return pipeline.NewErrorWith(types.TeConnection, "failed to connect to remote host", reqErr)
	}

	switch g.resp.StatusCode {
	case http.StatusOK, http.StatusPartialContent:
		analytics.AddConnection()

		if err := g.getSizeProgress(); err != nil {
			analytics.SubConnection()

			return err
		}

		return nil
	default:
		return getRemoteStatus(g.resp.Header, g.resp.Body, g.pip)
	}
}

func (g *getClient) getSizeProgress() *pipeline.Error {
	cols := []string{"progress"}
	trans := g.pip.TransCtx.Transfer

	progress, fileSize, rangeErr := getContentRange(g.resp.Header)
	if rangeErr != nil {
		return g.wrapAndSendError(rangeErr, types.TeBadSize, "failed to parse transfer info")
	}

	if trans.Filesize < 0 {
		cols = append(cols, "filesize")
		trans.Filesize = fileSize
	}

	trans.Progress = progress

	if err := g.pip.DB.Update(trans).Cols(cols...).Run(); err != nil {
		g.pip.Logger.Error("Failed to update transfer progress: %s", err)

		return g.wrapAndSendError(err, types.TeInternal, "database error")
	}

	return nil
}

func (g *getClient) Send(protocol.SendFile) *pipeline.Error {
	panic("cannot send file with a GET client")
}

func (g *getClient) Receive(file protocol.ReceiveFile) *pipeline.Error {
	if _, err := io.Copy(file, g.resp.Body); err != nil {
		return g.wrapAndSendError(err, types.TeDataTransfer, "Failed to read from remote HTTP file")
	}

	if err := g.resp.Body.Close(); err != nil {
		return g.wrapAndSendError(err, types.TeDataTransfer, "Failed to close remote HTTP file")
	}

	if err := getRemoteStatus(g.resp.Trailer, g.resp.Body, g.pip); err != nil {
		return g.wrapAndSendError(err, types.TeDataTransfer, "Failed to get remote HTTP status")
	}

	return nil
}

func (g *getClient) EndTransfer() *pipeline.Error {
	defer analytics.SubConnection()

	if g.resp != nil {
		if err := g.resp.Body.Close(); err != nil {
			g.pip.Logger.Warning("Error while closing the response body: %v", err)
		}
	}
	// nothing more to do at this point
	return nil
}

func (g *getClient) SendError(types.TransferErrorCode, string) {
	defer analytics.SubConnection()

	if g.resp != nil {
		_ = g.resp.Body.Close() //nolint:errcheck // error is irrelevant at this point
	}

	if g.cancel != nil {
		g.cancel()
	}
}

func (g *getClient) wrapAndSendError(cause error, code types.TransferErrorCode, details string,
) *pipeline.Error {
	var tErr *pipeline.Error
	if !errors.As(cause, &tErr) {
		tErr = pipeline.NewError(code, details)
	}

	g.pip.Logger.Error("%s: %v", details, cause)
	g.SendError(tErr.Code(), tErr.Redacted())

	return tErr
}
