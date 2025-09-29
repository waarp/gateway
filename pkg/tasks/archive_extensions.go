package tasks

import (
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/ordered"
)

const (
	extensionZip     = ".zip"
	extensionTar     = ".tar"
	extensionTarGz   = ".tar.gz"
	extensionTarBz2  = ".tar.bz2"
	extensionTarXz   = ".tar.xz"
	extensionTarZlib = ".tar.zlib"
	extensionTarZstd = ".tar.zstd"
)

//nolint:gochecknoglobals //global var is needed here for future-proofing
var (
	ArchiveExtensions = ordered.Map[string, func(*archiveTask) error]{}
	ExtractExtensions = ordered.Map[string, func(*extractTask) error]{}
)

//nolint:gochecknoinits //init is needed here to initialize constants
func init() {
	ArchiveExtensions.Add(extensionZip, (*archiveTask).makeZipArchive)
	ArchiveExtensions.Add(extensionTar, makeTarArchiver(noCompressor))
	ArchiveExtensions.Add(extensionTarGz, makeTarArchiver(gzipCompressor))
	ArchiveExtensions.Add(extensionTarBz2, makeTarArchiver(bzip2Compressor))
	ArchiveExtensions.Add(extensionTarXz, makeTarArchiver(xzCompressor))
	ArchiveExtensions.Add(extensionTarZlib, makeTarArchiver(zlibCompressor))
	ArchiveExtensions.Add(extensionTarZstd, makeTarArchiver(zstdCompressor))

	ExtractExtensions.Add(extensionZip, (*extractTask).extractZip)
	ExtractExtensions.Add(extensionTar, makeTarExtractor(noDecompressor))
	ExtractExtensions.Add(extensionTarGz, makeTarExtractor(gzipDecompressor))
	ExtractExtensions.Add(extensionTarBz2, makeTarExtractor(bzip2Decompressor))
	ExtractExtensions.Add(extensionTarXz, makeTarExtractor(xzDecompressor))
	ExtractExtensions.Add(extensionTarZlib, makeTarExtractor(zlibDecompressor))
	ExtractExtensions.Add(extensionTarZstd, makeTarExtractor(zstdDecompressor))
}

func hasExtension(path, extension string) bool {
	return strings.HasSuffix(strings.ToLower(path), extension)
}

func makeTarArchiver(comp compressorFunc) func(*archiveTask) error {
	return func(a *archiveTask) error {
		return a.makeTarArchive(comp)
	}
}

func makeTarExtractor(dec decompressorFunc) func(*extractTask) error {
	return func(e *extractTask) error {
		return e.extractTar(dec)
	}
}
