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
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"github.com/go-xorm/builder"
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

func makeHandlers(db *database.Db, logger *log.Logger, agent *model.LocalAgent,
	account *model.LocalAccount, report chan<- progress) sftp.Handlers {

	root, _ := os.Getwd()
	var conf map[string]interface{}
	if err := json.Unmarshal(agent.ProtoConfig, &conf); err == nil {
		root, _ = conf["root"].(string)
	}
	return sftp.Handlers{
		FileGet:  makeFileReader(db, logger, agent.ID, account.ID, root, report),
		FilePut:  makeFileWriter(db, logger, agent.ID, account.ID, root, report),
		FileCmd:  nil,
		FileList: makeFileLister(root),
	}
}

func runTasks(db *database.Db, logger *log.Logger, chain model.Chain,
	rule *model.Rule, trans *model.Transfer) error {

	var taskChain []*model.Task
	filters := &database.Filters{
		Order:      "rank ASC",
		Conditions: builder.Eq{"rule_id": rule.ID, "chain": chain},
	}
	if err := db.Select(&taskChain, filters); err != nil {
		return err
	}

	taskRunner := tasks.Processor{
		Db:       db,
		Logger:   logger,
		Rule:     rule,
		Transfer: trans,
	}
	if err := taskRunner.RunTasks(taskChain); err != nil {
		return err
	}

	return nil
}

func makeFileReader(db *database.Db, logger *log.Logger, agentID, accountID uint64,
	root string, report chan<- progress) fileReaderFunc {

	return func(r *sftp.Request) (io.ReaderAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := &model.Rule{Path: path, IsSend: true}
		if err := db.Get(rule); err != nil {
			return nil, err
		}

		// Create Transfer
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    agentID,
			AccountID:  accountID,
			SourcePath: r.Filepath,
			DestPath:   filepath.Base(r.Filepath),
			Start:      time.Now(),
			Status:     model.StatusPreTasks,
		}
		if err := db.Create(trans); err != nil {
			return nil, err
		}

		// Open requested file
		file, err := os.Open(filepath.Clean(filepath.Join(root, r.Filepath)))
		if err != nil {
			return nil, err
		}

		stream := &uploadStream{
			File:   file,
			ID:     trans.ID,
			Report: report,
		}

		if err := runTasks(db, logger, model.ChainPre, rule, trans); err != nil {
			return stream, err
		}

		if err := db.Update(&model.Transfer{Status: model.StatusTransfer},
			trans.ID, false); err != nil {
			return stream, err
		}

		return stream, nil
	}
}

func makeDir(root, path string) error {
	// Create dir if it doesn't exist
	dir := filepath.FromSlash(fmt.Sprintf("%s/%s", root, path))
	if info, err := os.Lstat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0740); err != nil {
				return err
			}
		} else {
			return err
		}
	} else if !info.IsDir() {
		if err := os.MkdirAll(dir, 0740); err != nil {
			return err
		}
	}
	return nil
}

func makeFileWriter(db *database.Db, logger *log.Logger, agentID, accountID uint64,
	root string, report chan<- progress) fileWriterFunc {

	return func(r *sftp.Request) (io.WriterAt, error) {
		// Get rule according to request filepath
		path := filepath.Dir(r.Filepath)
		if path == "." || path == "/" {
			return nil, fmt.Errorf("%s cannot be used to find a rule", r.Filepath)
		}
		rule := &model.Rule{Path: path, IsSend: false}
		if err := db.Get(rule); err != nil {
			return nil, err
		}

		if err := makeDir(root, rule.Path); err != nil {
			return nil, err
		}

		// Create Transfer
		trans := &model.Transfer{
			RuleID:     rule.ID,
			IsServer:   true,
			AgentID:    agentID,
			AccountID:  accountID,
			SourcePath: filepath.Base(r.Filepath),
			DestPath:   r.Filepath,
			Start:      time.Now(),
			Status:     model.StatusPreTasks,
		}
		if err := db.Create(trans); err != nil {
			return nil, err
		}

		// Create file
		file, err := os.Create(filepath.Clean(filepath.Join(root, r.Filepath)))
		if err != nil {
			return nil, err
		}

		stream := &downloadStream{
			File:   file,
			ID:     trans.ID,
			Report: report,
		}

		if err := runTasks(db, logger, model.ChainPre, rule, trans); err != nil {
			return stream, err
		}

		if err := db.Update(&model.Transfer{Status: model.StatusTransfer},
			trans.ID, false); err != nil {
			return stream, err
		}

		return stream, nil
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
