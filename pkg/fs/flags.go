package fs

import (
	"fmt"
	"io/fs"
	"os"
)

type FileMode = fs.FileMode

//nolint:gochecknoglobals //global variables are needed here
var (
	FilePerms FileMode = 0o644
	DirPerms  FileMode = 0o755
)

const (
	ModeDir        = fs.ModeDir
	ModeAppend     = fs.ModeAppend
	ModeExclusive  = fs.ModeExclusive
	ModeTemporary  = fs.ModeTemporary
	ModeSymlink    = fs.ModeSymlink
	ModeDevice     = fs.ModeDevice
	ModeNamedPipe  = fs.ModeNamedPipe
	ModeSocket     = fs.ModeSocket
	ModeSetuid     = fs.ModeSetuid
	ModeSetgid     = fs.ModeSetgid
	ModeCharDevice = fs.ModeCharDevice
	ModeSticky     = fs.ModeSticky
	ModeIrregular  = fs.ModeIrregular
	ModeType       = fs.ModeType
	ModePerm       = fs.ModePerm
)

type Flags = int

const (
	FlagReadOnly  = os.O_RDONLY
	FlagWriteOnly = os.O_WRONLY
	FlagReadWrite = os.O_RDWR

	FlagAppend    = os.O_APPEND
	FlagCreate    = os.O_CREATE
	FlagExclusive = os.O_EXCL
	FlagSync      = os.O_SYNC
	FlagTruncate  = os.O_TRUNC
)

func HasFlag(flags, flag Flags) bool {
	return flags&flag != 0
}

// PermsFromString parses a FileMode.String() permission string (e.g. "-rwxr-xr-x")
// and returns the corresponding permission bits. It returns an error if the string
// contains any non-permission characters (i.e. any special mode bits are set).
//
//nolint:mnd,err113 //numbers and errors here are too specific
func PermsFromString(mode string) (FileMode, error) {
	// Accept both "rwxrwxrwx" (9 chars) and "-rwxrwxrwx" (10 chars, as produced
	// by FileMode.String() for permission-only modes). Any other length is invalid.
	switch len(mode) {
	case 10:
		if mode[0] != '-' {
			return 0, fmt.Errorf("invalid permission string %q: contains non-permission bits", mode)
		}

		mode = mode[1:]
	case 9:
		// no leading type character, accepted as-is
	default:
		return 0, fmt.Errorf("invalid permission string %q: expected 9 or 10 characters", mode)
	}

	const rwx = "rwxrwxrwx"
	var result FileMode

	for i := range 9 {
		c := mode[i]
		if c == rwx[i] {
			result |= 1 << uint(8-i)
		} else if c != '-' {
			return 0, fmt.Errorf("invalid permission string %q: invalid character %q at position %d", mode, c, i)
		}
	}

	return result, nil
}
