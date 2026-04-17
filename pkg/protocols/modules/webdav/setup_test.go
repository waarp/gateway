package webdav_test

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/webdav"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestMain(m *testing.M) {
	gwtesting.Register(webdav.Webdav, gwtesting.ProtoFeatures{
		Protocol: webdav.Module{},
		TransID:  false,
		RuleName: false,
		Size:     false,
	})
	gwtesting.Register(webdav.WebdavTLS, gwtesting.ProtoFeatures{
		Protocol: webdav.ModuleTLS{},
		TransID:  false,
		RuleName: false,
		Size:     false,
	})

	m.Run()
}
