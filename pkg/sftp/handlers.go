package sftp

import (
	"io"
	"io/ioutil"
	"os"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
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

type fileCmdFunc func(r *sftp.Request) error

func (fc fileCmdFunc) Filecmd(r *sftp.Request) error {
	return fc(r)
}

type fileListerFunc func(r *sftp.Request) (sftp.ListerAt, error)

func (fl fileListerFunc) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	return fl(r)
}

type listerAtFunc func(ls []os.FileInfo, offset int64) (int, error)

func (la listerAtFunc) ListAt(ls []os.FileInfo, offset int64) (int, error) {
	return la(ls, offset)
}

func makeFileCmder() fileCmdFunc {
	return func(r *sftp.Request) error {
		return sftp.ErrSSHFxOpUnsupported
	}
}

func (l *sshListener) makeFileLister(paths *pipeline.Paths) fileListerFunc {
	return func(r *sftp.Request) (sftp.ListerAt, error) {
		rule, err := getRuleFromPath(l.DB, r, true)
		if err != nil {
			if rule, err = getRuleFromPath(l.DB, r, false); err != nil {
				l.Logger.Errorf("Failed to retrieve rule from request filepath '%s'", r.Filepath)
				return nil, sftp.ErrSSHFxPermissionDenied
			}
		}
		rulePath := rule.OutPath
		if !rule.IsSend {
			rulePath = rule.InPath
		}
		root := utils.GetPath("", utils.Elems{{paths.ServerRoot, false},
			{paths.GatewayHome, false}})

		listerAt := func(ls []os.FileInfo, offset int64) (int, error) {
			dir := utils.DenormalizePath(path.Join(root, rulePath))
			infos, err := ioutil.ReadDir(dir)
			if err != nil {
				return 0, err
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

		statAt := func(ls []os.FileInfo, offset int64) (int, error) {
			filepath := utils.DenormalizePath(path.Join(root, rulePath, path.Base(r.Filepath)))
			fi, err := os.Stat(filepath)
			if err != nil {
				return 0, err
			}
			tmp := []os.FileInfo{fi}
			n := copy(ls, tmp)
			if n < len(ls) {
				return n, io.EOF
			}
			return n, nil
		}

		switch r.Method {
		case "Stat":
			return listerAtFunc(statAt), nil
		default:
			return listerAtFunc(listerAt), nil
		}
	}
}
