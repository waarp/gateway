package fs

import "github.com/hack-pad/hackpadfs"

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

// Flags are bit-wise OR'd with each other in fs.OpenFile().
// Exactly one of Read/Write flags must be specified, and any other flags can be OR'd together.
const (
	FlagROnly = hackpadfs.FlagReadOnly  // open the file read-only.
	FlagWOnly = hackpadfs.FlagWriteOnly // open the file write-only.
	FlagRW    = hackpadfs.FlagReadWrite // open the file read-write.

	FlagAppend    = hackpadfs.FlagAppend    // append data to the file when writing.
	FlagCreate    = hackpadfs.FlagCreate    // create a new file if none exists.
	FlagExclusive = hackpadfs.FlagExclusive // used with Create, file must not exist.
	FlagSync      = hackpadfs.FlagSync      // open for synchronous I/O.
	FlagTruncate  = hackpadfs.FlagTruncate  // truncate regular writable file when opened.
)
