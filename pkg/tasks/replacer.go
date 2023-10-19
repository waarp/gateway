package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var errNotImplemented = errors.New("key word not implemented")

type replacer func(*Runner) (string, error)

type replacersMap map[string]replacer

//nolint:funlen //cannot split function without adding complexity and hurting readability
func getReplacers() replacersMap {
	return replacersMap{
		"#TRUEFULLPATH#": func(r *Runner) (string, error) {
			return r.transCtx.Transfer.LocalPath, nil
		},
		"#TRUEFILENAME#": func(r *Runner) (string, error) {
			return filepath.Base(r.transCtx.Transfer.LocalPath), nil
		},
		"#ORIGINALFULLPATH#": func(r *Runner) (string, error) {
			if r.transCtx.Rule.IsSend {
				return r.transCtx.Transfer.LocalPath, nil
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
			return fmt.Sprint(r.transCtx.Transfer.Filesize), nil
		},
		"#INPATH#": func(r *Runner) (string, error) {
			return makeInDir(r.transCtx), nil
		},
		"#OUTPATH#": func(r *Runner) (string, error) {
			return makeOutDir(r.transCtx), nil
		},
		"#WORKPATH#": func(r *Runner) (string, error) {
			return makeTmpDir(r.transCtx), nil
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
}

func replaceInfo(val interface{}) replacer {
	return func(*Runner) (string, error) {
		jVal, err := json.Marshal(val)
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON value: %w", err)
		}

		return string(jVal), nil
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

//nolint:dupl //factorising would add complexity
func getLocal(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		var agent model.LocalAgent
		if err := r.db.Get(&agent, "id=(SELECT local_agent_id FROM local_accounts WHERE id=?)",
			r.transCtx.Transfer.LocalAccountID).Run(); err != nil {
			return "", err
		}

		return agent.Name, nil
	}

	var account model.RemoteAccount
	if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.RemoteAccountID).Run(); err != nil {
		return "", err
	}

	return account.Login, nil
}

//nolint:dupl //factorising would add complexity
func getRemote(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		var account model.LocalAccount
		if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.LocalAccountID).Run(); err != nil {
			return "", err
		}

		return account.Login, nil
	}

	var agent model.RemoteAgent
	if err := r.db.Get(&agent, "id=(SELECT remote_agent_id FROM remote_accounts WHERE id=?)",
		r.transCtx.Transfer.RemoteAccountID).Run(); err != nil {
		return "", err
	}

	return agent.Name, nil
}

//nolint:dupl //factorising would add complexity
func getClient(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		var account model.LocalAccount
		if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.LocalAccountID).Run(); err != nil {
			return "", err
		}

		return account.Login, nil
	}

	var account model.RemoteAccount
	if err := r.db.Get(&account, "id=?", r.transCtx.Transfer.RemoteAccountID).Run(); err != nil {
		return "", err
	}

	return account.Login, nil
}

//nolint:dupl //factorising would add complexity
func getServer(r *Runner) (string, error) {
	if r.transCtx.Transfer.IsServer() {
		var agent model.LocalAgent
		if err := r.db.Get(&agent, "id=(SELECT local_agent_id FROM local_accounts WHERE id=?)",
			r.transCtx.Transfer.LocalAccountID).Run(); err != nil {
			return "", err
		}

		return agent.Name, nil
	}

	var agent model.RemoteAgent
	if err := r.db.Get(&agent, "id=(SELECT remote_agent_id FROM remote_accounts WHERE id=?)",
		r.transCtx.Transfer.RemoteAccountID).Run(); err != nil {
		return "", err
	}

	return agent.Name, nil
}
