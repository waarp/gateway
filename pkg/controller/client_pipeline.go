package controller

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoinits //needed to avoid import cycles with tasks module
func init() {
	tasks.NewClientPipeline = func(db *database.DB, trans *model.Transfer) (tasks.ClientPipeline, error) {
		return NewClientPipeline(db, trans)
	}
}

type Error = pipeline.Error

// ClientPipeline associates a Pipeline with a TransferClient, allowing to run complete
// client transfers.
type ClientPipeline struct {
	Pip    *pipeline.Pipeline
	Client protocol.TransferClient
}

// NewClientPipeline initializes and returns a new ClientPipeline for the given
// transfer.
func NewClientPipeline(db *database.DB, trans *model.Transfer) (*ClientPipeline, error) {
	logger := logging.NewLogger(fmt.Sprintf("Pipeline %d (client)", trans.ID))

	transCtx, ctxErr := model.GetTransferContext(db, logger, trans)
	if ctxErr != nil {
		trans.Status = types.StatusError
		trans.ErrCode = types.TeInternal
		trans.ErrDetails = fmt.Sprintf("failed to retrieve the transfer context: %v", ctxErr)

		if dbErr := db.Update(trans).Run(); dbErr != nil {
			logger.Errorf("Failed to update the transfer error: %s", dbErr)
		}

		return nil, fmt.Errorf("failed to retrieve the transfer context: %w", ctxErr)
	}

	cli, cliErr := newClientPipeline(db, logger, transCtx)
	if cliErr != nil {
		trans.Status = types.StatusError
		trans.ErrCode = cliErr.Code()
		trans.ErrDetails = cliErr.Details()

		if dbErr := db.Update(trans).Run(); dbErr != nil {
			logger.Errorf("Failed to update the transfer error: %s", dbErr)
		}

		return nil, cliErr
	}

	if dbErr := cli.Pip.UpdateTrans(); dbErr != nil {
		logger.Errorf("Failed to update the transfer details: %s", dbErr)

		return nil, dbErr
	}

	return cli, nil
}

func newClientPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) (*ClientPipeline, *Error) {
	dbClient := transCtx.Client

	client, ok := services.Clients[dbClient.Name]
	if !ok {
		logger.Errorf("No client %q found", dbClient.Name)

		return nil, pipeline.NewErrorf(types.TeInternal, "no client %q found", dbClient.Name)
	}

	if state, _ := client.State(); state != utils.StateRunning {
		logger.Errorf("Client %q is not active, cannot initiate transfer", dbClient.Name)

		return nil, pipeline.NewErrorf(types.TeShuttingDown, "client %q is not active", dbClient.Name)
	}

	pip, pipErr := pipeline.NewClientPipeline(db, logger, transCtx, snmp.GlobalService)
	if pipErr != nil {
		logger.Errorf("Failed to initialize the client transfer pipeline: %v", pipErr)

		return nil, pipErr
	}

	clientTransfer, cliErr := client.InitTransfer(pip)
	if cliErr != nil {
		pip.SetError(cliErr.Code(), cliErr.Details())
		logger.Errorf("Failed to instantiate the %q transfer client: %s",
			dbClient.Name, cliErr)

		return nil, cliErr
	}

	c := &ClientPipeline{
		Pip:    pip,
		Client: clientTransfer,
	}

	pip.SetInterruptionHandlers(c.Pause, c.Interrupt, c.Cancel)
	pip.SetProtocolAgent(clientTransfer)

	if transCtx.Rule.IsSend {
		logger.Infof("Starting upload of file %q to %q as %q using rule %q",
			transCtx.Transfer.LocalPath, transCtx.RemoteAgent.Name,
			transCtx.RemoteAccount.Login, transCtx.Rule.Name)
	} else {
		logger.Infof("Starting download of file %q from %q as %q using rule %q",
			transCtx.Transfer.RemotePath, transCtx.RemoteAgent.Name,
			transCtx.RemoteAccount.Login, transCtx.Rule.Name)
	}

	return c, nil
}

//nolint:dupl // factorizing would hurt readability
func (c *ClientPipeline) preTasks() *Error {
	// Simple pre-tasks
	pt, ok := c.Client.(protocol.PreTasksHandler)
	if !ok {
		if err := c.Pip.PreTasks(); err != nil {
			c.Client.SendError(err.Code(), err.Details())

			return err
		}

		return nil
	}

	// Extended pre-task handling
	if err := pt.BeginPreTasks(); err != nil {
		return wrapRemoteError("remote pre-tasks failed", err)
	}

	if err := c.Pip.PreTasks(); err != nil {
		c.Client.SendError(err.Code(), err.Details())

		return err
	}

	if err := pt.EndPreTasks(); err != nil {
		return wrapRemoteError("remote pre-tasks failed", err)
	}

	return nil
}

