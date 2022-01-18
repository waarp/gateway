package rest

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
)

func checkR66Password(cred *model.Credential, protocol string) {
	if protocol == r66.R66 || protocol == r66.R66TLS {
		cred.Value = r66.CryptPass(cred.Value)
	}
}
