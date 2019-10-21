package sftp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
)

var errShutdown = fmt.Errorf("server is shutting down")

func checkShutdown(db *database.Db, trans model.Transfer, shutdown <-chan bool) error {
	select {
	case <-shutdown:
		trans.Status = model.StatusError

		hist, err := trans.ToHistory(db, time.Now().UTC())
		if err != nil {
			return err
		}

		ses, err := db.BeginTransaction()
		if err != nil {
			return err
		}
		if err := ses.Create(hist); err != nil {
			ses.Rollback()
			return err
		}
		if err := ses.Delete(trans); err != nil {
			ses.Rollback()
			return err
		}

		if err := ses.Commit(); err != nil {
			return err
		}
		return errShutdown
	default:
		return nil
	}
}

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

func makeHandlers(db *database.Db, agent *model.LocalAgent, account *model.LocalAccount, shutdown <-chan bool) sftp.Handlers {
	root, _ := os.Getwd()
	var conf map[string]interface{}
	if err := json.Unmarshal(agent.ProtoConfig, &conf); err == nil {
		root, _ = conf["root"].(string)
	}
	return sftp.Handlers{
		FileGet:  makeFileReader(db, agent.ID, account.ID, root, shutdown),
		FilePut:  makeFileWriter(db, agent.ID, account.ID, root, shutdown),
		FileCmd:  nil,
		FileList: makeFileLister(root),
	}
}

func makeFileReader(db *database.Db, agentID, accountID uint64, root string, shutdown <-chan bool) fileReaderFunc {
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
		trans := model.Transfer{
			RuleID:      rule.ID,
			IsServer:    true,
			RemoteID:    agentID,
			AccountID:   accountID,
			Source:      r.Filepath,
			Destination: filepath.Base(r.Filepath),
			Start:       time.Now(),
			Status:      model.StatusTransfer,
		}
		if err := db.Create(&trans); err != nil {
			return nil, err
		}

		if err := checkShutdown(db, trans, shutdown); err != nil {
			return nil, err
		}

		// Open requested file
		file, err := os.Open(filepath.FromSlash(fmt.Sprintf("%s/%s", root, r.Filepath)))
		if err != nil {
			return nil, err
		}
		return file, nil
	}
}

func makeFileWriter(db *database.Db, agentID, accountID uint64, root string, shutdown <-chan bool) fileWriterFunc {
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
				os.MkdirAll(dir, 1744)
			} else {
				return nil, err
			}
		} else if !info.IsDir() {
			os.MkdirAll(dir, 1744)
		}

		// Create Transfer
		trans := model.Transfer{
			RuleID:      rule.ID,
			IsServer:    true,
			RemoteID:    agentID,
			AccountID:   accountID,
			Source:      filepath.Base(r.Filepath),
			Destination: r.Filepath,
			Start:       time.Now(),
			Status:      model.StatusTransfer,
		}
		if err := db.Create(&trans); err != nil {
			return nil, err
		}

		if err := checkShutdown(db, trans, shutdown); err != nil {
			return nil, err
		}

		// Create file
		file, err := os.Create(filepath.FromSlash(fmt.Sprintf("%s/%s", root, r.Filepath)))
		if err != nil {
			return nil, err
		}

		return file, nil
	}
}

func makeFileLister(root string) fileListerFunc {
	listerAt := func(ls []os.FileInfo, offset int64) (int, error) {
		infos := []os.FileInfo{}
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
