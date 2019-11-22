package sftp

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
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

func makeHandlers(db *database.Db, agent *model.LocalAgent, account *model.LocalAccount,
	report chan<- progress) sftp.Handlers {

	root, _ := os.Getwd()
	var conf map[string]interface{}
	if err := json.Unmarshal(agent.ProtoConfig, &conf); err == nil {
		root, _ = conf["root"].(string)
	}
	return sftp.Handlers{
		FileGet:  makeFileReader(db, agent.ID, account.ID, root, report),
		FilePut:  makeFileWriter(db, agent.ID, account.ID, root, report),
		FileCmd:  nil,
		FileList: makeFileLister(root),
	}
}

func makeFileReader(db *database.Db, agentID, accountID uint64, root string,
	report chan<- progress) fileReaderFunc {

	return func(r *sftp.Request) (io.ReaderAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := model.Rule{OutPath: path, IsGet: true}
		if err := db.Get(&rule); err != nil {
			return nil, err
		}

		// Create Transfer
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			RemoteID:   agentID,
			AccountID:  accountID,
			SourcePath: r.Filepath,
			DestPath:   filepath.Base(r.Filepath),
			Start:      time.Now(),
			Status:     model.StatusTransfer,
		}
		if err := db.Create(trans); err != nil {
			return nil, err
		}

		// Open requested file
		file, err := os.Open(filepath.Clean(filepath.Join(root, r.Filepath)))
		if err != nil {
			return nil, err
		}

		stream := uploadStream{
			File:   file,
			ID:     trans.ID,
			Report: report,
		}

		return &stream, nil
	}
}

func makeFileWriter(db *database.Db, agentID, accountID uint64, root string,
	report chan<- progress) fileWriterFunc {

	return func(r *sftp.Request) (io.WriterAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := model.Rule{InPath: path, IsGet: false}
		if err := db.Get(&rule); err != nil {
			return nil, err
		}

		// Create dir if it doesn't exist
		dir := filepath.FromSlash(fmt.Sprintf("%s/%s", root, rule.InPath))
		if info, err := os.Lstat(dir); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(dir, 0740); err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		} else if !info.IsDir() {
			if err := os.MkdirAll(dir, 0740); err != nil {
				return nil, err
			}
		}

		// Create Transfer
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			RemoteID:   agentID,
			AccountID:  accountID,
			SourcePath: filepath.Base(r.Filepath),
			DestPath:   r.Filepath,
			Start:      time.Now(),
			Status:     model.StatusTransfer,
		}
		if err := db.Create(trans); err != nil {
			return nil, err
		}

		// Create file
		file, err := os.Create(filepath.Clean(filepath.Join(root, r.Filepath)))
		if err != nil {
			return nil, err
		}

		stream := downloadStream{
			File:   file,
			ID:     trans.ID,
			Report: report,
		}

		return &stream, nil
	}
}

func makeFileLister(root string) fileListerFunc {
	return func(r *sftp.Request) (sftp.ListerAt, error) {
		listerAt := func(ls []os.FileInfo, offset int64) (int, error) {
			dir := root + r.Filepath
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
			path := root + r.Filepath
			fi, err := os.Stat(path)
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
