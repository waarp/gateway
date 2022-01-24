package tasks

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// TransferTask is a task which schedules a new transfer.
type TransferTask struct{}

//nolint:gochecknoinits // This package is designed this way
func init() {
	model.ValidTasks["TRANSFER"] = &TransferTask{}
}

// Validate checks if the tasks has all the required arguments.
func (t *TransferTask) Validate(args map[string]string) error {
	if file, ok := args["file"]; !ok || file == "" {
		return fmt.Errorf("missing transfer source file: %w", errBadTaskArguments)
	}

	if to, ok := args["to"]; !ok || to == "" {
		if from, ok := args["from"]; !ok || from == "" {
			return fmt.Errorf("missing transfer remote partner: %w", errBadTaskArguments)
		}
	} else if from, ok := args["from"]; ok && from != "" {
		return fmt.Errorf("cannot have both 'to' and 'from': %w", errBadTaskArguments)
	}

	if as, ok := args["as"]; !ok || as == "" {
		return fmt.Errorf("missing transfer account: %w", errBadTaskArguments)
	}

	if rule, ok := args["rule"]; !ok || rule == "" {
		return fmt.Errorf("missing transfer rule: %w", errBadTaskArguments)
	}

	return nil
}

func getTransferInfo(db *database.DB, args map[string]string) (file string,
	ruleID, agentID, accountID uint64, infoErr error) {
	fileName, fileOK := args["file"]
	if !fileOK || fileName == "" {
		infoErr = fmt.Errorf("missing transfer file: %w", errBadTaskArguments)

		return
	}

	agentName, agentOK := args["to"]
	if !agentOK || agentName == "" {
		agentName, agentOK = args["from"]
		if !agentOK || agentName == "" {
			infoErr = fmt.Errorf("a missing transfer remote partner: %w", errBadTaskArguments)

			return
		}
	} else if from, ok := args["from"]; ok && from != "" {
		infoErr = fmt.Errorf("cannot have both 'to' and 'from': %w", errBadTaskArguments)

		return
	}

	ruleName, ruleOK := args["rule"]
	if !ruleOK || ruleName == "" {
		infoErr = fmt.Errorf("missing transfer rule: %w", errBadTaskArguments)

		return
	}

	rule := &model.Rule{}
	if err := db.Get(rule, "name=? AND send=?", ruleName, args["to"] != "").Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve rule '%s': %w", ruleName, err)

		return
	}

	agent := &model.RemoteAgent{}
	if err := db.Get(agent, "name=?", agentName).Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve partner '%s': %w", agentName, err)

		return
	}

	accName, accOK := args["as"]
	if !accOK || accName == "" {
		infoErr = fmt.Errorf("missing transfer account: %w", errBadTaskArguments)

		return
	}

	acc := &model.RemoteAccount{}
	if err := db.Get(acc, "remote_agent_id=? AND login=?", agent.ID, accName).Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve account '%s': %w", accName, err)

		return
	}

	return fileName, rule.ID, agent.ID, acc.ID, nil
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(_ context.Context, args map[string]string,
	db *database.DB, _ *model.TransferContext) (string, error) {
	file, ruleID, agentID, accID, err := getTransferInfo(db, args)
	if err != nil {
		return err.Error(), err
	}

	trans := &model.Transfer{
		RuleID:     ruleID,
		IsServer:   false,
		AgentID:    agentID,
		AccountID:  accID,
		LocalPath:  file,
		RemotePath: file,
	}

	if err := db.Insert(trans).Run(); err != nil {
		return fmt.Sprintf("cannot create transfer of file '%s': %s", file, err), err
	}

	return "", nil
}
