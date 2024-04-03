package r66

import (
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/compatibility"
)

func fileMode(file fs.FileInfo) string {
	fileType := "File"
	if file.IsDir() {
		fileType = "Directory"
	}

	return fileType
}

// CryptPass returns the R66 hash of the given password.
func CryptPass(pwd string) string {
	return string(r66.CryptPass([]byte(pwd)))
}

func usesLegacyCert(db database.ReadAccess, owner authentication.Owner) bool {
	if compatibility.IsLegacyR66CertificateAllowed {
		if n, err := db.Count(&model.Credential{}).Where(owner.GetCredCond()).
			Where("type=?", AuthLegacyCertificate).Run(); err == nil {
			return n != 0
		}
	}

	return false
}
