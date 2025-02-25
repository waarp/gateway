package fs

import (
	"io/fs"
	"os"

	rfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/vfs"
)

var _ FS = &VFS{}

type VFS struct{ *vfs.VFS }

func (v *VFS) Open(name string) (fs.File, error) {
	return v.OpenFile(name, FlagReadOnly, 0)
}

func (v *VFS) OpenFile(name string, flags Flags, perm FileMode) (File, error) {
	file, err := v.VFS.OpenFile(name, flags, perm)

	//nolint:wrapcheck //no need to wrap here
	return vFile{file}, err
}

func (v *VFS) Stat(name string) (FileInfo, error) {
	//nolint:wrapcheck //no need to wrap here
	return v.VFS.Stat(name)
}

func (v *VFS) ReadDir(name string) ([]DirEntry, error) {
	infos, err := v.VFS.ReadDir(name)

	//nolint:wrapcheck //no need to wrap here
	return asDirEntries(infos), err
}

func (v *VFS) fs() rfs.Fs { return v.Fs() }

type vFile struct {
	vfs.Handle
}

func (v vFile) ReadDir(n int) ([]DirEntry, error) {
	infos, err := v.Handle.Readdir(n)

	//nolint:wrapcheck //no need to wrap here
	return asDirEntries(infos), err
}

func (v vFile) Sync() error {
	//nolint:wrapcheck //no need to wrap here
	return v.Flush()
}

type vDirEntry struct {
	os.FileInfo
}

func (v vDirEntry) Type() FileMode          { return v.Mode() }
func (v vDirEntry) Info() (FileInfo, error) { return v, nil }

func asDirEntries(infos []os.FileInfo) []DirEntry {
	entries := make([]DirEntry, len(infos))
	for i, info := range infos {
		entries[i] = vDirEntry{info}
	}

	return entries
}
