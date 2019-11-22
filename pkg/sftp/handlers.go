package sftp

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
)

type fileWriterFunc func(r *sftp.Request) (io.WriterAt, error)

func (fw fileWriterFunc) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	return fw(r)
}

type fileReaderFunc func(r *sftp.Request) (io.ReaderAt, error)

func (fr fileReaderFunc) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	return fr(r)
}

type fileListerFunc func(r *sftp.Request) (sftp.ListerAt, error)

func (fl fileListerFunc) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	return fl(r)
}

type listerAtFunc func(ls []os.FileInfo, offset int64) (int, error)

func (la listerAtFunc) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	return la(ls, offset)
}

func makeHandlers() sftp.Handlers {
	return sftp.Handlers{
		FileGet:  makeFileReader(),
		FilePut:  makeFileWriter(),
		FileCmd:  nil,
		FileList: makeFileLister(),
	}
}

func makeFileReader() fileReaderFunc {
	return func(r *sftp.Request) (io.ReaderAt, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot get working directory: %w", err)
		}

		file, err := os.Open(filepath.Clean(filepath.Join(dir, r.Filepath)))
		if err != nil {
			return nil, err
		}

		return file, nil
	}
}

func makeFileWriter() fileWriterFunc {
	return func(r *sftp.Request) (io.WriterAt, error) {
		dir, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot get working directory: %w", err)
		}

		file, err := os.Create(filepath.Clean(filepath.Join(dir, r.Filepath)))
		if err != nil {
			return nil, err
		}

		return file, nil
	}
}

func makeFileLister() fileListerFunc {
	listerAt := func(ls []os.FileInfo, offset int64) (int, error) {
		dir, err := os.Getwd()
		if err != nil {
			return 0, fmt.Errorf("cannot get working directory: %w", err)
		}

		infos := []os.FileInfo{}
		err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			infos = append(infos, info)
			return nil
		})
		if err != nil {
			panic(err)
		}

		var n int
		if offset >= int64(len(infos)) {
			return 0, io.EOF
		}
		n = copy(ls, infos[offset:])
		if n < len(ls) {
			return n, io.EOF
		}
		return n, nil
	}

	return func(r *sftp.Request) (sftp.ListerAt, error) {
		return listerAtFunc(listerAt), nil
	}
}
