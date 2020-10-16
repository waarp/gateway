package executor

import (
	"io/ioutil"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

var (
	errConn  = model.NewPipelineError(model.TeConnection, "connection failed")
	errAuth  = model.NewPipelineError(model.TeBadAuthentication, "authentication failed")
	errReq   = model.NewPipelineError(model.TeForbidden, "request failed")
	errData  = model.NewPipelineError(model.TeDataTransfer, "data failed")
	errClose = model.NewPipelineError(model.TeExternalOperation, "remote post-tasks failed")
)

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

type AllSuccess struct{}

func NewAllSuccess(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AllSuccess{}, nil
}
func (AllSuccess) Connect() *model.PipelineError      { return nil }
func (AllSuccess) Authenticate() *model.PipelineError { return nil }
func (AllSuccess) Request() *model.PipelineError      { return nil }
func (a AllSuccess) Data(f pipeline.DataStream) *model.PipelineError {
	if _, err := ioutil.ReadAll(f); err != nil {
		return model.NewPipelineError(model.TeUnknown, err.Error())
	}
	return nil
}
func (AllSuccess) Close(*model.PipelineError) *model.PipelineError { return nil }

type ConnectFail struct{ AllSuccess }

func NewConnectFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return ConnectFail{}, nil
}
func (ConnectFail) Connect() *model.PipelineError { return errConn }

type AuthFail struct{ AllSuccess }

func NewAuthFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AuthFail{}, nil
}
func (AuthFail) Authenticate() *model.PipelineError { return errAuth }

type RequestFail struct{ AllSuccess }

func NewRequestFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return RequestFail{}, nil
}
func (RequestFail) Request() *model.PipelineError { return errReq }

type DataFail struct{ AllSuccess }

func NewDataFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return DataFail{}, nil
}
func (DataFail) Data(pipeline.DataStream) *model.PipelineError { return errData }

type CloseFail struct{ AllSuccess }

func NewCloseFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return CloseFail{}, nil
}
func (CloseFail) Close(*model.PipelineError) *model.PipelineError { return errClose }
