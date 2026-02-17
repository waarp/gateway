package fs

import (
	"io"
	"io/fs"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

type File interface {
	Name() string
	fs.File
	fs.ReadDirFile
	io.Writer
	io.ReaderAt
	io.WriterAt
	io.Seeker
	Sync() error
}

func Open(path string) (File, error) {
	return OpenFile(path, FlagReadOnly, 0)
}

func Create(path string) (File, error) {
	createPerms := FileMode(conf.GlobalConfig.Paths.FilePerms)

	return OpenFile(path, FlagReadWrite|FlagCreate|FlagTruncate, createPerms)
}

func OpenFile(path string, flags Flags, perm FileMode) (File, error) {
	parsed, srcFs, fsErr := parseFs(path)
	if fsErr != nil {
		return nil, fsErr
	}

	file, openErr := srcFs.OpenFile(parsed.Path, flags, perm)
	if openErr != nil {
		return nil, pathError("open", path, openErr)
	}

	return file, nil
}
