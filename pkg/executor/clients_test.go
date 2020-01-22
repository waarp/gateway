package executor

import (
	"fmt"
	"io"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
)

func init() {
	config.ProtoConfigs["test"] = func() config.ProtoConfig { return new(TestProtoConfig) }
}

type TestProtoConfig struct{}

func (*TestProtoConfig) ValidServer() error { return nil }
func (*TestProtoConfig) ValidClient() error { return nil }

type AllSuccess struct{}

func NewAllSuccess(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AllSuccess{}, nil
}
func (AllSuccess) Connect() error                { return nil }
func (AllSuccess) Authenticate() error           { return nil }
func (AllSuccess) Request() error                { return nil }
func (AllSuccess) Data(io.ReadWriteCloser) error { return nil }

type ConnectFail struct{}

func NewConnectFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return ConnectFail{}, nil
}
func (ConnectFail) Connect() error                { return fmt.Errorf("failed") }
func (ConnectFail) Authenticate() error           { return nil }
func (ConnectFail) Request() error                { return nil }
func (ConnectFail) Data(io.ReadWriteCloser) error { return nil }

type AuthFail struct{}

func NewAuthFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return AuthFail{}, nil
}
func (AuthFail) Connect() error                { return nil }
func (AuthFail) Authenticate() error           { return fmt.Errorf("failed") }
func (AuthFail) Request() error                { return nil }
func (AuthFail) Data(io.ReadWriteCloser) error { return nil }

type RequestFail struct{}

func NewRequestFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return RequestFail{}, nil
}
func (RequestFail) Connect() error                { return nil }
func (RequestFail) Authenticate() error           { return nil }
func (RequestFail) Request() error                { return fmt.Errorf("failed") }
func (RequestFail) Data(io.ReadWriteCloser) error { return nil }

type DataFail struct{}

func NewDataFail(_ model.OutTransferInfo, _ <-chan model.Signal) (pipeline.Client, error) {
	return DataFail{}, nil
}
func (DataFail) Connect() error                { return nil }
func (DataFail) Authenticate() error           { return nil }
func (DataFail) Request() error                { return nil }
func (DataFail) Data(io.ReadWriteCloser) error { return fmt.Errorf("failed") }
