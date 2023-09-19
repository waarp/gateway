package r66

import (
	"code.waarp.fr/lib/r66"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
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
