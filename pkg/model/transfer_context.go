package model

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
)

var errDatabase = types.NewTransferError(types.TeInternal, "database error")

// TransferContext regroups all the information necessary for an outgoing
// transfer.
type TransferContext struct {
	Transfer  *Transfer
	Rule      *Rule
	PreTasks  Tasks
	PostTasks Tasks
	ErrTasks  Tasks

	RemoteAgent        *RemoteAgent
	RemoteAgentCryptos []Crypto

	RemoteAccount        *RemoteAccount
	RemoteAccountCryptos []Crypto

	LocalAgent        *LocalAgent
	LocalAgentCryptos []Crypto

	LocalAccount        *LocalAccount
	LocalAccountCryptos []Crypto

	Paths *conf.PathsConfig
}

// GetTransferInfo retrieves all the information regarding the given transfer
// from the database, and returns it wrapped in a TransferInfo instance.
// An error is returned a problem occurs while accessing the database.
func GetTransferInfo(db *database.DB, logger *log.Logger, trans *Transfer,
) (*TransferContext, *types.TransferError) {
	transCtx := &TransferContext{
		Transfer:      trans,
		Paths:         &db.Conf.Paths,
		Rule:          &Rule{},
		RemoteAgent:   &RemoteAgent{},
		RemoteAccount: &RemoteAccount{},
		LocalAgent:    &LocalAgent{},
		LocalAccount:  &LocalAccount{},
	}

	if err := db.Get(transCtx.Rule, "id=?", trans.RuleID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer rule: %s", err)
		return nil, errDatabase
	}
	if err := db.Select(&transCtx.PreTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainPre).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer pre-tasks: %s", err)
		return nil, errDatabase
	}
	if err := db.Select(&transCtx.PostTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainPost).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer post-tasks: %s", err)
		return nil, errDatabase
	}
	if err := db.Select(&transCtx.ErrTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainError).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer error-tasks: %s", err)
		return nil, errDatabase
	}

	var err error
	if trans.IsServer {
		if err := db.Get(transCtx.LocalAgent, "id=?", trans.AgentID).Run(); err != nil {
			logger.Errorf("Failed to retrieve transfer server: %s", err)
			return nil, errDatabase
		}
		if transCtx.LocalAgentCryptos, err = transCtx.LocalAgent.GetCryptos(db); err != nil {
			logger.Errorf("Failed to retrieve server certificates: %s", err)
			return nil, errDatabase
		}
		if err := db.Get(transCtx.LocalAccount, "id=?", trans.AccountID).Run(); err != nil {
			logger.Errorf("Failed to retrieve transfer local account: %s", err)
			return nil, errDatabase
		}
		if transCtx.LocalAccountCryptos, err = transCtx.LocalAccount.GetCryptos(db); err != nil {
			logger.Errorf("Failed to retrieve local account certificates: %s", err)
			return nil, errDatabase
		}

		return transCtx, nil
	}

	if err := db.Get(transCtx.RemoteAgent, "id=?", trans.AgentID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer partner: %s", err)
		return nil, errDatabase
	}
	if transCtx.RemoteAgentCryptos, err = transCtx.RemoteAgent.GetCryptos(db); err != nil {
		logger.Errorf("Failed to retrieve partner certificates: %s", err)
		return nil, errDatabase
	}
	if err := db.Get(transCtx.RemoteAccount, "id=?", trans.AccountID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer remote account: %s", err)
		return nil, errDatabase
	}
	if transCtx.RemoteAccountCryptos, err = transCtx.RemoteAccount.GetCryptos(db); err != nil {
		logger.Errorf("Failed to retrieve remote account certificates: %s", err)
		return nil, errDatabase
	}

	return transCtx, nil
}
