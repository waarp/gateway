package tasks

import "strings"

const (
	extensionZip     = ".zip"
	extensionTar     = ".tar"
	extensionTarGz   = ".tar.gz"
	extensionTarBz2  = ".tar.bz2"
	extensionTarXz   = ".tar.xz"
	extensionTarZlib = ".tar.zlib"
	extensionTarZstd = ".tar.zstd"
)

func hasExtension(path, extension string) bool {
	return strings.HasSuffix(strings.ToLower(path), extension)
}
