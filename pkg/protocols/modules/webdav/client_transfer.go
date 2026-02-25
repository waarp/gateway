package webdav

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path"

	"github.com/studio-b12/gowebdav"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
)

type clientTransfer struct {
	client  *gowebdav.Client
	pip     *pipeline.Pipeline
	errChan *protoutils.ErrChan
}

func (c *clientTransfer) handleError(_ string, rq *http.Request) {
	go func() {
		if err := <-c.errChan.Receive(); err != nil {
			rq.Body.Close()
		}
	}()
}

func (c *clientTransfer) Request() *pipeline.Error {
	if err := c.client.Connect(); err != nil {
		return pipeline.NewErrorWith(types.TeConnection, "failed to connect to WebDAV server", err)
	}

	return nil
}

func (c *clientTransfer) Send(file protocol.SendFile) *pipeline.Error {
	defer c.errChan.Close()
	c.client.SetInterceptor(c.handleError)

	filepath := c.pip.TransCtx.Transfer.RemotePath
	dir := path.Dir(filepath)
	filePerms := os.FileMode(conf.GlobalConfig.Paths.FilePerms)
	dirPerms := os.FileMode(conf.GlobalConfig.Paths.DirPerms)

	if dirInfo, err := c.client.Stat(dir); errors.Is(err, os.ErrNotExist) {
		if err = c.client.MkdirAll(dir, dirPerms); err != nil {
			return pipeline.NewErrorWith(types.TeDataTransfer, "failed to create parent directory", err)
		}
	} else if err != nil {
		return pipeline.NewErrorWith(types.TeDataTransfer, "failed to check parent directory", err)
	} else if !dirInfo.IsDir() {
		return pipeline.NewError(types.TeDataTransfer, "parent path is not a directory")
	}

	if err := c.client.WriteStream(filepath, file, filePerms); err != nil {
		return pipeline.NewErrorWith(types.TeDataTransfer, "WebDAV PUT request failed", err)
	}

	return nil
}

func (c *clientTransfer) Receive(file protocol.ReceiveFile) *pipeline.Error {
	defer c.errChan.Close()

	remotePath := c.pip.TransCtx.Transfer.RemotePath
	offset := c.pip.TransCtx.Transfer.Progress

	var (
		rd  io.ReadCloser
		err error
	)

	if offset <= 0 {
		rd, err = c.client.ReadStream(remotePath)
	} else {
		rd, err = c.client.ReadStreamRange(remotePath, offset, -1)
	}

	if err != nil {
		return pipeline.NewErrorWith(types.TeDataTransfer, "WebDAV GET request failed", err)
	}
	defer rd.Close()

	if _, err = io.Copy(file, rd); err != nil {
		return pipeline.NewErrorWith(types.TeDataTransfer, "failed to retrieve data", err)
	}

	return nil
}

func (c *clientTransfer) EndTransfer() *pipeline.Error {
	return nil
}

func (c *clientTransfer) SendError(code types.TransferErrorCode, msg string) {
	c.errChan.Send(pipeline.NewError(code, msg))
}

//nolint:wrapcheck //no need to wrap here
func (c *clientTransfer) Delete(ctx context.Context, filepath string, recursive bool) error {
	c.client.SetInterceptor(func(_ string, rq *http.Request) {
		*rq = *rq.WithContext(ctx)
	})

	if recursive {
		return c.client.RemoveAll(filepath)
	}

	return c.client.Remove(filepath)
}
