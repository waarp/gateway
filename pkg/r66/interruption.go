package r66

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/r66/internal"
	"code.waarp.fr/waarp-r66/r66"
)

type interruptionHandler struct {
	c chan *r66.Error
}

func (i *interruptionHandler) SendError(err *types.TransferError) {
	i.c <- internal.ToR66Error(err)
}

func (i *interruptionHandler) Pause() *types.TransferError {
	i.c <- &r66.Error{Code: r66.StoppedTransfer, Detail: "transfer paused by user"}
	return nil
}

func (i *interruptionHandler) Cancel() *types.TransferError {
	i.c <- &r66.Error{Code: r66.CanceledTransfer, Detail: "transfer cancelled by user"}
	return nil
}
