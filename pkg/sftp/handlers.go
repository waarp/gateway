package sftp

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
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

func (l *SSHListener) makeFileLister(acc *model.LocalAccount) internal.FileListerAtFunc {
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

func (l *SSHListener) listAt(r *sftp.Request, acc *model.LocalAccount) internal.ListerAtFunc {
	return func(ls []os.FileInfo, offset int64) (int, error) {
		var infos []os.FileInfo

		l.Logger.Debugf("Received 'List' request on %s", r.Filepath)
		if r.Filepath == "" || r.Filepath == "/" {
			paths, err := internal.GetRulesPaths(l.DB, l.Logger, l.Agent, acc)
			if err != nil {
				return 0, fmt.Errorf("cannot list rule directories")
			}
			for i := range paths {
				infos = append(infos, internal.DirInfo(paths[i]))
			}
		} else {
			rule, err := internal.GetListRule(l.DB, l.Logger, acc, l.Agent, r.Filepath)
			if err != nil {
				return 0, err
			}
			if rule != nil {
				dir := utils.GetPath("", leaf(rule.LocalDir), leaf(l.Agent.LocalOutDir),
					branch(l.Agent.Root), leaf(l.GWConf.Paths.DefaultOutDir),
					branch(l.GWConf.Paths.GatewayHome))

				infos, err = ioutil.ReadDir(utils.ToOSPath(dir))
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

func (l *SSHListener) statAt(r *sftp.Request, acc *model.LocalAccount) internal.ListerAtFunc {
	return func(ls []os.FileInfo, offset int64) (int, error) {
		l.Logger.Debugf("Received 'Stat' request on %s", r.Filepath)

		rule, err := internal.GetRule(l.DB, l.Logger, acc, l.Agent, path.Dir(r.Filepath), true)
		if err != nil || rule == nil {
			return 0, fmt.Errorf("failed to retrieve rule for path '%s'", r.Filepath)
		}

		file := utils.GetPath(path.Base(r.Filepath), leaf(rule.LocalDir),
			leaf(l.Agent.LocalOutDir), branch(l.Agent.Root),
			leaf(l.GWConf.Paths.DefaultOutDir), branch(l.GWConf.Paths.GatewayHome))

		fi, err := os.Stat(utils.ToOSPath(file))
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
