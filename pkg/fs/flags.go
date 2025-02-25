package fs

import (
	"io/fs"
	"os"
)

type FileMode = fs.FileMode

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
