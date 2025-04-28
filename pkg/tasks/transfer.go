package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
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
		return fmt.Errorf("missing transfer source file: %w", ErrBadTaskArguments)
	}

	if to, ok := args["to"]; !ok || to == "" {
		if from, ok := args["from"]; !ok || from == "" {
			return fmt.Errorf("missing transfer remote partner: %w", ErrBadTaskArguments)
		}
	} else if from, ok := args["from"]; ok && from != "" {
		return fmt.Errorf("cannot have both 'to' and 'from': %w", ErrBadTaskArguments)
	}

	if as, ok := args["as"]; !ok || as == "" {
		return fmt.Errorf("missing transfer account: %w", ErrBadTaskArguments)
	}

	if rule, ok := args["rule"]; !ok || rule == "" {
		return fmt.Errorf("missing transfer rule: %w", ErrBadTaskArguments)
	}

	return nil
}

//nolint:funlen //function is perfectly readable despite being long
func getTransferInfo(db *database.DB, args map[string]string) (file string,
	ruleID, clientID, accountID int64, isSend bool, infoErr error,
) {
	fileName, fileOK := args["file"]
	if !fileOK || fileName == "" {
		infoErr = fmt.Errorf("missing transfer file: %w", ErrBadTaskArguments)

		return "", 0, 0, 0, false, infoErr
	}

	// client name is optional
	clientName := args["using"]

	agentName, agentOK := args["to"]
	if !agentOK || agentName == "" {
		agentName, agentOK = args["from"]
		if !agentOK || agentName == "" {
			return "", 0, 0, 0, false,
				fmt.Errorf("missing transfer remote partner: %w", ErrBadTaskArguments)
		}
	} else if from, ok := args["from"]; ok && from != "" {
		return "", 0, 0, 0, false,
			fmt.Errorf("cannot have both 'to' and 'from': %w", ErrBadTaskArguments)
	}

	ruleName, ruleOK := args["rule"]
	if !ruleOK || ruleName == "" {
		return "", 0, 0, 0, false, fmt.Errorf("missing transfer rule: %w", ErrBadTaskArguments)
	}

	rule := &model.Rule{}
	if err := db.Get(rule, "name=? AND is_send=?", ruleName, args["to"] != "").Run(); err != nil {
		return "", 0, 0, 0, false, fmt.Errorf("failed to retrieve rule %q: %w", ruleName, err)
	}

	agent := &model.RemoteAgent{}
	if err := db.Get(agent, "owner=? AND name=?", conf.GlobalConfig.GatewayName,
		agentName).Run(); err != nil {
		return "", 0, 0, 0, false, fmt.Errorf("failed to retrieve partner %q: %w", agentName, err)
	}

	accName, accOK := args["as"]
	if !accOK || accName == "" {
		return "", 0, 0, 0, false, fmt.Errorf("missing transfer account: %w", ErrBadTaskArguments)
	}

	acc := &model.RemoteAccount{}
	if err := db.Get(acc, "remote_agent_id=? AND login=?", agent.ID, accName).Run(); err != nil {
		return "", 0, 0, 0, false, fmt.Errorf("failed to retrieve account %q: %w", accName, err)
	}

	client := &model.Client{}

	if clientName == "" {
		var err error
		if client, err = model.GetDefaultTransferClient(db, acc.ID); err != nil {
			infoErr = fmt.Errorf("failed to retrieve default transfer client: %w", err)

			return "", 0, 0, 0, false, infoErr
		}
	} else {
		if err := db.Get(client, "owner=? AND name=?", conf.GlobalConfig.GatewayName,
			clientName).Run(); err != nil {
			infoErr = fmt.Errorf("failed to retrieve client %q: %w", clientName, err)

			return "", 0, 0, 0, false, infoErr
		}
	}

	return fileName, rule.ID, client.ID, acc.ID, rule.IsSend, nil
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(_ context.Context, args map[string]string,
	db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) error {
	file, ruleID, cliID, accID, isSend, infoErr := getTransferInfo(db, args)
	if infoErr != nil {
		return infoErr
	}

	transferInfo := map[string]any{}
	if args["copyInfo"] == "true" {
		transferInfo = transCtx.TransInfo
	}

	newInfoStr, hasNewInfo := args["info"]
	if hasNewInfo {
		if err := json.Unmarshal([]byte(newInfoStr), &transferInfo); err != nil {
			return fmt.Errorf("failed to parse the new transfer info: %w", err)
		}
	}

	trans := &model.Transfer{
		RuleID:          ruleID,
		ClientID:        utils.NewNullInt64(cliID),
		RemoteAccountID: utils.NewNullInt64(accID),
		SrcFilename:     file,
		DestFilename:    filepath.Base(file),
	}

	if err := db.Transaction(func(ses *database.Session) error {
		if err := ses.Insert(trans).Run(); err != nil {
			return fmt.Errorf("failed to insert transfer: %w", err)
		}

		if err := trans.SetTransferInfo(ses, transferInfo); err != nil {
			return fmt.Errorf("failed to set transfer info: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create transfer: %w", err)
	}

	recipient := fmt.Sprintf("from %q", args["from"])
	if isSend {
		recipient = fmt.Sprintf("to %q", args["to"])
	}

	logger.Debug("Programmed new transfer n°%d of file %q, %s as %q using rule %q",
		trans.ID, file, recipient, args["as"], args["rule"])

	return nil
}
