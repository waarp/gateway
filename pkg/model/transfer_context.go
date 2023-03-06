package model

import (
	"io/fs"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

var errDatabase = types.NewTransferError(types.TeInternal, "database error")

// TransferContext regroups all the information necessary for an outgoing
// transfer.
type TransferContext struct {
	Transfer  *Transfer
	TransInfo map[string]interface{}
	// FileInfo  map[string]interface{}
	Rule      *Rule
	PreTasks  Tasks
	PostTasks Tasks
	ErrTasks  Tasks

	Client *Client

	RemoteAgent        *RemoteAgent
	RemoteAgentCryptos Cryptos

	RemoteAccount        *RemoteAccount
	RemoteAccountCryptos Cryptos

	LocalAgent        *LocalAgent
	LocalAgentCryptos Cryptos

	LocalAccount        *LocalAccount
	LocalAccountCryptos Cryptos

	Paths *conf.PathsConfig
	FS    fs.FS
}

// GetTransferContext retrieves all the information regarding the given transfer
// from the database, and returns it wrapped in a TransferInfo instance.
// An error is returned a problem occurs while accessing the database.
func GetTransferContext(db *database.DB, logger *log.Logger, trans *Transfer,
) (*TransferContext, *types.TransferError) {
	transCtx := &TransferContext{
		Transfer:      trans,
		TransInfo:     map[string]interface{}{},
		Paths:         &conf.GlobalConfig.Paths,
		Client:        &Client{},
		Rule:          &Rule{},
		RemoteAgent:   &RemoteAgent{},
		RemoteAccount: &RemoteAccount{},
		LocalAgent:    &LocalAgent{},
		LocalAccount:  &LocalAccount{},
	}

	if err := db.Get(transCtx.Rule, "id=?", trans.RuleID).Run(); err != nil {
		logger.Error("Failed to retrieve transfer rule: %v", err)

		return nil, errDatabase
	}

	if err := db.Select(&transCtx.PreTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainPre).Run(); err != nil {
		logger.Error("Failed to retrieve transfer pre-tasks: %v", err)

		return nil, errDatabase
	}

	if err := db.Select(&transCtx.PostTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainPost).Run(); err != nil {
		logger.Error("Failed to retrieve transfer post-tasks: %v", err)

		return nil, errDatabase
	}

	if err := db.Select(&transCtx.ErrTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainError).Run(); err != nil {
		logger.Error("Failed to retrieve transfer error-tasks: %v", err)

		return nil, errDatabase
	}

	var err database.Error
	if transCtx.TransInfo, err = transCtx.Transfer.GetTransferInfo(db); err != nil {
		logger.Error("Failed to retrieve the transfer info: %v", err)

		return nil, errDatabase
	}

	return makeAgentContext(db, logger, transCtx)
}

func makeAgentContext(db *database.DB, logger *log.Logger, transCtx *TransferContext,
) (*TransferContext, *types.TransferError) {
	if transCtx.Transfer.IsServer() {
		return makeLocalAgentContext(db, logger, transCtx)
	}

	return makeRemoteAgentContext(db, logger, transCtx)
}

//nolint:dupl //factorizing would add complexity
func makeLocalAgentContext(db *database.DB, logger *log.Logger, transCtx *TransferContext,
) (*TransferContext, *types.TransferError) {
	if err := db.Get(transCtx.LocalAccount, "id=?", transCtx.Transfer.LocalAccountID).Run(); err != nil {
		logger.Error("Failed to retrieve transfer local account: %s", err)

		return nil, errDatabase
	}

	if err := db.Get(transCtx.LocalAgent, "id=?", transCtx.LocalAccount.LocalAgentID).Run(); err != nil {
		logger.Error("Failed to retrieve transfer server: %s", err)

		return nil, errDatabase
	}

	var err error
	if transCtx.LocalAccountCryptos, err = transCtx.LocalAccount.GetCryptos(db); err != nil {
		logger.Error("Failed to retrieve local account certificates: %s", err)

		return nil, errDatabase
	}

	if transCtx.LocalAgentCryptos, err = transCtx.LocalAgent.GetCryptos(db); err != nil {
		logger.Error("Failed to retrieve server certificates: %s", err)

		return nil, errDatabase
	}

	return transCtx, nil
}

//nolint:dupl //factorizing would add complexity
func makeRemoteAgentContext(db *database.DB, logger *log.Logger, transCtx *TransferContext,
) (*TransferContext, *types.TransferError) {
	if err := db.Get(transCtx.RemoteAccount, "id=?", transCtx.Transfer.RemoteAccountID).Run(); err != nil {
		logger.Error("Failed to retrieve transfer remote account: %s", err)

		return nil, errDatabase
	}

	if err := db.Get(transCtx.RemoteAgent, "id=?", transCtx.RemoteAccount.RemoteAgentID).Run(); err != nil {
		logger.Error("Failed to retrieve transfer partner: %s", err)

		return nil, errDatabase
	}

	if err := db.Get(transCtx.Client, "id=?", transCtx.Transfer.ClientID).Run(); err != nil {
		logger.Error("Failed to retrieve the transfer client: %v", err)

		return nil, errDatabase
	}

	var err error
	if transCtx.RemoteAccountCryptos, err = transCtx.RemoteAccount.GetCryptos(db); err != nil {
		logger.Error("Failed to retrieve remote account certificates: %s", err)

		return nil, errDatabase
	}

	if transCtx.RemoteAgentCryptos, err = transCtx.RemoteAgent.GetCryptos(db); err != nil {
		logger.Error("Failed to retrieve partner certificates: %s", err)

		return nil, errDatabase
	}

	return transCtx, nil
}
