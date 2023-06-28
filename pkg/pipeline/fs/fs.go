// Package fs is the package used for managing transfer files in a file system
// agnostic way.
package fs

import (
	"errors"
	"fmt"

	"github.com/hack-pad/hackpadfs"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var ErrUnknownFileSystem = errors.New("unknown file system")

// FileSystems contains a list of all the known file systems. The map associates
// the file system's scheme with a function to instantiate the file system.
//
//nolint:gochecknoglobals //a global var is required here
var FileSystems = map[string]FsMaker{
	"file": NewLocalFS,
}

func getFileSystem(url *types.URL) (FS, error) {
	mkfs := FileSystems[url.Scheme]
	if mkfs == nil {
		return nil, fmt.Errorf("%w %q", ErrUnknownFileSystem, url.Scheme)
	}

	return mkfs(url)
}

func DoesFileSystemExist(scheme string) bool {
	return FileSystems[scheme] != nil
}

type FsMaker func(path *types.URL) (FS, error)

type (
	FS         = hackpadfs.FS
	OpenFileFS = hackpadfs.OpenFileFS
	CreateFS   = hackpadfs.CreateFS
	MkdirFS    = hackpadfs.MkdirFS
	MkdirAllFS = hackpadfs.MkdirAllFS
)

type (
	File           = hackpadfs.File
	ReadWriterFile = hackpadfs.ReadWriterFile
	ReaderAtFile   = hackpadfs.ReaderAtFile
	WriterAtFile   = hackpadfs.WriterAtFile
	SeekerFile     = hackpadfs.SeekerFile
	DirReaderFile  = hackpadfs.DirReaderFile
)

type (
	DirEntry = hackpadfs.DirEntry
	FileInfo = hackpadfs.FileInfo
	FileMode = hackpadfs.FileMode
)

// Mode values are bit-wise OR'd with a file's permissions to form the FileMode.
const (
	ModeDir        = hackpadfs.ModeDir
	ModeAppend     = hackpadfs.ModeAppend
	ModeExclusive  = hackpadfs.ModeExclusive
	ModeTemporary  = hackpadfs.ModeTemporary
	ModeSymlink    = hackpadfs.ModeSymlink
	ModeDevice     = hackpadfs.ModeDevice
	ModeNamedPipe  = hackpadfs.ModeNamedPipe
	ModeSocket     = hackpadfs.ModeSocket
	ModeSetuid     = hackpadfs.ModeSetuid
	ModeSetgid     = hackpadfs.ModeSetgid
	ModeCharDevice = hackpadfs.ModeCharDevice
	ModeSticky     = hackpadfs.ModeSticky
	ModeIrregular  = hackpadfs.ModeIrregular

	ModeType = hackpadfs.ModeType
	ModePerm = hackpadfs.ModePerm
)

func IsOnSameFS(path1, path2 *types.URL) bool {
	return path1.Scheme == path2.Scheme && path1.Host == path2.Host
}
