package http

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
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
		p.pip.Logger.Errorf("Failed to make head HTTP request: %s", err)

		return types.NewTransferError(types.TeInternal, "failed to make head HTTP request")
	}

	req.SetBasicAuth(p.pip.TransCtx.RemoteAccount.Login, string(p.pip.TransCtx.RemoteAccount.Password))
	req.Header.Set(httpconst.TransferID, fmt.Sprint(p.pip.TransCtx.Transfer.ID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		p.pip.Logger.Errorf("HTTP Head request failed: %s", err)

		return types.NewTransferError(types.TeInternal, "Head HTTP request failed")
	}

	defer resp.Body.Close() //nolint:errcheck // this error is irrelevant

	var prog uint64

	switch resp.StatusCode {
	case http.StatusMethodNotAllowed:
		prog = 0
	case http.StatusNoContent:
		prog, _, err = getContentRange(resp.Header)
		if err != nil {
			p.pip.Logger.Errorf("Failed to parse response Content-Range: %s", err)

			return types.NewTransferError(types.TeInternal, err.Error())
		}
	default:
		p.pip.Logger.Errorf("HTTP Head replied with %s", resp.Status)

		return getRemoteError(resp.Header)
	}

	return p.updateTransForResume(prog)
}

func (p *postClient) updateTransForResume(prog uint64) *types.TransferError {
	if prog != p.pip.TransCtx.Transfer.Progress {
		cols := []string{"progression"}

		p.pip.TransCtx.Transfer.Progress = prog
		if p.pip.TransCtx.Transfer.Step > types.StepData {
			cols = append(cols, "step")
			p.pip.TransCtx.Transfer.Step = types.StepData
		}

		if err := p.pip.UpdateTrans(cols...); err != nil {
			p.pip.Logger.Errorf("Failed to parse response Content-Range: %s", err)

			return types.NewTransferError(types.TeInternal, "database error")
		}
	}

	return nil
}

func (p *postClient) prepareRequest(ready chan struct{}) *types.TransferError {
	scheme := "http://"
	if p.transport.TLSClientConfig != nil {
		scheme = "https://"
	}

	url := scheme + path.Join(p.pip.TransCtx.RemoteAgent.Address, p.pip.TransCtx.Transfer.RemotePath)
	if err := p.checkResume(url); err != nil {
		return err
	}

	body, writer := io.Pipe()
	p.writer = writer

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, body)
	if err != nil {
		p.pip.Logger.Errorf("Failed to make HTTP request: %s", err)

		return types.NewTransferError(types.TeInternal, "failed to make HTTP request")
	}

	req.SetBasicAuth(p.pip.TransCtx.RemoteAccount.Login, string(p.pip.TransCtx.RemoteAccount.Password))

	ct := mime.TypeByExtension(filepath.Ext(p.pip.TransCtx.Transfer.LocalPath))
	if ct == "" {
		ct = "application/octet-stream"
	}

	req.Header.Set("Content-Type", ct)
	req.Header.Set("Expect", "100-continue")
	req.Header.Set(httpconst.TransferID, fmt.Sprint(p.pip.TransCtx.Transfer.ID))
	req.Header.Set(httpconst.RuleName, p.pip.TransCtx.Rule.Name)
	makeContentRange(req.Header, p.pip.TransCtx.Transfer)

	req.Trailer = make(http.Header)
	req.Trailer.Set(httpconst.TransferStatus, "")
	req.Trailer.Set(httpconst.ErrorCode, "")
	req.Trailer.Set(httpconst.ErrorMessage, "")

	fileInfo, err := os.Stat(p.pip.TransCtx.Transfer.LocalPath)
	if err != nil {
		p.pip.Logger.Errorf("Failed to retrieve local file size: %s", err)

		return types.NewTransferError(types.TeInternal,
			"failed to retrieve local file size: %s")
	}

	req.Header.Set("Waarp-File-Size", fmt.Sprint(fileInfo.Size()))

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
			p.pip.Logger.Errorf("HTTP transfer failed: %s", err)
			p.reqErr <- err
		} else {
			p.resp <- resp
		}
	}()

	select {
	case <-ready:
		return nil
	case err := <-p.reqErr:
		return types.NewTransferError(types.TeConnection, "HTTP request failed: %s", err)
	case resp := <-p.resp:
		defer resp.Body.Close() //nolint:errcheck // error is irrelevant at this point

		msg := resp.Header.Get(httpconst.ErrorMessage)
		if body, err := ioutil.ReadAll(resp.Body); msg == "" && err == nil {
			msg = string(body)
		}

		return types.NewTransferError(types.TeConnection, "HTTP request failed: %s", msg)
	}
}

func (p *postClient) Data(stream pipeline.DataStream) *types.TransferError {
	_, err := io.Copy(p.writer, stream)
	if err != nil {
		p.pip.Logger.Errorf("Failed to write to remote HTTP file: %s", err)
		select {
		case err := <-p.reqErr:
			tErr := types.NewTransferError(types.TeDataTransfer, "HTTP transfer failed", err)
			p.SendError(tErr)

			return tErr
		case resp := <-p.resp:
			if cErr := resp.Body.Close(); cErr != nil {
				p.pip.Logger.Warningf("Error while closing response body: %v", cErr)
			}

			if resp.StatusCode != http.StatusCreated {
				return getRemoteStatus(resp.Header, p.pip)
			}

			return types.NewTransferError(types.TeDataTransfer,
				"failed to write to remote HTTP file")
		default:
			tErr := types.NewTransferError(types.TeDataTransfer,
				"failed to write to remote HTTP file")
			p.SendError(tErr)

			return tErr
		}
	}

	return nil
}

func (p *postClient) EndTransfer() *types.TransferError {
	p.req.Trailer.Set(httpconst.TransferStatus, string(types.StatusDone))

	if err := p.writer.Close(); err != nil {
		p.pip.Logger.Warningf("Error while closing file pipe writer: %v", err)
	}

	select {
	case err := <-p.reqErr:
		return types.NewTransferError(types.TeUnknownRemote, "HTTP request failed", err)
	case resp := <-p.resp:
		if err := resp.Body.Close(); err != nil {
			p.pip.Logger.Warningf("Error while closing response body: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			return getRemoteStatus(resp.Header, p.pip)
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
