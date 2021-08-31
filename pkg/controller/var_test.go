package controller

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

func init() {
	pipeline.ClientConstructors[config.TestProtocol] = NewAllSuccess

	_ = log.InitBackend("DEBUG", "stdout", "")
}

type AllSuccess struct{}

func NewAllSuccess(*pipeline.Pipeline) (pipeline.Client, *types.TransferError) {
	return AllSuccess{}, nil
}

func (a AllSuccess) Request() *types.TransferError                 { return nil }
func (a AllSuccess) Data(pipeline.DataStream) *types.TransferError { return nil }
func (a AllSuccess) EndTransfer() *types.TransferError             { return nil }
func (a AllSuccess) SendError(*types.TransferError)                {}
