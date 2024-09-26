package ftp

import (
	"code.waarp.fr/lib/goftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/analytics"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
)

type clientStorTransfer struct {
	client *goftp.Client
	pip    *pipeline.Pipeline

	trans *goftp.StoreTransfer
}

func (t *clientStorTransfer) Request() *pipeline.Error {
	path := t.pip.TransCtx.Transfer.RemotePath
	offset := t.pip.TransCtx.Transfer.Progress

	stor, err := t.client.Store(path, offset)
	if err != nil {
		defer t.sendError()

		return toPipelineError(err, `FTP "STORE" transfer request failed`)
	}

	if offset != 0 {
		t.pip.TransCtx.Transfer.Progress = stor.Offset()
	}

	t.trans = stor

	analytics.AddOutgoingConnection()

	return nil
}

func (t *clientStorTransfer) Send(file protocol.SendFile) *pipeline.Error {
	analytics.AddOutgoingConnection()
	defer analytics.SubOutgoingConnection()

	if _, err := t.trans.ReadFrom(file); err != nil {
		defer t.sendError()

		return toPipelineError(err, "FTP transfer send failed")
	}

	return nil
}

func (t *clientStorTransfer) Receive(protocol.ReceiveFile) *pipeline.Error {
	defer t.sendError()

	return pipeline.NewError(types.TeInternal,
		`cannot call "Receive" on a send transfer`)
}

func (t *clientStorTransfer) EndTransfer() *pipeline.Error {
	defer analytics.SubOutgoingConnection()

	defer func() {
		if err := t.client.Close(); err != nil {
			t.pip.Logger.Warning("Failed to close FTP connection: %v", err)
		}
	}()

	if err := t.trans.Done(); err != nil {
		return toPipelineError(err, "FTP transfer finalization failed")
	}

	return nil
}

func (t *clientStorTransfer) SendError(types.TransferErrorCode, string) { t.sendError() }
func (t *clientStorTransfer) sendError() {
	if t.trans != nil {
		analytics.SubOutgoingConnection()

		if err := t.trans.Abort(); err != nil {
			t.pip.Logger.Warning("Failed to abort FTP transfer: %v", err)
		}
	}

	if err := t.client.Close(); err != nil {
		t.pip.Logger.Warning("Failed to close FTP connection: %v", err)
	}
}
