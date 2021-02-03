package tasks

import (
	"context"
	"fmt"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// TransferTask is a task which schedules a new transfer.
type TransferTask struct{}

func init() {
	model.ValidTasks["TRANSFER"] = &TransferTask{}
}

// Validate checks if the tasks has all the required arguments.
func (t *TransferTask) Validate(args map[string]string) error {
	if file, ok := args["file"]; !ok || file == "" {
		return fmt.Errorf("missing transfer source file")
	}
	if to, ok := args["to"]; !ok || to == "" {
		if from, ok := args["from"]; !ok || from == "" {
			return fmt.Errorf("b missing transfer remote partner")
		}
	} else if from, ok := args["from"]; ok && from != "" {
		return fmt.Errorf("cannot have both 'to' and 'from'")
	}
	if as, ok := args["as"]; !ok || as == "" {
		return fmt.Errorf("missing transfer account")
	}
	if rule, ok := args["rule"]; !ok || rule == "" {
		return fmt.Errorf("missing transfer rule")
	}

	return nil
}

func getTransferInfo(db *database.DB, args map[string]string) (file string,
	ruleID, agentID, accountID uint64, infoErr error) {

	fileName, fileOK := args["file"]
	if !fileOK || fileName == "" {
		infoErr = fmt.Errorf("missing transfer file")
		return
	}
	file = fileName

	isSend := true
	agentName, agentOK := args["to"]
	if !agentOK || agentName == "" {
		isSend = false
		agentName, agentOK = args["from"]
		if !agentOK || agentName == "" {
			infoErr = fmt.Errorf("a missing transfer remote partner")
			return
		}
	} else if from, ok := args["from"]; ok && from != "" {
		infoErr = fmt.Errorf("cannot have both 'to' and 'from'")
		return
	}

	ruleName, ruleOK := args["rule"]
	if !ruleOK || ruleName == "" {
		infoErr = fmt.Errorf("missing transfer rule")
		return
	}
	rule := &model.Rule{}
	if err := db.Get(rule, "name=? AND send=?", ruleName, isSend).Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve rule '%s': %s", ruleName, err)
		return
	}
	ruleID = rule.ID

	agent := &model.RemoteAgent{}
	if err := db.Get(agent, "name=?", agentName).Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve partner '%s': %s", agentName, err)
		return
	}
	agentID = agent.ID

	accName, accOK := args["as"]
	if !accOK || accName == "" {
		infoErr = fmt.Errorf("missing transfer account")
		return
	}
	acc := &model.RemoteAccount{}
	if err := db.Get(acc, "remote_agent_id=? AND login=?", agent.ID, accName).Run(); err != nil {
		infoErr = fmt.Errorf("failed to retrieve account '%s': %s", accName, err)
		return
	}
	accountID = acc.ID
	return file, ruleID, agentID, accountID, nil
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(args map[string]string, db *database.DB,
	_ *model.TransferContext, _ context.Context) (string, error) {
	file, ruleID, agentID, accID, err := getTransferInfo(db, args)
	if err != nil {
		return err.Error(), err
	}

	trans := &model.Transfer{
		RuleID:       ruleID,
		IsServer:     false,
		AgentID:      agentID,
		AccountID:    accID,
		TrueFilepath: file,
		SourceFile:   filepath.Base(file),
		DestFile:     filepath.Base(file),
	}
	if err := db.Insert(trans).Run(); err != nil {
		return fmt.Sprintf("cannot create transfer of file '%s': %s", file, err), err
	}
	return "", nil
}
