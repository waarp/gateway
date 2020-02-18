package executor

import (
	"io"

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

func (*TestProtoConfig) ValidServer() error { return nil }
func (*TestProtoConfig) ValidClient() error { return nil }

type AllSuccess struct{}

func NewAllSuccess(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AllSuccess{}, nil
}
func (AllSuccess) Connect() *model.PipelineError                   { return nil }
func (AllSuccess) Authenticate() *model.PipelineError              { return nil }
func (AllSuccess) Request() *model.PipelineError                   { return nil }
func (AllSuccess) Data(io.ReadWriteCloser) *model.PipelineError    { return nil }
func (AllSuccess) Close(*model.PipelineError) *model.PipelineError { return nil }

type ConnectFail struct{}

func NewConnectFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return ConnectFail{}, nil
}
func (ConnectFail) Connect() *model.PipelineError                   { return errConn }
func (ConnectFail) Authenticate() *model.PipelineError              { return nil }
func (ConnectFail) Request() *model.PipelineError                   { return nil }
func (ConnectFail) Data(io.ReadWriteCloser) *model.PipelineError    { return nil }
func (ConnectFail) Close(*model.PipelineError) *model.PipelineError { return nil }

type AuthFail struct{}

func NewAuthFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AuthFail{}, nil
}
func (AuthFail) Connect() *model.PipelineError                   { return nil }
func (AuthFail) Authenticate() *model.PipelineError              { return errAuth }
func (AuthFail) Request() *model.PipelineError                   { return nil }
func (AuthFail) Data(io.ReadWriteCloser) *model.PipelineError    { return nil }
func (AuthFail) Close(*model.PipelineError) *model.PipelineError { return nil }

type RequestFail struct{}

func NewRequestFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return RequestFail{}, nil
}
func (RequestFail) Connect() *model.PipelineError                   { return nil }
func (RequestFail) Authenticate() *model.PipelineError              { return nil }
func (RequestFail) Request() *model.PipelineError                   { return errReq }
func (RequestFail) Data(io.ReadWriteCloser) *model.PipelineError    { return nil }
func (RequestFail) Close(*model.PipelineError) *model.PipelineError { return nil }

type DataFail struct{}

func NewDataFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return DataFail{}, nil
}
func (DataFail) Connect() *model.PipelineError                   { return nil }
func (DataFail) Authenticate() *model.PipelineError              { return nil }
func (DataFail) Request() *model.PipelineError                   { return nil }
func (DataFail) Data(io.ReadWriteCloser) *model.PipelineError    { return errData }
func (DataFail) Close(*model.PipelineError) *model.PipelineError { return nil }

type CloseFail struct{}

func NewCloseFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return CloseFail{}, nil
}
func (CloseFail) Connect() *model.PipelineError                   { return nil }
func (CloseFail) Authenticate() *model.PipelineError              { return nil }
func (CloseFail) Request() *model.PipelineError                   { return nil }
func (CloseFail) Data(io.ReadWriteCloser) *model.PipelineError    { return nil }
func (CloseFail) Close(*model.PipelineError) *model.PipelineError { return errClose }
