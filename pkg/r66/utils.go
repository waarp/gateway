package r66

import "code.waarp.fr/apps/gateway/gateway/pkg/fs"

func fileMode(file fs.FileInfo) string {
	fileType := "File"
	if file.IsDir() {
		fileType = "Directory"
	}

	return fileType
}
