package filewatcher

import (
	"io/fs"
	"path"
	"strings"
	"testing/fstest"
	"time"
)

var memFs = fstest.MapFS{
	"file1.txt": &fstest.MapFile{
		Data:    make([]byte, 111),
		Mode:    0o640,
		ModTime: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	"file2.txt": &fstest.MapFile{
		Data:    make([]byte, 222),
		Mode:    0o644,
		ModTime: time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	"dirA": &fstest.MapFile{
		Mode:    fs.ModeDir | 0o700,
		ModTime: time.Date(2022, 2, 2, 0, 0, 0, 0, time.UTC),
	},
	"dirA/fileA1.txt": &fstest.MapFile{
		Data:    make([]byte, 333),
		Mode:    0o600,
		ModTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	"dirA/fileA2.txt": &fstest.MapFile{
		Data:    make([]byte, 444),
		Mode:    0o600,
		ModTime: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	},
	"dirB": &fstest.MapFile{
		Mode:    fs.ModeDir | 0o666,
		ModTime: time.Date(2025, 2, 2, 0, 0, 0, 0, time.UTC),
	},
}

func toFileInfo(entries []fs.DirEntry) ([]fs.FileInfo, error) {
	infos := make([]fs.FileInfo, len(entries))
	var err error
	for i, entry := range entries {
		if infos[i], err = entry.Info(); err != nil {
			return nil, err
		}
	}

	return infos, nil
}

func readDirInfoFile(file fs.File, count int) ([]fs.FileInfo, error) {
	readDirFile, ok := file.(fs.ReadDirFile)
	if !ok {
		return nil, errNotImplemented
	}

	entries, err := readDirFile.ReadDir(count)
	if err != nil {
		return nil, err
	}

	return toFileInfo(entries)
}

func validPath(name string) string {
	name = strings.TrimPrefix(name, "/")
	name = path.Clean(name)
	if !fs.ValidPath(name) {
		panic("invalid path")
	}

	return name
}
