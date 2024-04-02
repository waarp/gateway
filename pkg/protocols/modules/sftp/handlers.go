package sftp

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type (
	leaf   = utils.Leaf
	branch = utils.Branch
)

func (l *sshListener) makeFileCmder(acc *model.LocalAccount) internal.CmdFunc {
	return func(r *sftp.Request) error {
		switch r.Method {
		case "Mkdir":
			return l.mkdir(r, acc)
		default:
			return sftp.ErrSSHFxOpUnsupported
		}
	}
}

func (l *sshListener) mkdir(r *sftp.Request, acc *model.LocalAccount) error {
	l.Logger.Debug("Received 'Mkdir' request on %s", r.Filepath)

	realDir, dirErr := l.getRealPath(acc, r.Filepath)
	if dirErr != nil {
		return dirErr
	}

	filesys, fsErr := fs.GetFileSystem(l.DB, realDir)
	if fsErr != nil {
		return ErrFileSystem
	}

	if err := fs.MkdirAll(filesys, realDir); err != nil {
		//nolint:goerr113 //too specific
		return errors.New("failed to create directory")
	}

	return nil
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

func (l *sshListener) listAt(r *sftp.Request, acc *model.LocalAccount,
) internal.ListerAtFunc {
	return func(fileInfos []fs.FileInfo, offset int64) (int, error) {
		l.Logger.Debug("Received 'List' request on %s", r.Filepath)

		var infos []fs.FileInfo

		realDir, pathErr := l.getRealPath(acc, r.Filepath)
		if pathErr != nil {
			return 0, pathErr
		}

		if realDir != nil {
			var listErr error
			if infos, listErr = l.listReadDir(realDir); listErr != nil {
				return 0, listErr
			}
		} else {
			rulesPaths, rulesErr := l.getRulesPaths(acc, r.Filepath)
			if rulesErr != nil {
				return 0, rulesErr
			}

			for _, rulePath := range rulesPaths {
				infos = append(infos, &internal.DirInfo{Dir: rulePath})
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

func (l *sshListener) statAt(r *sftp.Request, acc *model.LocalAccount,
) internal.ListerAtFunc {
	return func(fileInfos []fs.FileInfo, offset int64) (int, error) {
		l.Logger.Debug("Received 'Stat' request on %s", r.Filepath)

		var infos fs.FileInfo

		realDir, err := l.getRealPath(acc, r.Filepath)
		if err != nil {
			return 0, err
		}

		if realDir != nil {
			filesys, fsErr := fs.GetFileSystem(l.DB, realDir)
			if fsErr != nil {
				return 0, ErrFileSystem
			}

			infos, err = fs.Stat(filesys, realDir)
			if err != nil {
				if fs.IsNotExist(err) {
					return 0, sftp.ErrSSHFxNoSuchFile
				}

				l.Logger.Error("Failed to stat file %s", err)

				return 0, ErrFileSystem
			}
		} else {
			if n, err := l.DB.Count(&model.Rule{}).Where("path LIKE ?",
				r.Filepath[1:]+"%").Run(); err != nil {
				return 0, fmt.Errorf("failed to retrieve rules: %w", err)
			} else if n == 0 {
				return 0, sftp.ErrSSHFxNoSuchFile
			}

			infos = &internal.DirInfo{Dir: path.Base(r.Filepath)}
		}

		copy(fileInfos, []fs.FileInfo{infos})

		return 1, io.EOF
	}
}

func (l *sshListener) listReadDir(realDir *types.URL) ([]fs.FileInfo, error) {
	var infos []fs.FileInfo

	filesys, fsErr := fs.GetFileSystem(l.DB, realDir)
	if fsErr != nil {
		return nil, ErrFileSystem
	}

	entries, readErr := fs.ReadDir(filesys, realDir)
	if readErr != nil {
		if fs.IsNotExist(readErr) {
			return nil, sftp.ErrSSHFxNoSuchFile
		}

		l.Logger.Error("Failed to list directory: %s", readErr)

		return nil, ErrFileSystem
	}

	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr != nil {
			l.Logger.Error("Failed to retrieve the file info: %v", infoErr)

			return nil, ErrFileSystem
		}

		infos = append(infos, info)
	}

	return infos, nil
}

func (l *sshListener) getRealPath(acc *model.LocalAccount, dir string,
) (*types.URL, error) {
	dir = strings.TrimPrefix(dir, "/")

	rule, err := l.getClosestRule(acc, dir, true)
	if errors.Is(err, sftp.ErrSSHFxNoSuchFile) {
		return nil, nil //nolint:nilnil //returning nil here makes more sense than using a sentinel error
	}

	if err != nil {
		return nil, err
	}

	confPaths := &conf.GlobalConfig.Paths
	rest := strings.TrimPrefix(dir, rule.Path)
	rest = strings.TrimPrefix(rest, "/")

	realDir, dirErr := utils.GetPath(rest, leaf(rule.LocalDir), leaf(l.Server.SendDir),
		branch(l.Server.RootDir), leaf(confPaths.DefaultOutDir),
		branch(confPaths.GatewayHome))
	if dirErr != nil {
		l.Logger.Error("Failed to retrieve real path for request %q: %s", dir, dirErr)

		return nil, ErrInternal
	}

	return (*types.URL)(realDir), nil
}

func (l *sshListener) getClosestRule(acc *model.LocalAccount, rulePath string,
	isSendPriority bool,
) (*model.Rule, error) {
	rulePath = strings.TrimPrefix(rulePath, "/")
	if rulePath == "" || rulePath == "." || rulePath == "/" {
		return nil, sftp.ErrSSHFxNoSuchFile
	}

	var rule model.Rule
	if err := l.DB.Get(&rule, "path=? AND is_send=?", rulePath, isSendPriority).Run(); err != nil {
		if !database.IsNotFound(err) {
			l.Logger.Error("Failed to retrieve rule: %s", err)

			return nil, ErrDatabase
		}

		if err := l.DB.Get(&rule, "path=? AND is_send=?", rulePath, !isSendPriority).Run(); err != nil {
			if database.IsNotFound(err) {
				return l.getClosestRule(acc, path.Dir(rulePath), isSendPriority)
			}

			l.Logger.Error("Failed to retrieve rule: %s", err)

			return nil, ErrDatabase
		}
	}

	if ok, err := rule.IsAuthorized(l.DB, acc); err != nil {
		l.Logger.Error("Failed to check rule permissions: %s", err)

		return nil, ErrDatabase
	} else if !ok {
		return &rule, sftp.ErrSSHFxPermissionDenied
	}

	return &rule, nil
}

func (l *sshListener) getRulesPaths(acc *model.LocalAccount, dir string,
) ([]string, error) {
	dir = strings.TrimPrefix(path.Clean(dir), "/")

	var rules model.Rules

	query := l.DB.Select(&rules).Distinct("path").Where(
		`(path LIKE ?) AND
		(
			(id IN 
				(SELECT DISTINCT rule_id FROM `+model.TableRuleAccesses+` WHERE
					(local_account_id=? OR local_agent_id=?)
				)
			)
			OR 
			( (SELECT COUNT(*) FROM `+model.TableRuleAccesses+` WHERE rule_id = id) = 0 )
		)`,
		dir+"%", acc.ID, l.Server.ID).OrderBy("path", true)

	if err := query.Run(); err != nil {
		l.Logger.Error("Failed to retrieve rule list: %s", err)

		//nolint:goerr113 //too specific
		return nil, errors.New("failed to retrieve rule list")
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
