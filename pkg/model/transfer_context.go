package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

// TransferContext regroups all the information necessary for an outgoing
// transfer.
type TransferContext struct {
	Transfer  *Transfer
	Rule      *Rule
	PreTasks  Tasks
	PostTasks Tasks
	ErrTasks  Tasks

	RemoteAgent      *RemoteAgent
	RemoteAgentCerts []Cert

	RemoteAccount      *RemoteAccount
	RemoteAccountCerts []Cert

	LocalAgent      *LocalAgent
	LocalAgentCerts []Cert

	LocalAccount      *LocalAccount
	LocalAccountCerts []Cert

	Paths *conf.PathsConfig
}

// NewOutTransferInfo retrieves all the information regarding the given transfer
// from the database, and returns it wrapped in a TransferInfo instance.
// An error is returned a problem occurs while accessing the database.
func NewOutTransferInfo(db *database.DB, trans *Transfer, paths *conf.PathsConfig) (*TransferContext, error) {
	info := &TransferContext{
		Transfer: trans,
		Paths:    paths,
	}

	if err := db.Get(info.Rule, "id=?", trans.RuleID).Run(); err != nil {
		return nil, err
	}

	if trans.IsServer {
		if err := db.Get(info.LocalAgent, "id=?", trans.AgentID).Run(); err != nil {
			return nil, err
		}
		if err := db.Get(info.LocalAccount, "id=?", trans.AccountID).Run(); err != nil {
			return nil, err
		}

		var err error
		info.LocalAgentCerts, err = info.LocalAgent.GetCerts(db)
		if err != nil {
			return nil, err
		}
		info.LocalAccountCerts, err = info.LocalAccount.GetCerts(db)
		if err != nil {
			return nil, err
		}

		return info, nil
	}

	if err := db.Get(info.RemoteAgent, "id=?", trans.AgentID).Run(); err != nil {
		return nil, err
	}
	if err := db.Get(info.RemoteAccount, "id=?", trans.AccountID).Run(); err != nil {
		return nil, err
	}

	var err error
	info.RemoteAgentCerts, err = info.RemoteAgent.GetCerts(db)
	if err != nil {
		return nil, err
	}
	info.RemoteAccountCerts, err = info.RemoteAccount.GetCerts(db)
	if err != nil {
		return nil, err
	}

	return info, nil
}
