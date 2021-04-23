package controller

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	pipeline.ClientConstructors["test"] = NewAllSuccess

	_ = log.InitBackend("DEBUG", "stdout", "")
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }
func (*TestProtoConfig) CertRequired() bool  { return false }

type AllSuccess struct{ isSend bool }

func NewAllSuccess(_ *log.Logger, ctx *model.TransferContext) (pipeline.Client, error) {
	return AllSuccess{ctx.Rule.IsSend}, nil
}

func (a AllSuccess) Request() error                 { return nil }
func (a AllSuccess) Data(pipeline.DataStream) error { return nil }
func (a AllSuccess) EndTransfer() error             { return nil }
func (a AllSuccess) SendError(error)                {}
