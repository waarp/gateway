package model

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

// OutTransferInfo regroups all the information necessary for an outgoing
// transfer.
type OutTransferInfo struct {
	Transfer Transfer
	Rule     Rule
	Agent    RemoteAgent
	Account  RemoteAccount
	Certs    []Cert
}

// NewOutTransferInfo retrieves all the information regarding the given transfer
// from the database, and returns it wrapped in a `OutTransferInfo` instance.
// An error is returned a problem occurs while accessing the database.
func NewOutTransferInfo(db *database.Db, trans Transfer) (*OutTransferInfo, error) {

	remote := RemoteAgent{ID: trans.AgentID}
	if err := db.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the partner n°%v does not exist", trans.AgentID)
		}
		return nil, err
	}
	certs, err := remote.GetCerts(db)
	if err != nil || len(certs) == 0 {
		if len(certs) == 0 {
			return nil, fmt.Errorf("no certificates found for agent n°%v", remote.ID)
		}
		return nil, err
	}
	account := RemoteAccount{ID: trans.AccountID}
	if err := db.Get(&account); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the account n°%v does not exist", account.ID)
		}
		return nil, err
	}
	if account.RemoteAgentID != remote.ID {
		return nil, fmt.Errorf("the account n°%v does not belong to agent n°%v",
			account.ID, remote.ID)
	}

	rule := Rule{ID: trans.RuleID}
	if err := db.Get(&rule); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the rule n°%v does not exist", rule.ID)
		}
		return nil, err
	}

	return &OutTransferInfo{
		Transfer: trans,
		Agent:    remote,
		Account:  account,
		Certs:    certs,
		Rule:     rule,
	}, nil
}

// InTransferInfo regroups all the information necessary for an incoming
// transfer.
type InTransferInfo struct {
	Transfer Transfer
	Rule     Rule
	Agent    LocalAgent
	Account  LocalAccount
}

// NewInTransferInfo retrieves all the information regarding the given transfer
// from the database, and returns it wrapped in a `InTransferInfo` instance.
// An error is returned a problem occurs while accessing the database.
func NewInTransferInfo(db *database.Db, trans Transfer) (*InTransferInfo, error) {

	server := LocalAgent{ID: trans.AgentID}
	if err := db.Get(&server); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the server n°%v does not exist", trans.AgentID)
		}
		return nil, err
	}
	account := LocalAccount{ID: trans.AccountID}
	if err := db.Get(&account); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the account n°%v does not exist", account.ID)
		}
		return nil, err
	}
	if account.LocalAgentID != server.ID {
		return nil, fmt.Errorf("the account n°%v does not belong to agent n°%v",
			account.ID, server.ID)
	}

	rule := Rule{ID: trans.RuleID}
	if err := db.Get(&rule); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the rule n°%v does not exist", rule.ID)
		}
		return nil, err
	}

	return &InTransferInfo{
		Transfer: trans,
		Agent:    server,
		Account:  account,
		Rule:     rule,
	}, nil
}
