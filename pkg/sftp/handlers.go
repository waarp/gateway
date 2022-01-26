package sftp

import (
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/sftp/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

func leaf(str string) utils.Leaf     { return utils.Leaf(str) }
func branch(str string) utils.Branch { return utils.Branch(str) }

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
	return func(fileInfos []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debugf("Received 'List' request on %s", r.Filepath)

		var infos []os.FileInfo

		realDir, err := l.getRealPath(acc, r.Filepath)
		if err != nil {
			return 0, toSFTPErr(err)
		}

		if realDir != "" {
			infos, err = ioutil.ReadDir(realDir)
			if err != nil {
				if os.IsNotExist(err) {
					return 0, sftp.ErrSSHFxNoSuchFile
				}

				l.Logger.Errorf("Failed to list directory: %s", err)

				return 0, errFileSystem
			}
		} else {
			rulesPaths, err := l.getRulesPaths(acc, r.Filepath)
			if err != nil {
				return 0, toSFTPErr(err)
			}
			for i := range rulesPaths {
				infos = append(infos, internal.DirInfo(rulesPaths[i]))
			}
		}

		if offset >= int64(len(infos)) {
			return 0, io.EOF
		}

		n := copy(fileInfos, infos[offset:])
		if n < len(fileInfos) {
			return n, io.EOF
		}

		return n, nil
	}
}

func (l *sshListener) statAt(r *sftp.Request, acc *model.LocalAccount) internal.ListerAtFunc {
	return func(fileInfos []os.FileInfo, _ int64) (int, error) {
		l.Logger.Debugf("Received 'Stat' request on %s", r.Filepath)

		var infos os.FileInfo

		realDir, err := l.getRealPath(acc, r.Filepath)
		if err != nil {
			return 0, toSFTPErr(err)
		}

		if realDir != "" {
			infos, err = os.Stat(realDir)
			if err != nil {
				if os.IsNotExist(err) {
					return 0, sftp.ErrSSHFxNoSuchFile
				}

				l.Logger.Errorf("Failed to stat file %s", err)

				return 0, errFileSystem
			}
		} else {
			if n, err := l.DB.Count(&model.Rule{}).Where("path LIKE ?",
				r.Filepath[1:]+"%").Run(); err != nil {
				return 0, err
			} else if n == 0 {
				return 0, sftp.ErrSSHFxNoSuchFile
			}
			infos = internal.DirInfo(path.Base(r.Filepath))
		}

		copy(fileInfos, []os.FileInfo{infos})

		return 1, io.EOF
	}
}

func (l *sshListener) getRealPath(acc *model.LocalAccount, dir string) (string, error) {
	dir = strings.TrimPrefix(dir, "/")

	rule, err := l.getClosestRule(acc, dir, true)
	if errors.Is(err, sftp.ErrSSHFxNoSuchFile) {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	confPaths := &conf.GlobalConfig.Paths
	rest := strings.TrimPrefix(dir, rule.Path)
	rest = strings.TrimPrefix(rest, "/")
	realDir := utils.GetPath(rest, leaf(rule.LocalDir), leaf(l.Agent.SendDir),
		branch(l.Agent.RootDir), leaf(confPaths.DefaultOutDir),
		branch(confPaths.GatewayHome))

	return realDir, nil
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

			return nil, errDatabase
		}

		if err := l.DB.Get(&rule, "path=? AND send=?", rulePath, !isSendPriority).Run(); err != nil {
			if database.IsNotFound(err) {
				return l.getClosestRule(acc, path.Dir(rulePath), isSendPriority)
			}

			l.Logger.Errorf("Failed to retrieve rule: %s", err)

			return nil, errDatabase
		}
	}

	if ok, err := rule.IsAuthorized(l.DB, acc); err != nil {
		l.Logger.Errorf("Failed to check rule permissions: %s", err)

		return nil, errDatabase
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
		return nil, sftp.ErrSSHFxNoSuchFile
	}

	paths := make([]string, 0, len(rules))

	for i := range rules {
		p := rules[i].Path
		p = strings.TrimPrefix(p, dir)
		p = strings.SplitN(p, "/", 2)[0] //nolint:gomnd //not needed here

		if len(paths) == 0 || paths[len(paths)-1] != p {
			paths = append(paths, p)
		}
	}

	return paths, nil
}
