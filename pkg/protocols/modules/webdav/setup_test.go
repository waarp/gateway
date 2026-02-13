package webdav_test

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestMain(m *testing.M) {
	gwtesting.Protocols[webdav.Webdav] = gwtesting.ProtoFeatures{
		MakeClient:        webdav.Module{}.NewClient,
		MakeServer:        webdav.Module{}.NewServer,
		MakeServerConfig:  webdav.Module{}.MakeServerConfig,
		MakeClientConfig:  webdav.Module{}.MakeClientConfig,
		MakePartnerConfig: webdav.Module{}.MakePartnerConfig,
		TransID:           false,
		RuleName:          false,
		Size:              false,
	}
	gwtesting.Protocols[webdav.WebdavTLS] = gwtesting.ProtoFeatures{
		MakeClient:        webdav.ModuleTLS{}.NewClient,
		MakeServer:        webdav.ModuleTLS{}.NewServer,
		MakeServerConfig:  webdav.ModuleTLS{}.MakeServerConfig,
		MakeClientConfig:  webdav.ModuleTLS{}.MakeClientConfig,
		MakePartnerConfig: webdav.ModuleTLS{}.MakePartnerConfig,
		TransID:           false,
		RuleName:          false,
		Size:              false,
	}

	m.Run()
}
