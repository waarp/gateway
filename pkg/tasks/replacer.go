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

type replacer func(*model.TransferContext) (string, error)

type replacersMap map[string]replacer

//nolint:funlen //cannot split function without adding complexity and hurting readability
func getReplacers() replacersMap {
	return replacersMap{
		"#TRUEFULLPATH#": func(ctx *model.TransferContext) (string, error) {
			return ctx.Transfer.LocalPath, nil
		},
		"#TRUEFILENAME#": func(ctx *model.TransferContext) (string, error) {
			return path.Base(ctx.Transfer.LocalPath), nil
		},
		"#ORIGINALFULLPATH#": func(ctx *model.TransferContext) (string, error) {
			if ctx.Rule.IsSend {
				return ctx.Transfer.LocalPath, nil
			}

			if !ctx.Transfer.IsServer() {
				return ctx.Transfer.RemotePath, nil
			}

			return ctx.Transfer.DestFilename, nil
		},
		"#ORIGINALFILENAME#": func(ctx *model.TransferContext) (string, error) {
			if ctx.Transfer.IsServer() && !ctx.Rule.IsSend {
				return filepath.Base(ctx.Transfer.DestFilename), nil
			}

			return filepath.Base(ctx.Transfer.SrcFilename), nil
		},
		"#FILESIZE#": func(ctx *model.TransferContext) (string, error) {
			return utils.FormatInt(ctx.Transfer.Filesize), nil
		},
		"#INPATH#":   makeInDir,
		"#OUTPATH#":  makeOutDir,
		"#WORKPATH#": makeTmpDir,
		"#ARCHPATH#": notImplemented("#ARCHPATH#"),
		"#HOMEPATH#": func(ctx *model.TransferContext) (string, error) {
			return ctx.Paths.GatewayHome, nil
		},
		"#RULE#": func(ctx *model.TransferContext) (string, error) {
			return ctx.Rule.Name, nil
		},
		"#DATE#": func(*model.TransferContext) (string, error) {
			t := time.Now()

			return t.Format("20060102"), nil
		},
		"#HOUR#": func(*model.TransferContext) (string, error) {
			t := time.Now()

			return t.Format("030405"), nil
		},
		"#REMOTEHOST#":   getRemote,
		"#REMOTEHOSTIP#": notImplemented("#REMOTEHOSTIP#"),
		"#LOCALHOST#":    getLocal,
		"#LOCALHOSTIP#":  notImplemented("#LOCALHOSTIP#"),
		"#TRANSFERID#": func(ctx *model.TransferContext) (string, error) {
			return utils.FormatInt(ctx.Transfer.ID), nil
		},
		"#REQUESTERHOST#": getClient,
		"#REQUESTEDHOST#": getServer,
		"#FULLTRANSFERID#": func(ctx *model.TransferContext) (string, error) {
			// DEPRECATED
			client, err := getClient(ctx)
			if err != nil {
				return "", nil
			}

			server, err := getServer(ctx)
			if err != nil {
				return "", nil
			}

			return fmt.Sprintf("%d_%s_%s", ctx.Transfer.ID, client, server), nil
		},
		"#RANKTRANSFER#": notImplemented("#RANKTRANSFER#"),
		"#BLOCKSIZE#":    notImplemented("#BLOCKSIZE#"),
		"#ERRORMSG#": func(ctx *model.TransferContext) (string, error) {
			return ctx.Transfer.ErrDetails, nil
		},
		"#ERRORCODE#": func(ctx *model.TransferContext) (string, error) {
			return string(ctx.Transfer.ErrCode.R66Code()), nil
		},
		"#ERRORSTRCODE#": func(ctx *model.TransferContext) (string, error) {
			return ctx.Transfer.ErrDetails, nil
		},
		"#NOWAIT#":    notImplemented("#NOWAIT#"),
		"#LOCALEXEC#": notImplemented("#LOCALEXEC#"),
	}
}

func replaceInfo(val any) replacer {
	return func(*model.TransferContext) (string, error) {
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

func notImplemented(word string) func(*model.TransferContext) (string, error) {
	return func(*model.TransferContext) (string, error) {
		return "", fmt.Errorf("%w: %s", errNotImplemented, word)
	}
}

func getLocal(ctx *model.TransferContext) (string, error) {
	if ctx.Transfer.IsServer() {
		return ctx.LocalAgent.Name, nil
	}

	return ctx.RemoteAccount.Login, nil
}

func getRemote(ctx *model.TransferContext) (string, error) {
	if ctx.Transfer.IsServer() {
		return ctx.LocalAccount.Login, nil
	}

	return ctx.RemoteAgent.Name, nil
}

func getClient(ctx *model.TransferContext) (string, error) {
	if ctx.Transfer.IsServer() {
		return ctx.LocalAccount.Login, nil
	}

	return ctx.RemoteAccount.Login, nil
}

func getServer(ctx *model.TransferContext) (string, error) {
	if ctx.Transfer.IsServer() {
		return ctx.LocalAgent.Name, nil
	}

	return ctx.RemoteAgent.Name, nil
}
