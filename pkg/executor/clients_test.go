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
	errConn = model.NewTransferError(model.TeConnection, "connection failed")
	errAuth = model.NewTransferError(model.TeBadAuthentication, "authentication failed")
	errReq  = model.NewTransferError(model.TeForbidden, "request failed")
	errData = model.NewTransferError(model.TeDataTransfer, "data failed")
)

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error { return nil }
func (*TestProtoConfig) ValidClient() error { return nil }

type AllSuccess struct{}

func NewAllSuccess(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AllSuccess{}, nil
}
func (AllSuccess) Connect() model.TransferError                { return model.TransferError{} }
func (AllSuccess) Authenticate() model.TransferError           { return model.TransferError{} }
func (AllSuccess) Request() model.TransferError                { return model.TransferError{} }
func (AllSuccess) Data(io.ReadWriteCloser) model.TransferError { return model.TransferError{} }

type ConnectFail struct{}

func NewConnectFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return ConnectFail{}, nil
}
func (ConnectFail) Connect() model.TransferError                { return errConn }
func (ConnectFail) Authenticate() model.TransferError           { return model.TransferError{} }
func (ConnectFail) Request() model.TransferError                { return model.TransferError{} }
func (ConnectFail) Data(io.ReadWriteCloser) model.TransferError { return model.TransferError{} }

type AuthFail struct{}

func NewAuthFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AuthFail{}, nil
}
func (AuthFail) Connect() model.TransferError                { return model.TransferError{} }
func (AuthFail) Authenticate() model.TransferError           { return errAuth }
func (AuthFail) Request() model.TransferError                { return model.TransferError{} }
func (AuthFail) Data(io.ReadWriteCloser) model.TransferError { return model.TransferError{} }

type RequestFail struct{}

func NewRequestFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return RequestFail{}, nil
}
func (RequestFail) Connect() model.TransferError                { return model.TransferError{} }
func (RequestFail) Authenticate() model.TransferError           { return model.TransferError{} }
func (RequestFail) Request() model.TransferError                { return errReq }
func (RequestFail) Data(io.ReadWriteCloser) model.TransferError { return model.TransferError{} }

type DataFail struct{}

func NewDataFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return DataFail{}, nil
}
func (DataFail) Connect() model.TransferError                { return model.TransferError{} }
func (DataFail) Authenticate() model.TransferError           { return model.TransferError{} }
func (DataFail) Request() model.TransferError                { return model.TransferError{} }
func (DataFail) Data(io.ReadWriteCloser) model.TransferError { return errData }
