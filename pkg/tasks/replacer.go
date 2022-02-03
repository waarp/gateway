package tasks

import (
	"errors"
	"fmt"
	"path"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var errNotImplemented = errors.New("key word not implemented")

type replacer func(*Runner) (string, error)

//nolint:gochecknoglobals // can hardly be otherwise, or it should be designed another way
var replacers = map[string]replacer{
	"#TRUEFULLPATH#": func(r *Runner) (string, error) {
		return r.transCtx.Transfer.LocalPath, nil
	},
	"#TRUEFILENAME#": func(r *Runner) (string, error) {
		return path.Base(r.transCtx.Transfer.LocalPath), nil
	},
	"#ORIGINALFULLPATH#": func(r *Runner) (string, error) {
		if r.transCtx.Rule.IsSend {
			return utils.ToOSPath(r.transCtx.Transfer.LocalPath), nil
		}

		return r.transCtx.Transfer.RemotePath, nil
	},
	"#ORIGINALFILENAME#": func(r *Runner) (string, error) {
		if r.transCtx.Rule.IsSend {
			return path.Base(r.transCtx.Transfer.LocalPath), nil
		}

		return path.Base(r.transCtx.Transfer.RemotePath), nil
	},
	"#FILESIZE#": func(r *Runner) (string, error) {
		return fmt.Sprint(r.transCtx.Transfer.Filesize), nil
	},
	"#INPATH#":   notImplemented("#INPATH#"),
	"#OUTPATH#":  notImplemented("#OUTPATH#"),
	"#WORKPATH#": notImplemented("#WORKPATH#"),
	"#ARCHPATH#": notImplemented("#ARCHPATH#"),
	"#HOMEPATH#": notImplemented("#HOMEPATH#"),
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
	"#TRANFERID#": func(r *Runner) (string, error) {
		return fmt.Sprint(r.transCtx.Transfer.ID), nil
	},
	"#REQUESTERHOST#": func(r *Runner) (string, error) {
		client, err := getClient(r)

		return client, err
	},
	"#REQUESTEDHOST#": func(r *Runner) (string, error) {
		server, err := getServer(r)

		return server, err
	},
	"#FULLTRANFERID#": func(r *Runner) (string, error) {
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
		return r.transCtx.Transfer.Error.Details, nil
	},
	"#ERRORCODE#": func(r *Runner) (string, error) {
		return string(r.transCtx.Transfer.Error.Code.R66Code()), nil
	},
	"#ERRORSTRCODE#": func(r *Runner) (string, error) {
		return r.transCtx.Transfer.Error.Details, nil
	},
	"#NOWAIT#":    notImplemented("#NOWAIT#"),
	"#LOCALEXEC#": notImplemented("#LOCALEXEC#"),
}

func notImplemented(word string) func(*Runner) (string, error) {
	return func(*Runner) (string, error) {
		return "", fmt.Errorf("%w: %s", errNotImplemented, word)
	}
}

//nolint:dupl //factorising would add complexity
func getLocal(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer {
		var agent model.LocalAgent
		if err := r.db.Get(&agent, "id=?", r.transCtx.Transfer.AgentID).Run(); err != nil {
			return "", err
		}

		return agent.Name, nil
	}

	var account model.RemoteAccount
	if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.AccountID).Run(); err != nil {
		return "", err
	}

	return account.Login, nil
}

//nolint:dupl //factorising would add complexity
func getRemote(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer {
		var account model.LocalAccount
		if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.AccountID).Run(); err != nil {
			return "", err
		}

		return account.Login, nil
	}

	var agent model.RemoteAgent
	if err := r.db.Get(&agent, "id=?", r.transCtx.Transfer.AgentID).Run(); err != nil {
		return "", err
	}

	return agent.Name, nil
}

//nolint:dupl //factorising would add complexity
func getClient(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer {
		var account model.LocalAccount
		if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.AccountID).Run(); err != nil {
			return "", err
		}

		return account.Login, nil
	}

	var account model.RemoteAccount
	if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.AccountID).Run(); err != nil {
		return "", err
	}

	return account.Login, nil
}

//nolint:dupl //factorising would add complexity
func getServer(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer {
		var agent model.LocalAgent
		if err := r.db.Get(&agent, "id=?", r.transCtx.Transfer.AgentID).Run(); err != nil {
			return "", err
		}

		return agent.Name, nil
	}

	var agent model.RemoteAgent
	if err := r.db.Get(&agent, "id=?", r.transCtx.Transfer.AgentID).Run(); err != nil {
		return "", err
	}

	return agent.Name, nil
}
