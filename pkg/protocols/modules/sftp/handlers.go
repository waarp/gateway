package sftp

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protoutils"
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

	if err := fs.MkdirAll(realDir); err != nil {
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
	return func(fileInfos []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debug("Received 'List' request on %s", r.Filepath)

		var infos []os.FileInfo

		realDir, pathErr := l.getRealPath(acc, r.Filepath)
		if pathErr != nil {
			return 0, pathErr
		}

		if realDir != "" {
			var listErr error
			if infos, listErr = l.listReadDir(realDir); listErr != nil {
				return 0, listErr
			}
		} else {
			rulesPaths, rulesErr := l.getRulesPaths(acc, r.Filepath)
			if rulesErr != nil {
				return 0, rulesErr
			}

			infos = append(infos, rulesPaths...)
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
	return func(fileInfos []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debug("Received 'Stat' request on %s", r.Filepath)

		var infos os.FileInfo

		realDir, err := l.getRealPath(acc, r.Filepath)
		if err != nil {
			return 0, err
		}

		if realDir != "" {
			infos, err = fs.Stat(realDir)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
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

			infos = protoutils.FakeDirInfo(path.Base(r.Filepath))
		}

		copy(fileInfos, []os.FileInfo{infos})

		return 1, io.EOF
	}
}

func (l *sshListener) listReadDir(realDir string) ([]os.FileInfo, error) {
	entries, readErr := fs.List(realDir)
	if readErr != nil {
		if errors.Is(readErr, fs.ErrNotExist) {
			return nil, sftp.ErrSSHFxNoSuchFile
		}

		l.Logger.Error("Failed to list directory: %s", readErr)

		return nil, ErrFileSystem
	}

	infos := make([]os.FileInfo, len(entries))

	for i, entry := range entries {
		var err error
		if infos[i], err = entry.Info(); err != nil {
			l.Logger.Error("Failed to retrieve the file info: %v", err)

			return nil, ErrFileSystem
		}
	}

	return infos, nil
}

func (l *sshListener) getRealPath(acc *model.LocalAccount, dir string,
) (string, error) {
	realPath, err := protoutils.GetRealPath(false, l.DB, l.Logger, l.Server, acc, dir)

	switch {
	case errors.Is(err, protoutils.ErrPermissionDenied):
		return "", sftp.ErrSSHFxPermissionDenied
	case errors.Is(err, protoutils.ErrRuleNotFound):
		return "", sftp.ErrSSHFxNoSuchFile
	case err != nil:
		return "", err //nolint:wrapcheck //no need to wrap here
	}

	return realPath, nil
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
) ([]os.FileInfo, error) {
	entries, err := protoutils.GetRulesPaths(l.DB, l.Server, acc, dir)
	if errors.Is(err, protoutils.ErrRuleNotFound) {
		return nil, sftp.ErrSSHFxNoSuchFile
	} else if err != nil {
		l.Logger.Error("Failed to retrieve rules list: %s", err)

		return nil, err //nolint:wrapcheck //no need to wrap here
	}

	return entries, nil
}
