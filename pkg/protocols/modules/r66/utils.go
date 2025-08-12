package r66

import (
	"fmt"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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

func hashServerPassword(field *string) error {
	if field == nil || *field == "" {
		return nil
	}

	pwd := []byte(*field)

	if _, err := bcrypt.Cost(pwd); err == nil {
		return nil // password already hashed
	}

	pwd = r66.CryptPass(pwd)

	hashed, err := utils.HashPassword(database.BcryptRounds, string(pwd))
	if err != nil {
		return fmt.Errorf("failed to hash server password: %w", err)
	}

	*field = hashed

	return nil
}

func encryptServerPassword(field *string) error {
	if field == nil || *field == "" {
		return nil
	}

	pwd, err := utils.AESCrypt(database.GCM, *field)
	if err != nil {
		return fmt.Errorf("failed to encrypt server password: %w", err)
	}

	*field = pwd

	return nil
}
