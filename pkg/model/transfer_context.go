package model

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// TransferContext regroups all the information necessary for an outgoing
// transfer.
type TransferContext struct {
	Transfer  *Transfer
	TransInfo map[string]any
	// FileInfo  map[string]any
	Rule      *Rule
	PreTasks  Tasks
	PostTasks Tasks
	ErrTasks  Tasks

	Client *Client

	RemoteAgent      *RemoteAgent
	RemoteAgentCreds Credentials

	RemoteAccount      *RemoteAccount
	RemoteAccountCreds Credentials

	LocalAgent      *LocalAgent
	LocalAgentCreds Credentials

	LocalAccount      *LocalAccount
	LocalAccountCreds Credentials

	Paths *conf.PathsConfig
}

// GetTransferContext retrieves all the information regarding the given transfer
// from the database, and returns it wrapped in a TransferInfo instance.
// An error is returned a problem occurs while accessing the database.
func GetTransferContext(db *database.DB, logger *log.Logger, trans *Transfer,
) (*TransferContext, error) {
	transCtx := &TransferContext{
		Transfer:      trans,
		TransInfo:     map[string]any{},
		Paths:         &conf.GlobalConfig.Paths,
		Client:        &Client{},
		Rule:          &Rule{},
		RemoteAgent:   &RemoteAgent{},
		RemoteAccount: &RemoteAccount{},
		LocalAgent:    &LocalAgent{},
		LocalAccount:  &LocalAccount{},
	}

	if err := db.Get(transCtx.Rule, "id=?", trans.RuleID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer rule: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer rule: %w", err)
	}

	if err := db.Select(&transCtx.PreTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainPre).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer pre-tasks: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer pre-tasks: %w", err)
	}

	if err := db.Select(&transCtx.PostTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainPost).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer post-tasks: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer post-tasks: %w", err)
	}

	if err := db.Select(&transCtx.ErrTasks).Where("rule_id=? AND chain=?",
		trans.RuleID, ChainError).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer error-tasks: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer error-tasks: %w", err)
	}

	var err error
	if transCtx.TransInfo, err = transCtx.Transfer.GetTransferInfo(db); err != nil {
		logger.Errorf("Failed to retrieve the transfer info: %v", err)

		return nil, fmt.Errorf("failed to retrieve the transfer info: %w", err)
	}

	return makeAgentContext(db, logger, transCtx)
}

func makeAgentContext(db *database.DB, logger *log.Logger, transCtx *TransferContext,
) (*TransferContext, error) {
	if transCtx.Transfer.IsServer() {
		return makeLocalAgentContext(db, logger, transCtx)
	}

	return makeRemoteAgentContext(db, logger, transCtx)
}

//nolint:dupl //factorizing would add complexity
func makeLocalAgentContext(db *database.DB, logger *log.Logger, transCtx *TransferContext,
) (*TransferContext, error) {
	if err := db.Get(transCtx.LocalAccount, "id=?", transCtx.Transfer.LocalAccountID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer local account: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer local account: %w", err)
	}

	if err := db.Get(transCtx.LocalAgent, "id=?", transCtx.LocalAccount.LocalAgentID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer server: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer server: %w", err)
	}

	var err error
	if transCtx.LocalAccountCreds, err = transCtx.LocalAccount.GetCredentials(db); err != nil {
		logger.Errorf("Failed to retrieve local account auth methods: %v", err)

		return nil, fmt.Errorf("failed to retrieve local account certificates: %w", err)
	}

	if transCtx.LocalAgentCreds, err = transCtx.LocalAgent.GetCredentials(db); err != nil {
		logger.Errorf("Failed to retrieve server auth methods: %v", err)

		return nil, fmt.Errorf("failed to retrieve server certificates: %w", err)
	}

	return transCtx, nil
}

//nolint:dupl //factorizing would add complexity
func makeRemoteAgentContext(db *database.DB, logger *log.Logger, transCtx *TransferContext,
) (*TransferContext, error) {
	if err := db.Get(transCtx.RemoteAccount, "id=?", transCtx.Transfer.RemoteAccountID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer remote account: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer remote account: %w", err)
	}

	if err := db.Get(transCtx.RemoteAgent, "id=?", transCtx.RemoteAccount.RemoteAgentID).Run(); err != nil {
		logger.Errorf("Failed to retrieve transfer partner: %v", err)

		return nil, fmt.Errorf("failed to retrieve transfer partner: %w", err)
	}

	if err := db.Get(transCtx.Client, "id=?", transCtx.Transfer.ClientID).Run(); err != nil {
		logger.Errorf("Failed to retrieve the transfer client: %v", err)

		return nil, fmt.Errorf("failed to retrieve the transfer client: %w", err)
	}

	var err error
	if transCtx.RemoteAccountCreds, err = transCtx.RemoteAccount.GetCredentials(db); err != nil {
		logger.Errorf("Failed to retrieve remote account auth methods: %v", err)

		return nil, fmt.Errorf("failed to retrieve remote account certificates: %w", err)
	}

	if transCtx.RemoteAgentCreds, err = transCtx.RemoteAgent.GetCredentials(db); err != nil {
		logger.Errorf("Failed to retrieve partner auth methods: %v", err)

		return nil, fmt.Errorf("failed to retrieve partner certificates: %w", err)
	}

	return transCtx, nil
}
