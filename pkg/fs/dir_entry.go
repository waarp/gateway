package fs

import (
	"io/fs"

	"golang.org/x/exp/slices"
)

type GenericDirEntry struct {
	*GenericFileInfo
}

func (g *GenericDirEntry) Type() FileMode          { return g.FileMode.Type() }
func (g *GenericDirEntry) Info() (FileInfo, error) { return g.GenericFileInfo, nil }

func SortDirEntries(entries []fs.DirEntry) {
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		switch {
		case a.Name() < b.Name():
			return -1
		case a.Name() > b.Name():
			return 1
		default:
			return 0
		}
	})
}
