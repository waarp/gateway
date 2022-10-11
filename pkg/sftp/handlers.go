package sftp

import (
	"errors"
	"io"
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

func (l *sshListener) makeFileLister(ag *model.LocalAgent, acc *model.LocalAccount,
) internal.FileListerAtFunc {
	return func(r *sftp.Request) (sftp.ListerAt, error) {
		switch r.Method {
		case "Stat":
			return l.statAt(r, ag, acc), nil
		case "List":
			return l.listAt(r, ag, acc), nil
		default:
			return nil, sftp.ErrSSHFxOpUnsupported
		}
	}
}

func (l *sshListener) listAt(r *sftp.Request, locAgent *model.LocalAgent,
	acc *model.LocalAccount,
) internal.ListerAtFunc {
	return func(fileInfos []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debug("Received 'List' request on %s", r.Filepath)

		var (
			entries []os.DirEntry
			infos   []os.FileInfo
		)

		realDir, err := l.getRealPath(locAgent, acc, r.Filepath)
		if err != nil {
			return 0, toSFTPErr(err)
		}

		if realDir != "" {
			entries, err = os.ReadDir(realDir)
			if err != nil {
				if os.IsNotExist(err) {
					return 0, sftp.ErrSSHFxNoSuchFile
				}

				l.Logger.Error("Failed to list directory: %s", err)

				return 0, errFileSystem
			}

			for _, entry := range entries {
				info, err := entry.Info()
				if err != nil {
					l.Logger.Error("Failed to retrieve the file info: %v", err)

					return 0, errFileSystem
				}

				infos = append(infos, info)
			}
		} else {
			rulesPaths, err := l.getRulesPaths(locAgent, acc, r.Filepath)
			if err != nil {
				return 0, toSFTPErr(err)
			}

			for i := range rulesPaths {
				infos = append(infos, &internal.DirInfo{Dir: rulesPaths[i]})
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

func (l *sshListener) statAt(r *sftp.Request, ag *model.LocalAgent,
	acc *model.LocalAccount,
) internal.ListerAtFunc {
	return func(fileInfos []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debug("Received 'Stat' request on %s", r.Filepath)

		var infos os.FileInfo

		realDir, err := l.getRealPath(ag, acc, r.Filepath)
		if err != nil {
			return 0, toSFTPErr(err)
		}

		if realDir != "" {
			infos, err = os.Stat(realDir)
			if err != nil {
				if os.IsNotExist(err) {
					return 0, sftp.ErrSSHFxNoSuchFile
				}

				l.Logger.Error("Failed to stat file %s", err)

				return 0, errFileSystem
			}
		} else {
			if n, err := l.DB.Count(&model.Rule{}).Where("path LIKE ?",
				r.Filepath[1:]+"%").Run(); err != nil {
				return 0, err
			} else if n == 0 {
				return 0, sftp.ErrSSHFxNoSuchFile
			}
			infos = &internal.DirInfo{Dir: path.Base(r.Filepath)}
		}

		copy(fileInfos, []os.FileInfo{infos})

		return 1, io.EOF
	}
}

func (l *sshListener) getRealPath(ag *model.LocalAgent, acc *model.LocalAccount,
	dir string,
) (string, error) {
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
	realDir := utils.GetPath(rest, leaf(rule.LocalDir), leaf(ag.SendDir),
		branch(ag.RootDir), leaf(confPaths.DefaultOutDir),
		branch(confPaths.GatewayHome))

	return realDir, nil
}

func (l *sshListener) getClosestRule(acc *model.LocalAccount, rulePath string,
	isSendPriority bool,
) (*model.Rule, error) {
	rulePath = strings.TrimPrefix(rulePath, "/")
	if rulePath == "" || rulePath == "." || rulePath == "/" {
		return nil, sftp.ErrSSHFxNoSuchFile
	}

	var rule model.Rule
	if err := l.DB.Get(&rule, "path=? AND send=?", rulePath, isSendPriority).Run(); err != nil {
		if !database.IsNotFound(err) {
			l.Logger.Error("Failed to retrieve rule: %s", err)

			return nil, errDatabase
		}

		if err := l.DB.Get(&rule, "path=? AND send=?", rulePath, !isSendPriority).Run(); err != nil {
			if database.IsNotFound(err) {
				return l.getClosestRule(acc, path.Dir(rulePath), isSendPriority)
			}

			l.Logger.Error("Failed to retrieve rule: %s", err)

			return nil, errDatabase
		}
	}

	if ok, err := rule.IsAuthorized(l.DB, acc); err != nil {
		l.Logger.Error("Failed to check rule permissions: %s", err)

		return nil, errDatabase
	} else if !ok {
		return &rule, sftp.ErrSSHFxPermissionDenied
	}

	return &rule, nil
}

func (l *sshListener) getRulesPaths(ag *model.LocalAgent, acc *model.LocalAccount,
	dir string,
) ([]string, error) {
	dir = strings.TrimPrefix(path.Clean(dir), "/")

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
		dir+"%", acc.ID, model.TableLocAccounts, ag.ID, model.TableLocAgents).
		OrderBy("path", true)

	if err := query.Run(); err != nil {
		l.Logger.Error("Failed to retrieve rule list: %s", err)

		return nil, err
	}

	if len(rules) == 0 {
		return nil, sftp.ErrSSHFxNoSuchFile
	}

	paths := make([]string, 0, len(rules))
	dir += "/"

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
