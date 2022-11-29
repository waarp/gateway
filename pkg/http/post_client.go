package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptrace"
	"os"
	"path"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/http/httpconst"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const resumeTimeout = 3 * time.Second

type postClient struct {
	pip       *pipeline.Pipeline
	transport *http.Transport

	writer *io.PipeWriter
	req    *http.Request

	reqErr chan error
	resp   chan *http.Response
}

func (p *postClient) checkResume(url string) *types.TransferError {
	if p.pip.TransCtx.Transfer.Progress == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), resumeTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		p.pip.Logger.Error("Failed to make head HTTP request: %s", err)

		return types.NewTransferError(types.TeInternal, "failed to make head HTTP request")
	}

	req.SetBasicAuth(p.pip.TransCtx.RemoteAccount.Login, string(p.pip.TransCtx.RemoteAccount.Password))
	req.Header.Set(httpconst.TransferID, p.pip.TransCtx.Transfer.RemoteTransferID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		p.pip.Logger.Error("HTTP Head request failed: %s", err)

		return types.NewTransferError(types.TeInternal, "Head HTTP request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // this error is irrelevant

	var prog int64

	switch resp.StatusCode {
	case http.StatusMethodNotAllowed:
		prog = 0
	case http.StatusNoContent:
		prog, _, err = getContentRange(resp.Header)
		if err != nil {
			p.pip.Logger.Error("Failed to parse response Content-Range: %s", err)

			return types.NewTransferError(types.TeInternal, err.Error())
		}
	default:
		p.pip.Logger.Error("HTTP Head replied with %s", resp.Status)

		return getRemoteError(resp.Header, resp.Body)
	}

	return p.updateTransForResume(prog)
}

func (p *postClient) updateTransForResume(prog int64) *types.TransferError {
	if prog != p.pip.TransCtx.Transfer.Progress {
		p.pip.TransCtx.Transfer.Progress = prog
		if p.pip.TransCtx.Transfer.Step > types.StepData {
			p.pip.TransCtx.Transfer.Step = types.StepData
		}

		if err := p.pip.UpdateTrans(); err != nil {
			p.pip.Logger.Error("Failed to parse response Content-Range: %s", err)

			return types.NewTransferError(types.TeInternal, "database error")
		}
	}

	return nil
}

func (p *postClient) setRequestHeaders(req *http.Request) *types.TransferError {
	req.SetBasicAuth(p.pip.TransCtx.RemoteAccount.Login, string(p.pip.TransCtx.RemoteAccount.Password))

	ct := mime.TypeByExtension(filepath.Ext(p.pip.TransCtx.Transfer.LocalPath))
	if ct == "" {
		ct = "application/octet-stream"
	}

	req.Header.Set("Content-Type", ct)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set(httpconst.TransferID, p.pip.TransCtx.Transfer.RemoteTransferID)
	req.Header.Set(httpconst.RuleName, p.pip.TransCtx.Rule.Name)
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

	fileInfo, err := os.Stat(p.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		p.pip.Logger.Error("Failed to retrieve local file size: %s", err)

		return types.NewTransferError(types.TeInternal,
			"failed to retrieve local file size: %s")
	}

	req.Header.Set("Waarp-File-Size", fmt.Sprint(fileInfo.Size()))

	return nil
}

func (p *postClient) prepareRequest(ready chan struct{}) *types.TransferError {
	scheme := "http://"
	if p.transport.TLSClientConfig != nil {
		scheme = "https://"
	}

	addr := p.pip.TransCtx.RemoteAgent.Address
	url := scheme + path.Join(addr, p.pip.TransCtx.Transfer.RemotePath)

	if err := p.checkResume(url); err != nil {
		return err
	}

	body, writer := io.Pipe()
	p.writer = writer

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, body)
	if err != nil {
		p.pip.Logger.Error("Failed to make HTTP request: %s", err)

		return types.NewTransferError(types.TeInternal, "failed to make HTTP request")
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

func (p *postClient) Request() *types.TransferError {
	ready := make(chan struct{})
	if err := p.prepareRequest(ready); err != nil {
		return err
	}

	go func() {
		defer close(p.resp)
		defer close(p.reqErr)

		client := &http.Client{Transport: p.transport}

		resp, err := client.Do(p.req) //nolint:bodyclose //body is closed in another function
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
		return types.NewTransferError(types.TeConnection, "HTTP request failed: %v", err)
	case resp := <-p.resp:
		defer resp.Body.Close() //nolint:errcheck // error is irrelevant at this point

		return parseRemoteError(resp.Header, resp.Body, types.TeConnection,
			"failed to connect to remote host")
	}
}

func (p *postClient) Data(stream pipeline.DataStream) *types.TransferError {
	_, err := io.Copy(p.writer, stream)
	if err != nil {
		p.pip.Logger.Error("Failed to write to remote HTTP file: %s", err)
		select {
		case reqErr := <-p.reqErr:
			tErr := types.NewTransferError(types.TeDataTransfer, "HTTP transfer failed", reqErr)
			p.SendError(tErr)

			return tErr
		case resp := <-p.resp:
			if cErr := resp.Body.Close(); cErr != nil {
				p.pip.Logger.Warning("Error while closing response body: %v", cErr)
			}

			if resp.StatusCode != http.StatusCreated {
				return getRemoteStatus(resp.Header, resp.Body, p.pip)
			}

			return types.NewTransferError(types.TeDataTransfer,
				"failed to write to remote HTTP file")
		default:
			var tErr *types.TransferError
			if !errors.As(err, &tErr) {
				tErr = types.NewTransferError(types.TeDataTransfer,
					"failed to write to remote HTTP file")
			}

			p.SendError(tErr)

			return tErr
		}
	}

	return nil
}

func (p *postClient) EndTransfer() *types.TransferError {
	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusDone))

	if err := p.writer.Close(); err != nil {
		p.pip.Logger.Warning("Error while closing file pipe writer: %v", err)
	}

	select {
	case err := <-p.reqErr:
		return types.NewTransferError(types.TeUnknownRemote, "HTTP request failed", err)
	case resp := <-p.resp:
		if err := resp.Body.Close(); err != nil {
			p.pip.Logger.Warning("Error while closing response body: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			return getRemoteStatus(resp.Header, resp.Body, p.pip)
		}
	}

	return nil
}

func (p *postClient) SendError(err *types.TransferError) {
	if p.writer == nil {
		return
	}

	defer p.writer.Close() //nolint:errcheck // error is irrelevant at this point

	if p.req == nil {
		return
	}

	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusError))
	p.req.Trailer.Set(httpconst.ErrorCode, err.Code.String())
	p.req.Trailer.Set(httpconst.ErrorMessage, err.Details)
}

func (p *postClient) Pause() *types.TransferError {
	if p.writer == nil {
		return nil
	}

	defer p.writer.Close() //nolint:errcheck // error is irrelevant at this point

	if p.req == nil {
		return nil
	}

	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusPaused))

	return nil
}

func (p *postClient) Cancel() *types.TransferError {
	if p.writer == nil {
		return nil
	}

	defer p.writer.Close() //nolint:errcheck // error is irrelevant at this point

	if p.req == nil {
		return nil
	}

	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusCancelled))

	return nil
}
