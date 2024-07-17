package sftp

import (
	"net"
	"path"

	"github.com/smartystreets/goconvey/convey"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline/pipelinetest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func init() {
	pipelinetest.Protocols[SFTP] = pipelinetest.ProtoFeatures{
		MakeClient:        Module{}.NewClient,
		MakeServer:        Module{}.NewServer,
		MakeServerConfig:  Module{}.MakeServerConfig,
		MakePartnerConfig: Module{}.MakePartnerConfig,
		MakeClientConfig:  Module{}.MakeClientConfig,
		TransID:           false,
		RuleName:          false,
	}
}

func mkURL(elem ...string) *types.URL {
	full := path.Join(elem...)

	url, err := types.ParseURL(full)
	convey.So(err, convey.ShouldBeNil)

	return url
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
