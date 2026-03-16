package r66

import (
	"fmt"
	"path"
	"strings"

	"code.waarp.fr/lib/r66"
	"golang.org/x/crypto/bcrypt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func fileMode(file fs.FileInfo) string {
	fileType := "File"
	if file.IsDir() {
		fileType = "Directory"
	}

	return fileType
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

func trimRequestPath(filename string) string {
	filename = strings.ReplaceAll(filename, `\`, `/`)

	if path.IsAbs(filename) || !fs.IsLocalPath(filename) || fs.IsAbsPath(filename) {
		filename = path.Base(filename)
	}

	return filename
}
