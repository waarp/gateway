package sftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
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

func (l *sshListener) makeFileLister(paths *pipeline.Paths, accountID uint64) fileListerFunc {
	return func(r *sftp.Request) (sftp.ListerAt, error) {
		switch r.Method {
		case "Stat":
			return l.statAt(r, paths, accountID), nil
		case "List":
			return l.listAt(r, paths, accountID), nil
		default:
			return nil, sftp.ErrSSHFxOpUnsupported
		}
	}
}

func (l *sshListener) listAt(r *sftp.Request, paths *pipeline.Paths, accountID uint64) listerAtFunc {
	return func(ls []os.FileInfo, offset int64) (int, error) {
		var infos []os.FileInfo

		l.Logger.Debugf("Received 'List' request on %s", r.Filepath)
		if r.Filepath == "" || r.Filepath == "/" {
			paths, err := l.getRulesPaths(accountID)
			if err != nil {
				return 0, fmt.Errorf("cannot list rule directories")
			}
			for i := range paths {
				infos = append(infos, dirInfo(paths[i]))
			}
		} else {
			rule, err := l.getListRule(r.Filepath, accountID)
			if err != nil {
				return 0, err
			}
			if rule != nil {
				dir := utils.GetPath("", utils.Elems{{rule.OutPath, true},
					{paths.ServerOut, true}, {paths.ServerRoot, false},
					{paths.OutDirectory, true}, {paths.GatewayHome, false}})

				infos, err = ioutil.ReadDir(utils.DenormalizePath(dir))
				if err != nil {
					l.Logger.Errorf("Failed to list directory: %s", err)
					return 0, fmt.Errorf("failed to list directory")
				}
			}
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
}

func (l *sshListener) getListRule(rulePath string, accountID uint64) (*model.Rule, error) {
	sndRule, err := l.getRule(accountID, rulePath, true)
	if err != nil {
		return nil, err
	}

	rcvRule, err := l.getRule(accountID, rulePath, false)
	if err != nil {
		return nil, err
	}

	if sndRule == nil && rcvRule == nil {
		l.Logger.Infof("No rule found with path '%s'", rulePath)
		return nil, sftp.ErrSSHFxNoSuchFile
	}
	return sndRule, nil
}

func (l *sshListener) statAt(r *sftp.Request, paths *pipeline.Paths, accountID uint64) listerAtFunc {
	return func(ls []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debugf("Received 'Stat' request on %s", r.Filepath)

		rule, err := l.getRule(accountID, path.Dir(r.Filepath), true)
		if err != nil || rule == nil {
			return 0, fmt.Errorf("failed to retrieve rule for path '%s'", r.Filepath)
		}

		file := utils.GetPath(path.Base(r.Filepath), utils.Elems{{rule.OutPath, true},
			{paths.ServerOut, true}, {paths.ServerRoot, false},
			{paths.OutDirectory, true}, {paths.GatewayHome, false}})

		fi, err := os.Stat(utils.DenormalizePath(file))
		if err != nil {
			if os.IsNotExist(err) {
				return 0, sftp.ErrSSHFxNoSuchFile
			}
			l.Logger.Errorf("Failed to get file stats: %s", r.Filepath)
			return 0, fmt.Errorf("failed to get file stats")
		}
		tmp := []os.FileInfo{fi}
		n := copy(ls, tmp)
		if n < len(ls) {
			return n, io.EOF
		}
		return n, nil
	}
}

func (l *sshListener) getRule(accountID uint64, rulePath string, isSend bool) (*model.Rule, error) {
	var rule model.Rule
	if err := l.DB.Get(&rule, "path=? AND send=?", rulePath, isSend).Run(); err != nil {
		if database.IsNotFound(err) {
			direction := "receive"
			if isSend {
				direction = "sending"
			}
			l.Logger.Debugf("No %s rule found for path '%s'", direction, rulePath)
			return nil, nil
		}
		l.Logger.Errorf("Failed to retrieve rule: %s", err)
		return nil, fmt.Errorf("failed to retrieve rule: %s", err)
	}

	var accesses model.RuleAccesses
	if err := l.DB.Select(&accesses).Where("rule_id=?", rule.ID).Run(); err != nil {
		l.Logger.Errorf("Failed to retrieve rule permissions: %s", err)
		return nil, fmt.Errorf("failed to retrieve rule permissions")
	}

	if len(accesses) == 0 {
		return &rule, nil
	}

	for _, access := range accesses {
		if (access.ObjectType == model.TableLocAgents && access.ObjectID == l.Agent.ID) ||
			(access.ObjectType == model.TableLocAccounts && access.ObjectID == accountID) {
			return &rule, nil
		}
	}
	return nil, fmt.Errorf("user is not allowed to use the specified rule")
}

func (l *sshListener) getRulesPaths(accountID uint64) ([]string, error) {
	var rules model.Rules
	query := l.DB.Select(&rules).Distinct("path").Where(
		`(id IN 
			(SELECT DISTINCT rule_id FROM `+model.TableRuleAccesses+` WHERE
				(object_id=? AND object_type=?) OR
				(object_id=? AND object_type=?)
			)
		)
		OR 
		( (SELECT COUNT(*) FROM `+model.TableRuleAccesses+` WHERE rule_id = id) = 0 )`,
		accountID, model.TableLocAccounts, l.Agent.ID, model.TableLocAgents).
		OrderBy("path", true)
	if err := query.Run(); err != nil {
		l.Logger.Errorf("Failed to retrieve rule list: %s", err)
		return nil, err
	}

	paths := make([]string, len(rules))
	for i := range rules {
		paths[i] = rules[i].Path
	}
	return paths, nil
}

type dirInfo string

func (f dirInfo) Name() string {
	return string(f)
}

func (f dirInfo) Size() int64 {
	return 0
}

func (f dirInfo) Mode() os.FileMode {
	return os.ModeDir | 0o700
}

func (f dirInfo) ModTime() time.Time {
	return time.Now()
}

func (f dirInfo) IsDir() bool {
	return true
}

func (f dirInfo) Sys() interface{} {
	return nil
}
