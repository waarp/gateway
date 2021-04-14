package model

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

// OutTransferInfo regroups all the information necessary for an outgoing
// transfer.
type OutTransferInfo struct {
	Transfer      *Transfer
	Rule          *Rule
	Agent         *RemoteAgent
	Account       *RemoteAccount
	ServerCryptos Cryptos
	ClientCryptos Cryptos
}

// NewOutTransferInfo retrieves all the information regarding the given transfer
// from the database, and returns it wrapped in a `OutTransferInfo` instance.
// An error is returned a problem occurs while accessing the database.
func NewOutTransferInfo(db *database.DB, trans *Transfer) (*OutTransferInfo, error) {

	var remote RemoteAgent
	if err := db.Get(&remote, "id=?", trans.AgentID).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("the partner n°%v does not exist", trans.AgentID)
		}
		return nil, err
	}
	serverCerts, err := remote.GetCryptos(db)
	if err != nil {
		return nil, err
	}
	var account RemoteAccount
	if err := db.Get(&account, "id=?", trans.AccountID).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("the account n°%v does not exist", account.ID)
		}
		return nil, err
	}
	if account.RemoteAgentID != remote.ID {
		return nil, fmt.Errorf("the account n°%v does not belong to agent n°%v",
			account.ID, remote.ID)
	}

	var rule Rule
	if err := db.Get(&rule, "id=?", trans.RuleID).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, fmt.Errorf("the rule n°%v does not exist", rule.ID)
		}
		return nil, err
	}
	clientCerts, err := account.GetCryptos(db)
	if err != nil {
		return nil, err
	}

	return &OutTransferInfo{
		Transfer:      trans,
		Agent:         &remote,
		Account:       &account,
		Rule:          &rule,
		ServerCryptos: serverCerts,
		ClientCryptos: clientCerts,
	}, nil
}
