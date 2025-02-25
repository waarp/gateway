package tasks

import (
	"compress/bzip2"
	"compress/gzip"
	"compress/zlib"
	"io"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

func noDecompressor(out fs.File) (io.Reader, error) { return out, nil }

//nolint:wrapcheck //wrapping adds nothing here
func gzipDecompressor(out fs.File) (io.Reader, error) {
	return gzip.NewReader(out)
}

//nolint:wrapcheck //wrapping adds nothing here
func bzip2Decompressor(out fs.File) (io.Reader, error) {
	return bzip2.NewReader(out), nil
}

//nolint:wrapcheck //wrapping adds nothing here
func xzDecompressor(out fs.File) (io.Reader, error) {
	return xz.NewReader(out)
}

//nolint:wrapcheck //wrapping adds nothing here
func zstdDecompressor(out fs.File) (io.Reader, error) {
	return zstd.NewReader(out)
}

//nolint:wrapcheck //wrapping adds nothing here
func zlibDecompressor(out fs.File) (io.Reader, error) {
	return zlib.NewReader(out)
}