//nolint:dupl // factorizing would hurt readability
func (c *ClientPipeline) postTasks() *Error {
	// Simple post-tasks
	pt, ok := c.Client.(protocol.PostTasksHandler)
	if !ok {
		if err := c.Pip.PostTasks(); err != nil {
			c.Client.SendError(err.Code(), err.Details())

			return err
		}

		return nil
	}

	// Extended post-task handling
	if err := pt.BeginPostTasks(); err != nil {
		return wrapRemoteError("remote post-tasks failed", err)
	}

	if err := c.Pip.PostTasks(); err != nil {
		c.Client.SendError(err.Code(), err.Details())

		return err
	}

	if err := pt.EndPostTasks(); err != nil {
		return wrapRemoteError("remote post-tasks failed", err)
	}

	return nil
}

// Run executes the full client transfer pipeline in order. If a transfer error
// occurs, it will be handled internally.
//
//nolint:funlen //splitting would hurt readability
func (c *ClientPipeline) Run() error {
	// REQUEST
	if err := c.Client.Request(); err != nil {
		tErr := wrapRemoteError("transfer request failed", err)
		c.Pip.SetError(tErr.Code(), tErr.Details())

		return tErr
	}

	// PRE-TASKS
	if err := c.preTasks(); err != nil {
		return err
	}

	// DATA
	file, fErr := c.Pip.StartData()
	if fErr != nil {
		c.Client.SendError(fErr.Code(), fErr.Details())

		return fErr
	}

	var dataErr *Error

	if c.Pip.TransCtx.Rule.IsSend {
		if err := c.Client.Send(file); err != nil {
			dataErr = wrapRemoteError("file sending failed", err)
		}
	} else {
		if err := c.Client.Receive(file); err != nil {
			dataErr = wrapRemoteError("file reception failed", err)
		}
	}

	if dataErr != nil {
		c.Pip.SetError(dataErr.Code(), dataErr.Details())

		return dataErr
	}

	if err := c.Pip.EndData(); err != nil {
		c.Client.SendError(err.Code(), err.Details())

		return err
	}

	// POST-TASKS
	if err := c.postTasks(); err != nil {
		return err
	}

	// END TRANSFER
	if err := c.Client.EndTransfer(); err != nil {
		tErr := wrapRemoteError("transfer finalization failed", err)
		c.Pip.SetError(tErr.Code(), tErr.Details())

		return tErr
	}

	//nolint:revive //do not drop the "if", return types are different
	if err := c.Pip.EndTransfer(); err != nil {
		return err
	}

	return nil
}

// Pause stops the client pipeline and pauses the transfer.
//
//nolint:dupl //factorizing would add complexity
func (c *ClientPipeline) Pause(ctx context.Context) error {
	if err := utils.RunWithCtx(ctx, func() error {
		if pa, ok := c.Client.(protocol.PauseHandler); ok {
			if err := pa.Pause(); err != nil {
				c.Pip.Logger.Errorf("Failed to pause remote transfer: %v", err)

				return fmt.Errorf("failed to pause remote transfer: %w", err)
			}
		} else {
			c.Client.SendError(types.TeStopped, "transfer paused by user")
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to pause remote transfer: %w", err)
	}

	return nil
}

// Interrupt stops the client pipeline and interrupts the transfer.
func (c *ClientPipeline) Interrupt(ctx context.Context) error {
	if err := utils.RunWithCtx(ctx, func() error {
		c.Client.SendError(types.TeShuttingDown, "transfer interrupted by service shutdown")

		return nil
	}); err != nil {
		return fmt.Errorf("failed to interrupt remote transfer: %w", err)
	}

	return nil
}

// Cancel stops the client pipeline and cancels the transfer.
//
//nolint:dupl //factorizing would add complexity
func (c *ClientPipeline) Cancel(ctx context.Context) error {
	if err := utils.RunWithCtx(ctx, func() error {
		if ca, ok := c.Client.(protocol.CancelHandler); ok {
			if err := ca.Cancel(); err != nil {
				c.Pip.Logger.Errorf("Failed to cancel remote transfer: %v", err)

				return fmt.Errorf("failed to cancel remote transfer: %w", err)
			}
		} else {
			c.Client.SendError(types.TeCanceled, "transfer canceled by user")
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to cancel remote transfer: %w", err)
	}

	return nil
}

func wrapRemoteError(msg string, err error) *Error {
	var pErr *Error
	if errors.As(err, &pErr) {
		return pErr
	}

	return pipeline.NewErrorWith(types.TeUnknownRemote, msg, err)
}
