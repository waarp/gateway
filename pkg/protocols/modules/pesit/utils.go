package pesit

import (
	"fmt"
	"math"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
)

func trimRequestPath(path string, rule *model.Rule) string {
	return strings.TrimLeft(strings.TrimPrefix(path, rule.Path), "/")
}

func getPassword(creds []*model.Credential) string {
	for _, cred := range creds {
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

func generateDestFilename(remoteTransferID string, partner *model.LocalAccount,
	rule *model.Rule,
) string {
	return fmt.Sprintf("%s_%s_%s",
		rule.Name,
		partner.Login,
		remoteTransferID)
}
