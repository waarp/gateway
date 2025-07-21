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

//nolint:gochecknoglobals //global var is needed here for future-proofing
var (
	ArchiveExtensions = map[string]func(*archiveTask) error{
		extensionZip:     (*archiveTask).makeZipArchive,
		extensionTar:     makeTarArchiver(noCompressor),
		extensionTarGz:   makeTarArchiver(gzipCompressor),
		extensionTarBz2:  makeTarArchiver(bzip2Compressor),
		extensionTarXz:   makeTarArchiver(xzCompressor),
		extensionTarZlib: makeTarArchiver(zlibCompressor),
		extensionTarZstd: makeTarArchiver(zstdCompressor),
	}
	ExtractExtensions = map[string]func(*extractTask) error{
		extensionZip:     (*extractTask).extractZip,
		extensionTar:     makeTarExtractor(noDecompressor),
		extensionTarGz:   makeTarExtractor(gzipDecompressor),
		extensionTarBz2:  makeTarExtractor(bzip2Decompressor),
		extensionTarXz:   makeTarExtractor(xzDecompressor),
		extensionTarZlib: makeTarExtractor(zlibDecompressor),
		extensionTarZstd: makeTarExtractor(zstdDecompressor),
	}
)

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
