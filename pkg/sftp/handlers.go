package sftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/sftp"

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

		if r.Filepath == "" || r.Filepath == "/" {
			paths, err := getRulesPaths(l.DB, l.Logger, l.Agent, acc)
			if err != nil {
				return 0, fmt.Errorf("cannot list rule directories: %w", err)
			}

			for i := range paths {
				infos = append(infos, internal.DirInfo(paths[i]))
			}
		} else {
			rule, err := getListRule(l.DB, l.Logger, acc, r.Filepath)
			if err != nil {
				return 0, err
			}

			if rule != nil {
				dir := utils.GetPath("", leaf(rule.LocalDir), leaf(l.Agent.SendDir),
					branch(l.Agent.RootDir), leaf(l.DB.Conf.Paths.DefaultOutDir),
					branch(l.DB.Conf.Paths.GatewayHome))

				infos, err = ioutil.ReadDir(utils.ToOSPath(dir))
				if err != nil {
					l.Logger.Errorf("Failed to list directory: %v", err)

					return 0, fmt.Errorf("failed to list directory: %w", err)
				}
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
	return func(fileInfos []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debugf("Received 'Stat' request on %s", r.Filepath)

		rule, err := getRule(l.DB, l.Logger, acc, path.Dir(r.Filepath), true)
		if err != nil {
			return 0, err
		}

		file := utils.GetPath(path.Base(r.Filepath), leaf(rule.LocalDir),
			leaf(l.Agent.SendDir), branch(l.Agent.RootDir),
			leaf(l.DB.Conf.Paths.DefaultOutDir), branch(l.DB.Conf.Paths.GatewayHome))

		fi, err := os.Stat(utils.ToOSPath(file))
		if err != nil {
			if os.IsNotExist(err) {
				return 0, sftp.ErrSSHFxNoSuchFile
			}

			l.Logger.Errorf("Failed to get file stats: %s", r.Filepath)

			return 0, fmt.Errorf("failed to get file stats: %w", err)
		}

		tmp := []os.FileInfo{fi}

		n := copy(fileInfos, tmp)
		if n < len(fileInfos) {
			return n, io.EOF
		}

		return n, nil
	}
}
