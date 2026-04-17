package ftp

import (
	"net"

	ftplib "github.com/fclairamb/ftpserverlib"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
)

func init() {
	pipelinetest.Register(FTP, pipelinetest.ProtoFeatures{
		Protocol: Module{},
		TransID:  false,
		RuleName: false,
	})

	pipelinetest.Register(FTPS, pipelinetest.ProtoFeatures{
		Protocol: ModuleFTPS{},
		TransID:  false,
		RuleName: false,
	})
}

type testClientContext struct {
	remoteAddr net.Addr
}

func (t testClientContext) RemoteAddr() net.Addr { return t.remoteAddr }

func (testClientContext) Path() string                                  { panic("not implemented") }
func (testClientContext) SetPath(string)                                { panic("not implemented") }
func (testClientContext) SetListPath(string)                            { panic("not implemented") }
func (testClientContext) SetDebug(bool)                                 { panic("not implemented") }
func (testClientContext) Debug() bool                                   { panic("not implemented") }
func (testClientContext) ID() uint32                                    { panic("not implemented") }
func (testClientContext) LocalAddr() net.Addr                           { panic("not implemented") }
func (testClientContext) GetClientVersion() string                      { panic("not implemented") }
func (testClientContext) Close() error                                  { panic("not implemented") }
func (testClientContext) HasTLSForControl() bool                        { panic("not implemented") }
func (testClientContext) HasTLSForTransfers() bool                      { panic("not implemented") }
func (testClientContext) GetLastCommand() string                        { panic("not implemented") }
func (testClientContext) GetLastDataChannel() ftplib.DataChannel        { panic("not implemented") }
func (testClientContext) SetTLSRequirement(ftplib.TLSRequirement) error { panic("not implemented") }
func (testClientContext) SetExtra(extra any)                            { panic("not implemented") }
func (testClientContext) Extra() any                                    { panic("not implemented") }
