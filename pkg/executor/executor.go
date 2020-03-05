// Package executor contains the module responsible for the execution and
// monitoring of a transfer, as well as executing the tasks tied to the transfer.
package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/sftp"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
	"github.com/go-xorm/builder"
)

var errShutdown = errors.New("server shutdown signal received")

type transferInfo struct {
	*model.Transfer
	remoteAgent   *model.RemoteAgent
	remoteAccount *model.RemoteAccount
	remoteCert    *model.Cert
	rule          *model.Rule
}

// Executor is the process responsible for executing outgoing transfers.
type Executor struct {
	Db       *database.Db
	Logger   *log.Logger
	R66Home  string
	Shutdown <-chan bool
}

func newTransferInfo(db *database.Db, trans *model.Transfer) (*transferInfo, error) {
	remote := model.RemoteAgent{ID: trans.AgentID}
	if err := db.Get(&remote); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the partner n°%v does not exist", trans.AgentID)
		}
		return nil, err
	}
	certs, err := remote.GetCerts(db)
	if err != nil || len(certs) == 0 {
		if len(certs) == 0 {
			return nil, fmt.Errorf("no certificates found for agent n°%v", remote.ID)
		}
		return nil, err
	}
	account := model.RemoteAccount{ID: trans.AccountID}
	if err := db.Get(&account); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the account n°%v does not exist", account.ID)
		}
		return nil, err
	}
	if account.RemoteAgentID != remote.ID {
		return nil, fmt.Errorf("the account n°%v does not belong to agent n°%v",
			account.ID, remote.ID)
	}

	rule := model.Rule{ID: trans.RuleID}
	if err := db.Get(&rule); err != nil {
		if err == database.ErrNotFound {
			return nil, fmt.Errorf("the rule n°%v does not exist", rule.ID)
		}
		return nil, err
	}

	return &transferInfo{
		Transfer:      trans,
		remoteAgent:   &remote,
		remoteAccount: &account,
		remoteCert:    &certs[0],
		rule:          &rule,
	}, nil
}

func (e *Executor) logTransfer(trans *model.Transfer, errTrans error) {

	dbErr := func() error {
		ses, err := e.Db.BeginTransaction()
		if err != nil {
			return err
		}

		if err := ses.Delete(&model.Transfer{ID: trans.ID}); err != nil {
			ses.Rollback()
			return err
		}

		if errTrans != nil {
			trans.Status = model.StatusError
			e.Logger.Errorf("Transfer error => %s", errTrans)
		} else {
			trans.Status = model.StatusDone
		}

		hist, err := trans.ToHistory(ses, time.Now())
		if err != nil {
			ses.Rollback()
			return err
		}

		if err := ses.Create(hist); err != nil {
			ses.Rollback()
			return err
		}

		return ses.Commit()
	}()
	if dbErr != nil {
		e.Logger.Errorf("Database error => %s", dbErr)
	}
}

type runner func(*transferInfo) error

func (e *Executor) sftpTransfer(info *transferInfo) error {

	context, err := sftp.Connect(info.remoteAgent, info.remoteCert, info.remoteAccount)
	if err != nil {
		return err
	}
	defer context.Close()

	done := make(chan bool)
	go func() {
		select {
		case <-e.Shutdown:
			updt := &model.Transfer{
				Error: model.NewTransferError(model.TeShuttingDown, errShutdown.Error()),
			}
			if err := e.Db.Update(updt, info.Transfer.ID, false); err != nil {
				e.Logger.Criticalf("Database error: %s", err)
			}
			context.Close()
		case <-done:
		}
	}()
	if err := sftp.DoTransfer(context.SftpClient, info.Transfer, info.rule); err != nil {
		return err
	}

	return nil
}

