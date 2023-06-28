// Package flags provides the flags which can be used when opening a file using
// the fs package.
package flags

import (
	"github.com/hack-pad/hackpadfs"
)

// Flags are bit-wise OR'd with each other in fs.OpenFile().
// Exactly one of Read/Write flags must be specified, and any other flags can be OR'd together.
const (
	ReadOnly  = hackpadfs.FlagReadOnly  // open the file read-only.
	WriteOnly = hackpadfs.FlagWriteOnly // open the file write-only.
	ReadWrite = hackpadfs.FlagReadWrite // open the file read-write.

	Append    = hackpadfs.FlagAppend    // append data to the file when writing.
	Create    = hackpadfs.FlagCreate    // create a new file if none exists.
	Exclusive = hackpadfs.FlagExclusive // used with Create, file must not exist.
	Sync      = hackpadfs.FlagSync      // open for synchronous I/O.
	Truncate  = hackpadfs.FlagTruncate  // truncate regular writable file when opened.
)
