//nolint:wrapcheck //these are just shortcuts, wrapping errors wouldn't add anything
package fs

import (
	"github.com/hack-pad/hackpadfs"
)

func ValidPath(path string) bool { return hackpadfs.ValidPath(path) }

func SeekFile(file File, offset int64, whence int) (int64, error) {
	return hackpadfs.SeekFile(file, offset, whence)
}

func WriteFile(file File, p []byte) (int, error) {
	return hackpadfs.WriteFile(file, p)
}

func ReadAtFile(file File, p []byte, off int64) (int, error) {
	return hackpadfs.ReadAtFile(file, p, off)
}

func WriteAtFile(file File, p []byte, off int64) (int, error) {
	return hackpadfs.WriteAtFile(file, p, off)
}
