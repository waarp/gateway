package r66

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-r66/r66"
	"code.waarp.fr/waarp-r66/r66/utils"
)

type clientAuthHandler struct {
	getFile func() utils.ReadWriterAt
	info    *model.OutTransferInfo
}

func (h *clientAuthHandler) ValidAuth(*r66.Authent) (r66.SessionHandler, error) {
	return &clientSessionHandler{h}, nil
}

type clientSessionHandler struct {
	*clientAuthHandler
}

func (h *clientSessionHandler) ValidRequest(r *r66.Request) (r66.TransferHandler, error) {
	curBlock := uint32(h.info.Transfer.Progress / uint64(r.Block))
	if r.Rank < curBlock {
		curBlock = r.Rank
	}
	r.Rank = curBlock
	if h.info.Transfer.Step <= model.StepData {
		h.info.Transfer.Progress = uint64(curBlock) * uint64(r.Block)
	}

	return &clientTransferHandler{h}, nil
}

type clientTransferHandler struct {
	*clientSessionHandler
}

func (h *clientTransferHandler) GetStream() (utils.ReadWriterAt, error) {
	return h.getFile(), nil
}

func (h *clientTransferHandler) RunPreTask() error                       { return nil }
func (h *clientTransferHandler) ValidEndTransfer(*r66.EndTransfer) error { return nil }
func (h *clientTransferHandler) RunPostTask() error                      { return nil }
func (h *clientTransferHandler) ValidEndRequest() error                  { return nil }
func (h *clientTransferHandler) RunErrorTask(error) error                { return nil }
