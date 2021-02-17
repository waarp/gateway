package model

import (
	"fmt"
	"path"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/config"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"
)

func init() {
	database.Tables = append(database.Tables, &Transfer{})
}

// Transfer represents one record of the 'transfers' table.
type Transfer struct {
	ID               uint64               `xorm:"pk autoincr <- 'id'"`
	Owner            string               `xorm:"notnull 'owner'"`
	RemoteTransferID string               `xorm:"unique(transRemID) 'remote_transfer_id'"`
	IsServer         bool                 `xorm:"notnull 'is_server'"`
	RuleID           uint64               `xorm:"notnull 'rule_id'"`
	AgentID          uint64               `xorm:"notnull 'agent_id'"`
	AccountID        uint64               `xorm:"notnull unique(transRemID) 'account_id'"`
	LocalPath        string               `xorm:"notnull 'local_path'"`
	RemotePath       string               `xorm:"notnull 'remote_path'"`
	Start            time.Time            `xorm:"notnull timestampz 'start'"`
	Status           types.TransferStatus `xorm:"notnull 'status'"`
	Step             types.TransferStep   `xorm:"notnull varchar(50) 'step'"`
	Progress         uint64               `xorm:"notnull 'progression'"`
	TaskNumber       uint64               `xorm:"notnull 'task_number'"`
	Error            types.TransferError  `xorm:"extends"`
}

// TableName returns the name of the transfers table.
func (*Transfer) TableName() string {
	return "transfers"
}

// Appellation returns the name of 1 element of the transfers table.
func (*Transfer) Appellation() string {
	return "transfer"
}

// GetID returns the transfer's ID
func (t *Transfer) GetID() uint64 {
	return t.ID
}

