package sftp

import (
	"net"

	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func init() {
	pipelinetest.Register(SFTP, pipelinetest.ProtoFeatures{
		Protocol: Module{},
		TransID:  false,
		RuleName: false,
	})
}

const (
	RSAPk  = testhelpers.RSAPk
	SSHPbk = testhelpers.SSHPbk
)

type testConnMetadata struct {
	user       string
	remoteAddr net.Addr
}

func (t testConnMetadata) User() string         { return t.user }
func (t testConnMetadata) RemoteAddr() net.Addr { return t.remoteAddr }

func (testConnMetadata) SessionID() []byte     { panic("not implemented") }
func (testConnMetadata) ClientVersion() []byte { panic("not implemented") }
func (testConnMetadata) ServerVersion() []byte { panic("not implemented") }
func (testConnMetadata) LocalAddr() net.Addr   { panic("not implemented") }
