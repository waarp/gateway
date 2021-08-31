package sftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp/internal"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
	"github.com/pkg/sftp"
)

var (
	leaf   = func(str string) utils.Leaf { return utils.Leaf(str) }
	branch = func(str string) utils.Branch { return utils.Branch(str) }
)

func makeFileCmder() internal.CmdFunc {
	return func(r *sftp.Request) error {
		return sftp.ErrSSHFxOpUnsupported
	}
}

func (l *sshListener) makeFileLister(acc *model.LocalAccount) internal.FileListerAtFunc {
	return func(r *sftp.Request) (sftp.ListerAt, error) {
		switch r.Method {
		case "Stat":
			return l.statAt(r, acc), nil
		case "List":
			return l.listAt(r, acc), nil
		default:
			return nil, sftp.ErrSSHFxOpUnsupported
		}
	}
}

func (l *sshListener) listAt(r *sftp.Request, acc *model.LocalAccount) internal.ListerAtFunc {
	return func(ls []os.FileInfo, offset int64) (int, error) {
		var infos []os.FileInfo
		l.Logger.Debugf("Received 'List' request on %s", r.Filepath)

		realDir, err := l.getRealPath(acc, r.Filepath)
		if err != nil {
			return 0, internal.ToSFTPErr(err)
		}

		if realDir != "" {
			infos, err = ioutil.ReadDir(realDir)
			if err != nil {
				if os.IsNotExist(err) {
					return 0, sftp.ErrSSHFxNoSuchFile
				}
				l.Logger.Errorf("Failed to list directory: %s", err)
				return 0, fmt.Errorf("failed to list directory")
			}
		} else {
			rulesPaths, err := l.getRulesPaths(acc, r.Filepath)
			if err != nil {
				return 0, internal.ToSFTPErr(err)
			}
			for i := range rulesPaths {
				infos = append(infos, dirInfo(rulesPaths[i]))
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

func (l *sshListener) statAt(r *sftp.Request, acc *model.LocalAccount) internal.ListerAtFunc {
	return func(ls []os.FileInfo, _ int64) (int, error) {
		var infos os.FileInfo
		l.Logger.Debugf("Received 'Stat' request on %s", r.Filepath)

		realDir, err := l.getRealPath(acc, r.Filepath)
		if err != nil {
			return 0, internal.ToSFTPErr(err)
		}

		if realDir != "" {
			infos, err = os.Stat(realDir)
			if err != nil {
				if os.IsNotExist(err) {
					return 0, sftp.ErrSSHFxNoSuchFile
				}
				l.Logger.Errorf("Failed to stat file %s", err)
				return 0, fmt.Errorf("failed to stat file")
			}
		} else {
			if n, err := l.DB.Count(&model.Rule{}).Where("path LIKE ?",
				r.Filepath[1:]+"%").Run(); err != nil {
				return 0, err
			} else if n == 0 {
				return 0, sftp.ErrSSHFxNoSuchFile
			}
			infos = dirInfo(path.Base(r.Filepath))
		}
		copy(ls, []os.FileInfo{infos})
		return 1, io.EOF
	}
}

func (l *sshListener) getRealPath(acc *model.LocalAccount, dir string) (string, error) {

	dir = strings.TrimPrefix(dir, "/")
	rule, err := l.getClosestRule(acc, dir, true)
	if err == sftp.ErrSSHFxNoSuchFile {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	confPaths := &conf.GlobalConfig.Paths
	rest := strings.TrimPrefix(dir, rule.Path)
	rest = strings.TrimPrefix(rest, "/")
	realDir := utils.GetPath(rest, leaf(rule.LocalDir), leaf(l.Agent.LocalOutDir),
		branch(l.Agent.Root), leaf(confPaths.DefaultOutDir), branch(confPaths.GatewayHome))
	return utils.ToOSPath(realDir), nil
}

func (l *sshListener) getClosestRule(acc *model.LocalAccount, rulePath string,
	isSendPriority bool) (*model.Rule, error) {

	rulePath = strings.TrimPrefix(rulePath, "/")
	if rulePath == "" || rulePath == "." || rulePath == "/" {
		return nil, sftp.ErrSSHFxNoSuchFile
	}

	var rule model.Rule
	if err := l.DB.Get(&rule, "path=? AND send=?", rulePath, isSendPriority).Run(); err != nil {
		if !database.IsNotFound(err) {
			l.Logger.Errorf("Failed to retrieve rule: %s", err)
			return nil, fmt.Errorf("failed to retrieve rule")
		}
		if err := l.DB.Get(&rule, "path=? AND send=?", rulePath, !isSendPriority).Run(); err != nil {
			if database.IsNotFound(err) {
				return l.getClosestRule(acc, path.Dir(rulePath), isSendPriority)
			}
			l.Logger.Errorf("Failed to retrieve rule: %s", err)
			return nil, fmt.Errorf("failed to retrieve rule")
		}
	}

	if ok, err := rule.IsAuthorized(l.DB, acc); err != nil {
		l.Logger.Errorf("Failed to check rule permissions: %s", err)
		return nil, fmt.Errorf("failed to check rule permissions")
	} else if !ok {
		return &rule, sftp.ErrSSHFxPermissionDenied
	}
	return &rule, nil
}

func (l *sshListener) getRulesPaths(acc *model.LocalAccount, dir string) ([]string, error) {
	dir = strings.TrimPrefix(dir, "/")
	var rules model.Rules
	query := l.DB.Select(&rules).Distinct("path").Where(
		`(path LIKE ?) AND
		(
			(id IN 
				(SELECT DISTINCT rule_id FROM `+model.TableRuleAccesses+` WHERE
					(object_id=? AND object_type=?) OR
					(object_id=? AND object_type=?)
				)
			)
			OR 
			( (SELECT COUNT(*) FROM `+model.TableRuleAccesses+` WHERE rule_id = id) = 0 )
		)`,
		dir+"%", acc.ID, model.TableLocAccounts, l.Agent.ID, model.TableLocAgents).
		OrderBy("path", true)
	if err := query.Run(); err != nil {
		l.Logger.Errorf("Failed to retrieve rule list: %s", err)
		return nil, err
	}
	if len(rules) == 0 {
		return nil, types.NewTransferError(types.TeFileNotFound, os.ErrNotExist.Error())
	}

	paths := make([]string, 0, len(rules))
	for i := range rules {
		p := rules[i].Path
		p = strings.TrimPrefix(p, dir)
		p = strings.SplitN(p, "/", 2)[0]
		if len(paths) == 0 || paths[len(paths)-1] != p {
			paths = append(paths, p)
		}
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
