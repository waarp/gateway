package internal

import (
	"io"
	"os"

	"github.com/pkg/sftp"
)

// WriterAtFunc is an function implementing sftp.FileWriter.
type WriterAtFunc func(r *sftp.Request) (io.WriterAt, error)

// Filewrite returns a new io.WriterAt pointing to the requested file.
func (wf WriterAtFunc) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	return wf(r)
}

// ReaderAtFunc is an function implementing sftp.FileReader.
type ReaderAtFunc func(r *sftp.Request) (io.ReaderAt, error)

// Fileread returns a new io.ReaderAt pointing to the requested file.
func (rf ReaderAtFunc) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	return rf(r)
}

// CmdFunc is an function implementing sftp.FileCmder.
type CmdFunc func(r *sftp.Request) error

// Filecmd executes the requested command and returns any encountered error.
func (cf CmdFunc) Filecmd(r *sftp.Request) error {
	return cf(r)
}

// FileListerAtFunc is an function implementing sftp.FileLister.
type FileListerAtFunc func(r *sftp.Request) (sftp.ListerAt, error)

// Filelist returns a handler for listing the files under the requested path.
func (flf FileListerAtFunc) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	return flf(r)
}

// ListerAtFunc is an function implementing sftp.ListerAt.
type ListerAtFunc func(ls []os.FileInfo, offset int64) (int, error)

// ListAt fills the given slice with the info of the files under the current
// directory, starting at the given offset.
func (lf ListerAtFunc) ListAt(fi []os.FileInfo, offset int64) (int, error) {
	return lf(fi, offset)
}
