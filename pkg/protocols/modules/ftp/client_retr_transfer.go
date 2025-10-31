package ftp

import (
	"context"
	"errors"

	"code.waarp.fr/lib/goftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type clientRetrTransfer struct {
	client *goftp.Client
	pip    *pipeline.Pipeline

	trans *goftp.RetrieveTransfer
}

func (t *clientRetrTransfer) Request() *pipeline.Error {
	path := t.pip.TransCtx.Transfer.RemotePath
	offset := t.pip.TransCtx.Transfer.Progress

	retr, err := t.client.Retrieve(path, offset)
	if err != nil {
		defer t.sendError()

		return toPipelineError(err, `FTP "RETRIEVE" transfer request failed`)
	}

	t.pip.TransCtx.Transfer.Filesize = retr.Size()
	t.trans = retr

	analytics.AddOutgoingConnection()

	return nil
}

func (t *clientRetrTransfer) Send(protocol.SendFile) *pipeline.Error {
	defer t.sendError()

	return pipeline.NewError(types.TeInternal,
		`cannot call "Send" on a receive transfer`)
}

func (t *clientRetrTransfer) Receive(file protocol.ReceiveFile) *pipeline.Error {
	analytics.AddOutgoingConnection()

	defer analytics.SubOutgoingConnection()

	if _, err := t.trans.WriteTo(file); err != nil {
		defer t.sendError()

		if errors.Is(err, goftp.ErrInvalidFileSize) {
			return pipeline.NewError(types.TeConnectionReset,
				"connection closed unexpectedly")
		}

		return toPipelineError(err, "FTP transfer receive failed")
	}

	return nil
}

func (t *clientRetrTransfer) EndTransfer() *pipeline.Error {
	defer analytics.SubOutgoingConnection()

	defer func() {
		if err := t.client.Close(); err != nil {
			t.pip.Logger.Warningf("Failed to close FTP connection: %v", err)
		}
	}()

	if err := t.trans.Done(); err != nil {
		return toPipelineError(err, "FTP transfer finalization failed")
	}

	return nil
}

func (t *clientRetrTransfer) SendError(types.TransferErrorCode, string) { t.sendError() }
func (t *clientRetrTransfer) sendError() {
	if t.trans != nil {
		defer analytics.SubOutgoingConnection()

		if err := t.trans.Abort(); err != nil {
			t.pip.Logger.Warningf("Failed to abort FTP transfer: %v", err)
		}
	}

	if err := t.client.Close(); err != nil {
		t.pip.Logger.Warningf("Failed to close FTP connection: %v", err)
	}
}

func (t *clientRetrTransfer) Delete(ctx context.Context, path string, recursive bool) error {
	return deleteRemoteCtx(ctx, t.client, path, recursive)
}
