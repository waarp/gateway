package internal

import (
	"io"
	"os"

	"github.com/pkg/sftp"
)

type WriterAtFunc func(r *sftp.Request) (io.WriterAt, error)

func (wf WriterAtFunc) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	return wf(r)
}

type ReaderAtFunc func(r *sftp.Request) (io.ReaderAt, error)

func (rf ReaderAtFunc) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	return rf(r)
}

type CmdFunc func(r *sftp.Request) error

func (cf CmdFunc) Filecmd(r *sftp.Request) error {
	return cf(r)
}

type FileListerAtFunc func(r *sftp.Request) (sftp.ListerAt, error)

func (flf FileListerAtFunc) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	return flf(r)
}

type ListerAtFunc func(ls []os.FileInfo, offset int64) (int, error)

func (lf ListerAtFunc) ListAt(fi []os.FileInfo, offset int64) (int, error) {
	return lf(fi, offset)
}
