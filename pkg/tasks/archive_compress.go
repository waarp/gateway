package tasks

import (
	"archive/zip"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"

	"github.com/dsnet/compress/bzip2"
	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
)

const defaultCompressionLevel = -1

var ErrInvalidCompressionLevel = errors.New("invalid compression level")

func isValidCompressionLevel(level int) bool { return level >= 0 && level <= 9 }

//nolint:wrapcheck //wrapping adds nothing here
func getDeflateCompressor(level int) zip.Compressor {
	return func(out io.Writer) (io.WriteCloser, error) {
		if level == defaultCompressionLevel {
			level = flate.DefaultCompression
		}

		return flate.NewWriter(out, level)
	}
}

func noCompressor(out fs.File, _ int) (io.WriteCloser, error) {
	return &nopCloser{out}, nil
}

//nolint:wrapcheck //wrapping adds nothing here
func gzipCompressor(out fs.File, level int) (io.WriteCloser, error) {
	if level == defaultCompressionLevel {
		level = gzip.DefaultCompression
	}

	return gzip.NewWriterLevel(out, level)
}

//nolint:wrapcheck //wrapping adds nothing here
func bzip2Compressor(out fs.File, level int) (io.WriteCloser, error) {
	conf := &bzip2.WriterConfig{Level: level}
	if level == defaultCompressionLevel {
		conf.Level = bzip2.DefaultCompression
	}

	return bzip2.NewWriter(out, conf)
}

//nolint:wrapcheck //wrapping adds nothing here
func xzCompressor(out fs.File, _ int) (io.WriteCloser, error) {
	return xz.NewWriter(out)
}

//nolint:wrapcheck //wrapping adds nothing here
func zstdCompressor(out fs.File, level int) (io.WriteCloser, error) {
	var eLevel zstd.EncoderLevel

	//nolint:mnd //too specific
	switch level {
	case 0, 1, 2:
		eLevel = zstd.SpeedFastest
	case 3, 4, 5, defaultCompressionLevel:
		eLevel = zstd.SpeedDefault
	case 6, 7, 8:
		eLevel = zstd.SpeedBetterCompression
	case 9:
		eLevel = zstd.SpeedBestCompression
	}

	return zstd.NewWriter(out, zstd.WithEncoderLevel(eLevel))
}

//nolint:wrapcheck //wrapping adds nothing here
func zlibCompressor(out fs.File, level int) (io.WriteCloser, error) {
	if level == defaultCompressionLevel {
		level = zlib.DefaultCompression
	}

	return zlib.NewWriterLevel(out, level)
}

type nopCloser struct{ io.Writer }

func (n *nopCloser) Close() error { return nil }
