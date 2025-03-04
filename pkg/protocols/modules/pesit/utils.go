package pesit

import (
	"math"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

func trimRequestPath(path string, rule *model.Rule) string {
	return strings.TrimLeft(strings.TrimPrefix(path, rule.Path), "/")
}

func getPassword(transCtx *model.TransferContext) string {
	for _, cred := range transCtx.RemoteAccountCreds {
		if cred.Type == auth.Password {
			return cred.Value
		}
	}

	return ""
}

func makeReservationSpaceKB(info fs.FileInfo) uint32 {
	sizeKB := float64(info.Size()) / float64(bytesPerKB)
	floor := math.Floor(sizeKB)

	return uint32(floor) + 1
}
