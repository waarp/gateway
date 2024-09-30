package http

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/http/httptrace"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const resumeTimeout = 3 * time.Second

type postClient struct {
	pip     *pipeline.Pipeline
	client  *http.Client
	isHTTPS bool

	writer *io.PipeWriter
	req    *http.Request

	reqErr chan error
	resp   chan *http.Response
	cancel func()
}

func (p *postClient) checkResume(url string) *pipeline.Error {
	if p.pip.TransCtx.Transfer.Progress == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), resumeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		p.pip.Logger.Error("Failed to make head HTTP request: %s", err)

		return pipeline.NewErrorWith(types.TeInternal, "failed to make head HTTP request", err)
	}

	var pwd string

	for _, a := range p.pip.TransCtx.RemoteAccountCreds {
		if a.Type == auth.Password {
			pwd = a.Value
		}
	}

	req.SetBasicAuth(p.pip.TransCtx.RemoteAccount.Login, pwd)
	req.Header.Set(httpconst.TransferID, p.pip.TransCtx.Transfer.RemoteTransferID)

	resp, err := p.client.Do(req)
	if err != nil {
		p.pip.Logger.Error("HTTP Head request failed: %s", err)

		return pipeline.NewErrorWith(types.TeInternal, "Head HTTP request failed", err)
	}

	defer discardResponse(resp)

	var prog int64

	switch resp.StatusCode {
	case http.StatusMethodNotAllowed:
		prog = 0
	case http.StatusNoContent:
		prog, _, err = getContentRange(resp.Header)
		if err != nil {
			p.pip.Logger.Error("Failed to parse response Content-Range: %s", err)

			return pipeline.NewErrorWith(types.TeInternal, "failed to parse response Content-Range", err)
		}
	default:
		p.pip.Logger.Error("HTTP Head replied with %s", resp.Status)

		return getRemoteError(resp.Header, resp.Body)
	}

	return p.updateTransForResume(prog)
}

func (p *postClient) updateTransForResume(prog int64) *pipeline.Error {
	if prog != p.pip.TransCtx.Transfer.Progress {
		p.pip.TransCtx.Transfer.Progress = prog
		if p.pip.TransCtx.Transfer.Step > types.StepData {
			p.pip.TransCtx.Transfer.Step = types.StepData
		}

		if err := p.pip.UpdateTrans(); err != nil {
			p.pip.Logger.Error("Failed to parse response Content-Range: %s", err)

			return pipeline.NewErrorWith(types.TeInternal, "database error", err)
		}
	}

	return nil
}

func (p *postClient) setRequestHeaders(req *http.Request) *pipeline.Error {
	var pwd string

	for _, a := range p.pip.TransCtx.RemoteAccountCreds {
		if a.Type == auth.Password {
			pwd = a.Value
		}
	}

	req.SetBasicAuth(p.pip.TransCtx.RemoteAccount.Login, pwd)

	ct := mime.TypeByExtension(path.Ext(p.pip.TransCtx.Transfer.LocalPath.Path))
	if ct == "" {
		ct = "application/octet-stream"
	}

	req.Header.Set("Content-Type", ct)
	req.Header.Set("Transfer-Encoding", "chunked")
	req.Header.Set("Expect", "100-continue")
	req.Header.Set(httpconst.TransferID, p.pip.TransCtx.Transfer.RemoteTransferID)
	req.Header.Set(httpconst.RuleName, p.pip.TransCtx.Rule.Name)
	req.Header.Set("Waarp-File-Size", utils.FormatInt(p.pip.TransCtx.Transfer.Filesize))
	makeContentRange(req.Header, p.pip.TransCtx.Transfer)

	if err := makeTransferInfo(req.Header, p.pip); err != nil {
		return err
	}

	// if err := makeFileInfo(req.Header, p.pip); err != nil {
	//	return err
	// }

	req.Trailer = make(http.Header)
	req.Trailer.Set(httpconst.TransferStatus, "")
	req.Trailer.Set(httpconst.ErrorCode, "")
	req.Trailer.Set(httpconst.ErrorMessage, "")

	fileInfo, err := fs.Stat(p.pip.TransCtx.FS, &p.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		p.pip.Logger.Error("Failed to retrieve local file size: %s", err)

		return pipeline.NewErrorWith(types.TeInternal,
			"failed to retrieve local file size", err)
	}

	req.Header.Set("Waarp-File-Size", utils.FormatInt(fileInfo.Size()))

	return nil
}