// SetTransferInfo replaces all the TransferInfo in the database of the the given transfer
// by those given in the map parameter.
func (t *Transfer) SetTransferInfo(db database.Access, info map[string]interface{}) error {
	if err := db.DeleteAll(&TransferInfo{TransferID: t.ID}).Run(); err != nil {
		return err
	}
	for name, val := range info {
		i := &TransferInfo{TransferID: t.ID, Name: name, Value: fmt.Sprint(val)}
		if err := db.Insert(i).Run(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transfer) validateClientTransfer(db database.ReadAccess) database.Error {
	remote := &RemoteAgent{}
	if err := db.Get(remote, "id=?", t.AgentID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationError("the partner %d does not exist", t.AgentID)
		}
		return err
	}
	n, err := db.Count(&RemoteAccount{}).Where("id=? AND remote_agent_id=?",
		t.AccountID, t.AgentID).Run()
	if err != nil {
		return err
	} else if n == 0 {
		return database.NewValidationError("the agent %d does not have an account %d",
			t.AgentID, t.AccountID)
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.NewValidationError("Rule %d is not authorized for this transfer",
			t.RuleID)
	}

	protoConf, err1 := config.GetProtoConfig(remote.Protocol, remote.ProtoConfig)
	if err1 != nil {
		return database.NewValidationError("invalid partner protocol configuration: %s", err1)
	}
	if protoConf.CertRequired() {
		n, err := db.Count(&Cert{}).Where("owner_type=? AND owner_id=?",
			remote.TableName(), remote.ID).Run()
		if err != nil {
			return err
		}
		if n == 0 {
			return database.NewValidationError(
				"the %s partner is missing a certificate when it was required",
				remote.Protocol)
		}
	}
	return nil
}

func (t *Transfer) validateServerTransfer(db database.ReadAccess) database.Error {
	n, err := db.Count(&LocalAgent{}).Where("id=?", t.AgentID).Run()
	if err != nil {
		return err
	}
	if n == 0 {
		return database.NewValidationError("the server %d does not exist", t.AgentID)
	}

	n, err = db.Count(&LocalAccount{}).Where("id=? AND local_agent_id=?",
		t.AccountID, t.AgentID).Run()
	if err != nil {
		return err
	}
	if n == 0 {
		return database.NewValidationError("the server %d does not have an account %d",
			t.AgentID, t.AccountID)
	}

	// Check for rule access
	if auth, err := IsRuleAuthorized(db, t); err != nil {
		return err
	} else if !auth {
		return database.NewValidationError("Rule %d is not authorized for this transfer", t.RuleID)
	}
	return nil
}

// BeforeWrite checks if the new `Transfer` entry is valid and can be
// inserted in the database.
//nolint:funlen
func (t *Transfer) BeforeWrite(db database.ReadAccess) database.Error {
	t.Owner = database.Owner

	if t.RuleID == 0 {
		return database.NewValidationError("the transfer's rule ID cannot be empty")
	}
	if t.AgentID == 0 {
		return database.NewValidationError("the transfer's remote ID cannot be empty")
	}
	if t.AccountID == 0 {
		return database.NewValidationError("the transfer's account ID cannot be empty")
	}
	if t.LocalPath == "" {
		return database.NewValidationError("the local filepath is missing")
	}

	if t.Start.IsZero() {
		t.Start = time.Now()
	}

	if t.Status == "" {
		t.Status = types.StatusPlanned
	}
	if !types.ValidateStatusForTransfer(t.Status) {
		return database.NewValidationError("'%s' is not a valid transfer status", t.Status)
	}
	if !t.Step.IsValid() {
		return database.NewValidationError("'%s' is not a valid transfer step", t.Step)
	}
	if !t.Error.Code.IsValid() {
		return database.NewValidationError("'%s' is not a valid transfer error code", t.Error.Code)
	}

	if t.LocalPath != "" {
		t.LocalPath = utils.ToOSPath(t.LocalPath)
		if !path.IsAbs(t.LocalPath) && t.LocalPath != path.Base(t.LocalPath) {
			return database.NewValidationError("the local file cannot contain subdirectories")
		}
	} else {
		return database.NewValidationError("the local file cannot be empty")
	}
	if t.RemotePath != "" {
		t.RemotePath = utils.ToStandardPath(t.RemotePath)
		if !path.IsAbs(t.RemotePath) && t.RemotePath != path.Base(t.RemotePath) {
			return database.NewValidationError("the remote file cannot contain subdirectories")
		}
	} else {
		return database.NewValidationError("the remote file cannot be empty")
	}

	n, err := db.Count(&Rule{}).Where("id=?", t.RuleID).Run()
	if err != nil {
		return err
	}
	if n == 0 {
		return database.NewValidationError("the rule %d does not exist", t.RuleID)
	}

	if t.IsServer {
		if err := t.validateServerTransfer(db); err != nil {
			return err
		}
	} else {
		if err := t.validateClientTransfer(db); err != nil {
			return err
		}
	}

	return nil
}

// ToHistory converts the `Transfer` entry into an equivalent `TransferHistory`
// entry with the given time as the end date.
func (t *Transfer) ToHistory(db database.ReadAccess, stop time.Time) (*TransferHistory, database.Error) {
	rule := &Rule{}
	if err := db.Get(rule, "id=?", t.RuleID).Run(); err != nil {
		return nil, err
	}
	agentName := ""
	accountLogin := ""
	protocol := ""

	if t.IsServer {
		agent := &LocalAgent{}
		if err := db.Get(agent, "id=?", t.AgentID).Run(); err != nil {
			return nil, err
		}
		account := &LocalAccount{}
		if err := db.Get(account, "id=?", t.AccountID).Run(); err != nil {
			return nil, err
		}
		agentName = agent.Name
		accountLogin = account.Login
		protocol = agent.Protocol
	} else {
		agent := &RemoteAgent{}
		if err := db.Get(agent, "id=?", t.AgentID).Run(); err != nil {
			return nil, err
		}
		account := &RemoteAccount{}
		if err := db.Get(account, "id=?", t.AccountID).Run(); err != nil {
			return nil, err
		}
		agentName = agent.Name
		accountLogin = account.Login
		protocol = agent.Protocol
	}
	if !types.ValidateStatusForHistory(t.Status) {
		return nil, database.NewValidationError(
			"a transfer cannot be recorded in history with status '%s'", t.Status,
		)
	}

	hist := TransferHistory{
		ID:               t.ID,
		Owner:            t.Owner,
		RemoteTransferID: t.RemoteTransferID,
		IsServer:         t.IsServer,
		IsSend:           rule.IsSend,
		Account:          accountLogin,
		Agent:            agentName,
		Protocol:         protocol,
		LocalPath:        t.LocalPath,
		RemotePath:       t.RemotePath,
		Rule:             rule.Name,
		Start:            t.Start,
		Stop:             stop,
		Status:           t.Status,
		Error:            t.Error,
		Step:             t.Step,
		Progress:         t.Progress,
		TaskNumber:       t.TaskNumber,
	}
	return &hist, nil
}

// GetTransferInfo returns the list of the transfer's TransferInfo as a map[string]string
func (t *Transfer) GetTransferInfo(db database.ReadAccess, tID uint64) (map[string]string, error) {
	var infoList TransferInfoList
	if err := db.Select(&infoList).Where("transfer_id=?", tID).Run(); err != nil {
		return nil, err
	}

	infoMap := make(map[string]string, len(infoList))
	for _, info := range infoList {
		infoMap[info.Name] = info.Value
	}
	return infoMap, nil
}