func (e *Executor) r66Transfer(info *transferInfo) error {

	script := fmt.Sprintf("%s/bin/waarp-r66client.sh", e.R66Home)
	args := []string{
		info.remoteAccount.Login,
		"send",
		"-to", info.remoteAgent.Name,
		"-file", info.Transfer.SourcePath,
		"-rule", info.rule.Name,
	}
	cmd := exec.Command(script, args...) //nolint:gosec
	out, err := cmd.Output()
	if err != nil {
		info.Transfer.Error = model.TransferError{
			Code:    model.TeExternalOperation,
			Details: err.Error(),
		}
	}
	if len(out) > 0 {
		// Get the second line of the output
		arrays := bytes.Split(out, []byte("\n"))
		if len(arrays) < 2 {
			return fmt.Errorf("bad output")
		}
		// Parse into a r66Result
		result := &r66Result{}
		if err := json.Unmarshal(arrays[1], result); err != nil {
			return err
		}
		// Add R66 result info to the transfer
		if err = info.Transfer.Error.Code.Scan(result.StatusCode); err != nil {
			return err
		}
		info.Transfer.Error.Details = result.StatusTxt
		info.Transfer.DestPath = result.FinalPath
		buf, err := json.Marshal(result)
		if err != nil {
			return err
		}
		info.Transfer.ExtInfo = buf
	}
	return err
}

type r66Result struct {
	SpecialID       int
	StatusCode      string
	StatusTxt       string
	FinalPath       string
	FileInformation string
	OriginalSize    uint
}

func updateStatus(db *database.Db, trans *model.Transfer, status model.TransferStatus) error {
	err := db.Update(&model.Transfer{Status: status, Error: trans.Error}, trans.ID, false)
	if err != nil {
		return err
	}
	trans.Status = status
	return nil
}

func getTasks(db *database.Db, ruleID uint64, chain model.Chain) ([]*model.Task, error) {
	list := []*model.Task{}
	filters := &database.Filters{
		Order:      "rank ASC",
		Conditions: builder.Eq{"rule_id": ruleID, "chain": chain},
	}

	if err := db.Select(&list, filters); err != nil {
		return nil, err
	}
	return list, nil
}

func transferPrologue(db *database.Db, trans *model.Transfer) (*transferInfo, error) {
	auth, err := model.IsRuleAuthorized(db, trans)
	if err != nil {
		return nil, err
	} else if !auth {
		return nil, database.InvalidError("Rule %d is not authorized for this transfer", trans.RuleID)
	}

	info, err := newTransferInfo(db, trans)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func (e *Executor) runTasks(chain model.Chain, info *transferInfo) error {
	taskChain, err := getTasks(e.Db, info.rule.ID, chain)
	if err != nil {
		return err
	}

	processor := tasks.Processor{
		Db:       e.Db,
		Logger:   e.Logger,
		Rule:     info.rule,
		Transfer: info.Transfer,
		Shutdown: e.Shutdown,
	}

	err = processor.RunTasks(taskChain)
	if err == tasks.ErrShutdown {
		info.Transfer.Error = model.NewTransferError(model.TeShuttingDown,
			errShutdown.Error())
		return errShutdown
	}
	return err
}

func (e *Executor) runTransfer(info *transferInfo, run runner) {
	err := func() error {
		if info.remoteAgent.Protocol != "r66" {
			if err := updateStatus(e.Db, info.Transfer, model.StatusPreTasks); err != nil {
				return err
			}

			if err := e.runTasks(model.ChainPre, info); err != nil {
				return err
			}
		}

		if err := run(info); err != nil {
			return err
		}

		if info.remoteAgent.Protocol != "r66" {
			if err := updateStatus(e.Db, info.Transfer, model.StatusPostTasks); err != nil {
				return err
			}

			return e.runTasks(model.ChainPost, info)
		}

		return nil
	}()
	if err != nil {
		_ = e.runTasks(model.ChainError, info)
	}
	e.logTransfer(info.Transfer, err)
}

// Run executes the given transfer.
func (e *Executor) Run(trans model.Transfer) {

	info, err := transferPrologue(e.Db, &trans)
	if err != nil {
		e.logTransfer(&trans, err)
		return
	}

	runner := e.getProtocolRunner(info.remoteAgent.Protocol)
	if runner == nil {
		e.logTransfer(&trans, fmt.Errorf("unknown protocol"))
		return
	}

	if err := updateStatus(e.Db, info.Transfer, model.StatusTransfer); err != nil {
		if err := updateStatus(e.Db, info.Transfer, model.StatusErrorTasks); err != nil {
			e.logTransfer(&trans, err)
			return
		}
	}

	e.runTransfer(info, runner)
}

func (e *Executor) getProtocolRunner(protocol string) runner {

	switch protocol {
	case "sftp":
		return e.sftpTransfer
	case "r66":
		return e.r66Transfer
	}
	return nil
}
