package r66

import (
	"crypto/subtle"
	"encoding/base64"
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-r66/r66"
	"code.waarp.fr/waarp-r66/r66/utils"
)

type clientAuthHandler struct {
	getFile func() utils.ReadWriterAt
	info    *model.OutTransferInfo
	config  *config.R66ProtoConfig
	size    uint64
}

func (h *clientAuthHandler) ValidAuth(auth *r66.Authent) (req r66.RequestHandler, err error) {

	var authErr error = &r66.Error{Code: r66.BadAuthent, Detail: "invalid credentials"}
	if subtle.ConstantTimeCompare([]byte(auth.Login), []byte(h.config.ServerLogin)) == 0 {
		err = authErr
	}
	pwd, err := base64.StdEncoding.DecodeString(h.config.ServerPassword)
	if err != nil {
		err = &r66.Error{Code: r66.Internal, Detail: "failed to check credentials"}
	}
	if subtle.ConstantTimeCompare(pwd, auth.Password) == 0 {
		err = authErr
	}

	req = &clientRequestHandler{h}
	return
}

type clientRequestHandler struct {
	*clientAuthHandler
}

func (h *clientRequestHandler) ValidRequest(r *r66.Request) (r66.TransferHandler, error) {
	curBlock := uint32(h.info.Transfer.Progress / uint64(r.Block))
	if r.Rank < curBlock {
		curBlock = r.Rank
	}
	r.Rank = curBlock
	if h.info.Transfer.Step <= types.StepData {
		h.info.Transfer.Progress = uint64(curBlock) * uint64(r.Block)
	}
	if !h.info.Rule.IsSend && r.FileSize > 0 {
		h.size = uint64(r.FileSize)
	}

	return &clientTransferHandler{h}, nil
}

type clientTransferHandler struct {
	*clientRequestHandler
}

func (h *clientTransferHandler) GetStream() (utils.ReadWriterAt, error) {
	return h.getFile(), nil
}

func (h *clientTransferHandler) ValidEndTransfer(end *r66.EndTransfer) error {
	if h.info.Transfer.Step > types.StepData {
		return nil
	}

	if h.info.Rule.IsSend {
		if !h.config.NoFinalHash {
			hash, err := makeHash(h.info.Transfer.TrueFilepath)
			if err != nil {
				return &r66.Error{Code: r66.FinalOp, Detail: "failed to calculate file hash"}
			}
			end.Hash = hash
		}
	} else {
		if h.info.Transfer.Progress != h.size {
			return &r66.Error{
				Code: r66.SizeNotAllowed,
				Detail: fmt.Sprintf("incorrect file size (expected %d, got %d)",
					h.size, h.info.Transfer.Progress),
			}
		}

		if !h.config.NoFinalHash {
			if err := checkHash(h.info.Transfer.TrueFilepath, end.Hash); err != nil {
				return &r66.Error{Code: r66.FinalOp, Detail: err.Error()}
			}
		}
	}

	return nil
}

func (h *clientTransferHandler) RunPreTask() error        { return nil }
func (h *clientTransferHandler) RunPostTask() error       { return nil }
func (h *clientTransferHandler) ValidEndRequest() error   { return nil }
func (h *clientTransferHandler) RunErrorTask(error) error { return nil }
