package tasks

import (
	"fmt"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// TransferTask is a task which schedules a new transfer.
type TransferTask struct{}

func init() {
	RunnableTasks["TRANSFER"] = &TransferTask{}
	model.ValidTasks["TRANSFER"] = &TransferTask{}
}

// Validate checks if the tasks has all the required arguments.
func (t *TransferTask) Validate(args map[string]string) error {
	if file, ok := args["file"]; !ok || file == "" {
		return fmt.Errorf("missing transfer source file")
	}
	if to, ok := args["to"]; !ok || to == "" {
		return fmt.Errorf("missing transfer remote partner")
	}
	if as, ok := args["as"]; !ok || as == "" {
		return fmt.Errorf("missing transfer account")
	}
	if rule, ok := args["rule"]; !ok || rule == "" {
		return fmt.Errorf("missing transfer rule")
	}

	return nil
}

func getTransferInfo(db *database.Db, args map[string]interface{}) (file string,
	ruleID, agentID, accountID uint64, infoErr error) {

	fileName, fileOK := args["file"].(string)
	if !fileOK || fileName == "" {
		infoErr = fmt.Errorf("missing transfer file")
		return
	}
	file = fileName

	ruleName, ruleOK := args["rule"].(string)
	if !ruleOK || ruleName == "" {
		infoErr = fmt.Errorf("missing transfer rule")
		return
	}
	rule := &model.Rule{Name: ruleName}
	if err := db.Get(rule); err != nil {
		infoErr = fmt.Errorf("error getting rule: %s", err)
		return
	}
	ruleID = rule.ID

	agentName, agentOK := args["to"].(string)
	if !agentOK || agentName == "" {
		infoErr = fmt.Errorf("missing transfer remote partner")
		return
	}
	agent := &model.RemoteAgent{Name: agentName}
	if err := db.Get(agent); err != nil {
		infoErr = fmt.Errorf("error getting partner: %s", err)
		return
	}
	agentID = agent.ID

	accName, accOK := args["as"].(string)
	if !accOK || accName == "" {
		infoErr = fmt.Errorf("missing transfer account")
		return
	}
	acc := &model.RemoteAccount{RemoteAgentID: agent.ID, Login: accName}
	if err := db.Get(acc); err != nil {
		infoErr = fmt.Errorf("error getting account: %s", err)
		return
	}
	accountID = acc.ID
	return file, ruleID, agentID, accountID, nil
}

// Run executes the task by scheduling a new transfer with the given parameters.
func (t *TransferTask) Run(args map[string]interface{}, processor *Processor) (string, error) {
	file, ruleID, agentID, accID, err := getTransferInfo(processor.Db, args)
	if err != nil {
		return err.Error(), err
	}

	trans := &model.Transfer{
		RuleID:     ruleID,
		IsServer:   false,
		AgentID:    agentID,
		AccountID:  accID,
		SourcePath: file,
		DestPath:   filepath.Base(file),
	}
	if err := processor.Db.Create(trans); err != nil {
		return err.Error(), err
	}
	return "", nil
}