func (p *postClient) prepareRequest(ready chan struct{}) *pipeline.Error {
	scheme := schemeHTTP
	if p.isHTTPS {
		scheme = schemeHTTPS
	}

	addr := conf.GetRealAddress(p.pip.TransCtx.RemoteAgent.Address.Host,
		utils.FormatUint(p.pip.TransCtx.RemoteAgent.Address.Port))
	url := scheme + path.Join(addr, p.pip.TransCtx.Transfer.RemotePath)

	if err := p.checkResume(url); err != nil {
		return err
	}

	body, writer := io.Pipe()
	p.writer = writer

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		p.pip.Logger.Error("Failed to make HTTP request: %s", err)

		return pipeline.NewErrorWith(types.TeInternal, "failed to make HTTP request", err)
	}

	if err := p.setRequestHeaders(req); err != nil {
		return err
	}

	trace := httptrace.ClientTrace{
		Wait100Continue: func() { close(ready) },
	}

	req = req.WithContext(httptrace.WithClientTrace(req.Context(), &trace))
	p.req = req

	return nil
}

func (p *postClient) Request() *pipeline.Error {
	ready := make(chan struct{})
	if err := p.prepareRequest(ready); err != nil {
		return err
	}

	go func() {
		defer close(p.resp)
		defer close(p.reqErr)

		resp, err := p.client.Do(p.req) //nolint:bodyclose //body is closed in another function
		if err != nil {
			p.pip.Logger.Error("HTTP transfer failed: %s", err)
			p.reqErr <- err
		} else {
			p.resp <- resp
		}
	}()

	select {
	case <-ready:
		return nil
	case err := <-p.reqErr:
		return pipeline.NewErrorWith(types.TeConnection, "HTTP request failed", err)
	case resp := <-p.resp:
		defer resp.Body.Close() //nolint:errcheck // error is irrelevant at this point

		return parseRemoteError(resp.Header, resp.Body, types.TeConnection,
			"failed to connect to remote host")
	}
}

func (p *postClient) Receive(protocol.ReceiveFile) *pipeline.Error {
	panic("cannot receive files with a POST client")
}

func (p *postClient) Send(file protocol.SendFile) *pipeline.Error {
	_, copyErr := io.Copy(p.writer, file)
	if copyErr == nil {
		return nil
	}

	p.pip.Logger.Error("Failed to write to remote HTTP file: %s", copyErr)
	select {
	case reqErr := <-p.reqErr:
		return p.wrapAndSendError(reqErr, types.TeDataTransfer, "HTTP transfer failed")
	case resp := <-p.resp:
		//nolint:errcheck,gosec //error is irrelevant here
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return getRemoteStatus(resp.Header, resp.Body, p.pip)
		}
	default:
	}

	return p.wrapAndSendError(copyErr, types.TeDataTransfer,
		"failed to write to remote HTTP file")
}

func (p *postClient) EndTransfer() *pipeline.Error {
	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusDone))

	if err := p.writer.Close(); err != nil {
		p.pip.Logger.Warning("Error while closing file pipe writer: %v", err)
	}

	select {
	case err := <-p.reqErr:
		return p.wrapAndSendError(err, types.TeDataTransfer, "HTTP transfer failed")
	case resp := <-p.resp:
		if resp.StatusCode != http.StatusCreated {
			//nolint:errcheck,gosec //error is irrelevant here
			defer resp.Body.Close()

			return getRemoteStatus(resp.Header, resp.Body, p.pip)
		} else {
			p.discardResponse()
		}
	}

	return nil
}

func (p *postClient) SendError(code types.TransferErrorCode, details string) {
	if p.writer == nil || p.req == nil {
		return
	}

	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusError))
	p.req.Trailer.Set(httpconst.ErrorCode, code.String())
	p.req.Trailer.Set(httpconst.ErrorMessage, details)

	p.discardResponse()
}

func (p *postClient) Pause() *pipeline.Error {
	if p.writer == nil || p.req == nil {
		return nil
	}

	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusPaused))

	p.discardResponse()

	return nil
}

func (p *postClient) Cancel() *pipeline.Error {
	if p.writer == nil || p.req == nil {
		return nil
	}

	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusCancelled))

	p.discardResponse()

	return nil
}

func (p *postClient) wrapAndSendError(cause error, code types.TransferErrorCode,
	details string,
) *pipeline.Error {
	var tErr *pipeline.Error
	if !errors.As(cause, &tErr) {
		tErr = pipeline.NewError(code, details)
	}

	p.pip.Logger.Error("%s: %v", details, cause)
	p.SendError(tErr.Code(), tErr.Redacted())

	return tErr
}

//nolint:errcheck,gosec //errors are irrelevant here, we just want to discard the response
func (p *postClient) discardResponse() {
	p.writer.Close()
	discardResponse(<-p.resp)
}

//nolint:errcheck,gosec //errors are irrelevant here, we just want to discard the response
func discardResponse(resp *http.Response) {
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}
