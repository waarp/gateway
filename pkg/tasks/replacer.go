package tasks

import (
	"fmt"
	"path/filepath"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type replacer func(*Runner) (string, error)

var replacers = map[string]replacer{
	"#TRUEFULLPATH#": func(r *Runner) (string, error) {
		return r.info.Transfer.TrueFilepath, nil
	},
	"#TRUEFILENAME#": func(r *Runner) (string, error) {
		if r.info.Rule.IsSend {
			return filepath.Clean(r.info.Transfer.SourceFile), nil
		}
		return filepath.Clean(r.info.Transfer.DestFile), nil
	},
	"#ORIGINALFULLPATH#": func(r *Runner) (string, error) {
		if r.info.Rule.IsSend {
			return r.info.Transfer.SourceFile, nil
		}
		return r.info.Transfer.DestFile, nil
	},
	"#ORIGINALFILENAME#": func(r *Runner) (string, error) {
		if r.info.Rule.IsSend {
			return filepath.Clean(r.info.Transfer.SourceFile), nil
		}
		return filepath.Clean(r.info.Transfer.DestFile), nil
	},
	"#FILESIZE#": func(r *Runner) (string, error) {
		return "0", nil
	},
	"#INPATH#": func(r *Runner) (string, error) {
		// DEPRECATED
		return "", nil
	},
	"#OUTPATH#": func(r *Runner) (string, error) {
		// DEPRECATED
		return "", nil
	},
	"#WORKPATH#": func(r *Runner) (string, error) {
		// DEPRECATED
		return "", nil
	},
	"#ARCHPATH#": func(r *Runner) (string, error) {
		// DEPRECATED
		return "", nil
	},
	"#HOMEPATH#": func(r *Runner) (string, error) {
		// TODO ???
		return "", nil
	},
	"#RULE#": func(r *Runner) (string, error) {
		return r.info.Rule.Name, nil
	},
	"#DATE#": func(r *Runner) (string, error) {
		t := time.Now()
		return t.Format("20060102"), nil
	},
	"#HOUR#": func(r *Runner) (string, error) {
		t := time.Now()
		return t.Format("030405"), nil
	},
	"#REMOTEHOST#": func(r *Runner) (string, error) {
		if r.info.Transfer.IsServer {
			account := &model.LocalAccount{}
			if err := r.db.Get(account, "id=?", r.info.Transfer.AccountID).Run(); err != nil {
				return "", err
			}
			return account.Login, nil
		}
		agent := &model.RemoteAgent{}
		if err := r.db.Get(agent, "id=?", r.info.Transfer.AgentID).Run(); err != nil {
			return "", err
		}
		return agent.Name, nil
	},
	"#REMOTEHOSTIP#": func(r *Runner) (string, error) {
		// TODO
		return "", nil
	},
	"#LOCALHOST#": func(r *Runner) (string, error) {
		if r.info.Transfer.IsServer {
			agent := &model.LocalAgent{}
			if err := r.db.Get(agent, "id=?", r.info.Transfer.AgentID).Run(); err != nil {
				return "", err
			}
			return agent.Name, nil
		}
		account := &model.RemoteAccount{}
		if err := r.db.Get(account, "id=?", r.info.Transfer.AccountID).Run(); err != nil {
			return "", err
		}
		return account.Login, nil
	},
	"#LOCALHOSTIP#": func(r *Runner) (string, error) {
		// TODO
		return "", nil
	},
	"#TRANFERID#": func(r *Runner) (string, error) {
		return fmt.Sprint(r.info.Transfer.ID), nil
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
		//DEPRECATED
		client, err := getClient(r)
		if err != nil {
			return "", nil
		}
		server, err := getServer(r)
		if err != nil {
			return "", nil
		}
		return fmt.Sprintf("%d_%s_%s", r.info.Transfer.ID, client, server), nil
	},
	"#RANKTRANSFER#": func(r *Runner) (string, error) {
		return "0", nil
	},
	"#BLOCKSIZE#": func(r *Runner) (string, error) {
		return "1", nil
	},
	"#ERRORMSG#": func(r *Runner) (string, error) {
		return r.info.Transfer.Error.Details, nil
	},
	"#ERRORCODE#": func(r *Runner) (string, error) {
		return string(r.info.Transfer.Error.Code.R66Code()), nil
	},
	"#ERRORSTRCODE#": func(r *Runner) (string, error) {
		return r.info.Transfer.Error.Details, nil
	},
	"#NOWAIT#": func(r *Runner) (string, error) {
		return "", nil
	},
	"#LOCALEXEC#": func(r *Runner) (string, error) {
		return "", nil
	},
}

func getClient(r *Runner) (string, error) {
	if r.info.Transfer.IsServer {
		account := &model.LocalAccount{}
		if err := r.db.Get(account, "id=?", r.info.Transfer.AccountID).Run(); err != nil {
			return "", err
		}
		return account.Login, nil
	}
	account := &model.RemoteAccount{}
	if err := r.db.Get(account, "id=?", r.info.Transfer.AccountID).Run(); err != nil {
		return "", err
	}
	return account.Login, nil
}

func getServer(r *Runner) (string, error) {
	if r.info.Transfer.IsServer {
		agent := &model.LocalAgent{}
		if err := r.db.Get(agent, "id=?", r.info.Transfer.AgentID).Run(); err != nil {
			return "", err
		}
		return agent.Name, nil
	}
	agent := &model.RemoteAgent{}
	if err := r.db.Get(agent, "id=?", r.info.Transfer.AgentID).Run(); err != nil {
		return "", err
	}
	return agent.Name, nil
}
