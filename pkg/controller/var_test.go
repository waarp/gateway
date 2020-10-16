package controller

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
	executor.ClientsConstructors["test"] = NewAllSuccess

	logConf := conf.LogConfig{
		Level: "DEBUG",
		LogTo: "stdout",
	}
	_ = log.InitBackend(logConf)
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

type AllSuccess struct{}

func NewAllSuccess(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AllSuccess{}, nil
}
func (AllSuccess) Connect() *model.PipelineError                   { return nil }
func (AllSuccess) Authenticate() *model.PipelineError              { return nil }
func (AllSuccess) Request() *model.PipelineError                   { return nil }
func (AllSuccess) Data(pipeline.DataStream) *model.PipelineError   { return nil }
func (AllSuccess) Close(*model.PipelineError) *model.PipelineError { return nil }
