package tasks

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errNotImplemented = errors.New("key word not implemented")

type replacer func(*Runner) (string, error)

type replacersMap map[string]replacer

//nolint:funlen //cannot split function without adding complexity and hurting readability
func getReplacers() replacersMap {
	return replacersMap{
		"#TRUEFULLPATH#": func(r *Runner) (string, error) {
			return r.transCtx.Transfer.LocalPath.String(), nil
		},
		"#TRUEFILENAME#": func(r *Runner) (string, error) {
			return path.Base(r.transCtx.Transfer.LocalPath.Path), nil
		},
		"#ORIGINALFULLPATH#": func(r *Runner) (string, error) {
			if r.transCtx.Rule.IsSend {
				return r.transCtx.Transfer.LocalPath.String(), nil
			}

			if !r.transCtx.Transfer.IsServer() {
				return r.transCtx.Transfer.RemotePath, nil
			}

			return r.transCtx.Transfer.DestFilename, nil
		},
		"#ORIGINALFILENAME#": func(r *Runner) (string, error) {
			if r.transCtx.Transfer.IsServer() && !r.transCtx.Rule.IsSend {
				return filepath.Base(r.transCtx.Transfer.DestFilename), nil
			}

			return filepath.Base(r.transCtx.Transfer.SrcFilename), nil
		},
		"#FILESIZE#": func(r *Runner) (string, error) {
			return utils.FormatInt(r.transCtx.Transfer.Filesize), nil
		},
		"#INPATH#": func(r *Runner) (string, error) {
			if in, err := makeInDir(r.transCtx); err != nil {
				return "", err
			} else {
				return in.String(), nil
			}
		},
		"#OUTPATH#": func(r *Runner) (string, error) {
			if out, err := makeOutDir(r.transCtx); err != nil {
				return "", err
			} else {
				return out.String(), nil
			}
		},
		"#WORKPATH#": func(r *Runner) (string, error) {
			if tmp, err := makeTmpDir(r.transCtx); err != nil {
				return "", err
			} else {
				return tmp.String(), nil
			}
		},
		"#ARCHPATH#": notImplemented("#ARCHPATH#"),
		"#HOMEPATH#": func(r *Runner) (string, error) {
			return r.transCtx.Paths.GatewayHome, nil
		},
		"#RULE#": func(r *Runner) (string, error) {
			return r.transCtx.Rule.Name, nil
		},
		"#DATE#": func(r *Runner) (string, error) {
			t := time.Now()

			return t.Format("20060102"), nil
		},
		"#HOUR#": func(r *Runner) (string, error) {
			t := time.Now()

			return t.Format("030405"), nil
		},
		"#REMOTEHOST#":   getRemote,
		"#REMOTEHOSTIP#": notImplemented("#REMOTEHOSTIP#"),
		"#LOCALHOST#":    getLocal,
		"#LOCALHOSTIP#":  notImplemented("#LOCALHOSTIP#"),
		"#TRANSFERID#": func(r *Runner) (string, error) {
			return utils.FormatInt(r.transCtx.Transfer.ID), nil
		},
		"#REQUESTERHOST#": getClient,
		"#REQUESTEDHOST#": getServer,
		"#FULLTRANSFERID#": func(r *Runner) (string, error) {
			// DEPRECATED
			client, err := getClient(r)
			if err != nil {
				return "", nil
			}

			server, err := getServer(r)
			if err != nil {
				return "", nil
			}

			return fmt.Sprintf("%d_%s_%s", r.transCtx.Transfer.ID, client, server), nil
		},
		"#RANKTRANSFER#": notImplemented("#RANKTRANSFER#"),
		"#BLOCKSIZE#":    notImplemented("#BLOCKSIZE#"),
		"#ERRORMSG#": func(r *Runner) (string, error) {
			return r.transCtx.Transfer.ErrDetails, nil
		},
		"#ERRORCODE#": func(r *Runner) (string, error) {
			return string(r.transCtx.Transfer.ErrCode.R66Code()), nil
		},
		"#ERRORSTRCODE#": func(r *Runner) (string, error) {
			return r.transCtx.Transfer.ErrDetails, nil
		},
		"#NOWAIT#":    notImplemented("#NOWAIT#"),
		"#LOCALEXEC#": notImplemented("#LOCALEXEC#"),
	}
}

func replaceInfo(val interface{}) replacer {
	return func(*Runner) (string, error) {
		return fmt.Sprint(val), nil
	}
}

func (r replacersMap) addInfo(c *model.TransferContext) {
	// for name, val := range c.FileInfo {
	// 	r[fmt.Sprintf("#FI_%s#", name)] = replaceInfo(val)
	// }
	for name, val := range c.TransInfo {
		r[fmt.Sprintf("#TI_%s#", name)] = replaceInfo(val)
	}
}

func notImplemented(word string) func(*Runner) (string, error) {
	return func(*Runner) (string, error) {
		return "", fmt.Errorf("%w: %s", errNotImplemented, word)
	}
}

func getLocal(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		return r.transCtx.LocalAgent.Name, nil
	}

	return r.transCtx.RemoteAccount.Login, nil
}

func getRemote(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		return r.transCtx.LocalAccount.Login, nil
	}

	return r.transCtx.RemoteAgent.Name, nil
}

func getClient(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		return r.transCtx.LocalAccount.Login, nil
	}

	return r.transCtx.RemoteAccount.Login, nil
}

func getServer(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		return r.transCtx.LocalAgent.Name, nil
	}

	return r.transCtx.RemoteAgent.Name, nil
}
