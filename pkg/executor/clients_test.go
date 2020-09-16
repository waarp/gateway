package executor

import (
	"io/ioutil"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

var (
	errConn  = types.NewTransferError(types.TeConnection, "connection failed")
	errAuth  = types.NewTransferError(types.TeBadAuthentication, "authentication failed")
	errReq   = types.NewTransferError(types.TeForbidden, "request failed")
	errData  = types.NewTransferError(types.TeDataTransfer, "data failed")
	errClose = types.NewTransferError(types.TeExternalOperation, "remote post-tasks failed")
)

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error  { return nil }
func (*TestProtoConfig) ValidPartner() error { return nil }

type AllSuccess struct{}

func NewAllSuccess(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AllSuccess{}, nil
}
func (AllSuccess) Connect() error      { return nil }
func (AllSuccess) Authenticate() error { return nil }
func (AllSuccess) Request() error      { return nil }
func (a AllSuccess) Data(f pipeline.DataStream) error {
	if _, err := ioutil.ReadAll(f); err != nil {
		return types.NewTransferError(types.TeUnknown, err.Error())
	}
	return nil
}
func (AllSuccess) Close(error) error { return nil }

type ConnectFail struct{ AllSuccess }

func NewConnectFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return ConnectFail{}, nil
}
func (ConnectFail) Connect() error { return errConn }

type AuthFail struct{ AllSuccess }

func NewAuthFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AuthFail{}, nil
}
func (AuthFail) Authenticate() error { return errAuth }

type RequestFail struct{ AllSuccess }

func NewRequestFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return RequestFail{}, nil
}
func (RequestFail) Request() error { return errReq }

type DataFail struct{ AllSuccess }

func NewDataFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return DataFail{}, nil
}
func (DataFail) Data(pipeline.DataStream) error { return errData }

type CloseFail struct{ AllSuccess }

func NewCloseFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return CloseFail{}, nil
}
func (CloseFail) Close(error) error { return errClose }
