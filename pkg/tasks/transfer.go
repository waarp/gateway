package tasks

import (
	"context"
	"database/sql"
	"fmt"

	"code.waarp.fr/lib/log"

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
	ruleID, accountID int64, isSend bool, infoErr error,
) {
	fileName, fileOK := args["file"]
	if !fileOK || fileName == "" {
		infoErr = fmt.Errorf("missing transfer file: %w", errBadTaskArguments)

		return "", 0, 0, false, infoErr
	}

	agentName, agentOK := args["to"]
	if !agentOK || agentName == "" {
		agentName, agentOK = args["from"]
		if !agentOK || agentName == "" {
			infoErr = fmt.Errorf("a missing transfer remote partner: %w", errBadTaskArguments)

			return "", 0, 0, false, infoErr
		}
	} else if from, ok := args["from"]; ok && from != "" {
		infoErr = fmt.Errorf("cannot have both 'to' and 'from': %w", errBadTaskArguments)

		return "", 0, 0, false, infoErr
	}

	ruleName, ruleOK := args["rule"]
	if !ruleOK || ruleName == "" {
		infoErr = fmt.Errorf("missing transfer rule: %w", errBadTaskArguments)

		return "", 0, 0, false, infoErr
	}

	rule := &model.Rule{}
	if err := db.Get(rule, "name=? AND is_send=?", ruleName, args["to"] != "").Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve rule '%s': %w", ruleName, err)

		return "", 0, 0, false, infoErr
	}

	agent := &model.RemoteAgent{}
	if err := db.Get(agent, "name=?", agentName).Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve partner '%s': %w", agentName, err)

		return "", 0, 0, false, infoErr
	}

	accName, accOK := args["as"]
	if !accOK || accName == "" {
		infoErr = fmt.Errorf("missing transfer account: %w", errBadTaskArguments)

		return "", 0, 0, false, infoErr
	}

	acc := &model.RemoteAccount{}
	if err := db.Get(acc, "remote_agent_id=? AND login=?", agent.ID, accName).Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve account '%s': %w", accName, err)

		return "", 0, 0, false, infoErr
	}

	return fileName, rule.ID, acc.ID, rule.IsSend, nil
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(_ context.Context, args map[string]string, db *database.DB,
	logger *log.Logger, _ *model.TransferContext,
) error {
	file, ruleID, accID, isSend, err := getTransferInfo(db, args)
	if err != nil {
		return err
	}

	trans := &model.Transfer{
		RuleID:          ruleID,
		RemoteAccountID: sql.NullInt64{Int64: accID, Valid: true},
		SrcFilename:     file,
	}

	if err := db.Insert(trans).Run(); err != nil {
		return err
	}

	recipient := fmt.Sprintf("from %q", args["from"])
	if isSend {
		recipient = fmt.Sprintf("to %q", args["to"])
	}

	logger.Debug("Programmed new transfer n°%d of file %q, %s as %q using rule %q",
		trans.ID, file, recipient, args["as"], args["rule"])

	return nil
}
