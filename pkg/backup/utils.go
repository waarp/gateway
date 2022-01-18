package backup

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func cryptR66Pswd(credType, val, proto string) string {
	if credType == auth.PasswordHash && proto == "r66" && !utils.IsPasswordHashed(val) {
		return r66.CryptPass(val)
	}

	return val
}
