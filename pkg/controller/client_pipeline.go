package controller

import (
	"context"
	"errors"
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

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

	transCtx, err := model.GetTransferContext(db, logger, trans)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the transfer context: %w", err)
	}

	cli, cliErr := newClientPipeline(db, logger, transCtx)
	if cliErr != nil {
		trans.Status = types.StatusError

		tErr := types.NewTransferError(types.TeInternal, cliErr.Error())
		errors.As(cliErr, &tErr)

		trans.Error = *tErr

		if dbErr := db.Update(trans).Run(); dbErr != nil {
			logger.Error("Failed to update the transfer error: %s", dbErr)
		}

		return nil, cliErr
	}

	if dbErr := cli.Pip.UpdateTrans(); dbErr != nil {
		logger.Error("Failed to update the transfer details: %s", dbErr)

		return nil, pipeline.ErrDatabase
	}

	return cli, nil
}

func newClientPipeline(db *database.DB, logger *log.Logger, transCtx *model.TransferContext,
) (*ClientPipeline, *types.TransferError) {
	dbClient := transCtx.Client

	client, ok := services.Clients[dbClient.Name]
	if !ok {
		logger.Error("No client %q found", dbClient.Name)

		return nil, types.NewTransferError(types.TeInternal,
			fmt.Sprintf("no client %q found", dbClient.Name))
	}

	if state, _ := client.State(); state != utils.StateRunning {
		logger.Error("Client %q is not active, cannot initiate transfer", dbClient.Name)

		return nil, types.NewTransferError(types.TeShuttingDown,
			fmt.Sprintf("client %q is not active", dbClient.Name))
	}

	pip, pipErr := pipeline.NewClientPipeline(db, logger, transCtx)
	if pipErr != nil {
		logger.Error("Failed to initialize the client transfer pipeline: %v", pipErr)

		return nil, asTransferError(types.TeInternal, pipErr)
	}

	clientService, err := client.InitTransfer(pip)
	if err != nil {
		tErr := asTransferError(types.TeUnknownRemote, err)

		pip.SetError(tErr)
		logger.Error("Failed to instantiate the %q transfer client: %s",
			dbClient.Name, err)

		return nil, tErr
	}

	c := &ClientPipeline{
		Pip:    pip,
		Client: clientService,
	}

	pip.SetInterruptionHandlers(c.Pause, c.Interrupt, c.Cancel)

	if transCtx.Rule.IsSend {
		logger.Info("Starting upload of file %q to %q as %q using rule %q",
			&transCtx.Transfer.LocalPath, transCtx.RemoteAgent.Name,
			transCtx.RemoteAccount.Login, transCtx.Rule.Name)
	} else {
		logger.Info("Starting download of file %q from %q as %q using rule %q",
			transCtx.Transfer.RemotePath, transCtx.RemoteAgent.Name,
			transCtx.RemoteAccount.Login, transCtx.Rule.Name)
	}

	return c, nil
}

//nolint:dupl // factorizing would hurt readability
func (c *ClientPipeline) preTasks() *types.TransferError {
	// Simple pre-tasks
	pt, ok := c.Client.(protocol.PreTasksHandler)
	if !ok {
		if err := c.Pip.PreTasks(); err != nil {
			tErr := asTransferError(types.TeExternalOperation, err)
			c.Client.SendError(tErr)

			return tErr
		}

		return nil
	}

	// Extended pre-task handling
	if err := pt.BeginPreTasks(); err != nil {
		return asTransferError(types.TeUnknownRemote, err)
	}

	if err := c.Pip.PreTasks(); err != nil {
		tErr := asTransferError(types.TeExternalOperation, err)
		c.Client.SendError(tErr)

		return tErr
	}

	if err := pt.EndPreTasks(); err != nil {
		return asTransferError(types.TeUnknownRemote, err)
	}

	return nil
}

//nolint:dupl // factorizing would hurt readability
func (c *ClientPipeline) postTasks() *types.TransferError {
	// Simple post-tasks
	pt, ok := c.Client.(protocol.PostTasksHandler)
	if !ok {
		if err := c.Pip.PostTasks(); err != nil {
			tErr := asTransferError(types.TeExternalOperation, err)
			c.Client.SendError(tErr)

			return tErr
		}

		return nil
	}

	// Extended post-task handling
	if err := pt.BeginPostTasks(); err != nil {
		return asTransferError(types.TeUnknownRemote, err)
	}

	if err := c.Pip.PostTasks(); err != nil {
		tErr := asTransferError(types.TeExternalOperation, err)
		c.Client.SendError(tErr)

		return tErr
	}

	if err := pt.EndPostTasks(); err != nil {
		return asTransferError(types.TeUnknownRemote, err)
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
		tErr := asTransferError(types.TeUnknownRemote, err)
		c.Pip.SetError(tErr)

		return tErr
	}

	// PRE-TASKS
	if err := c.preTasks(); err != nil {
		return err
	}

	// DATA
	file, fErr := c.Pip.StartData()
	if fErr != nil {
		tErr := asTransferError(types.TeInternal, fErr)
		c.Client.SendError(tErr)

		return tErr
	}

	var dataErr error
	if c.Pip.TransCtx.Rule.IsSend {
		dataErr = c.Client.Send(file)
	} else {
		dataErr = c.Client.Receive(file)
	}

	if dataErr != nil {
		tErr := asTransferError(types.TeUnknownRemote, dataErr)
		c.Pip.SetError(tErr)

		return tErr
	}

	if err := c.Pip.EndData(); err != nil {
		tErr := asTransferError(types.TeInternal, err)
		c.Client.SendError(tErr)

		return tErr
	}

	// POST-TASKS
	if err := c.postTasks(); err != nil {
		return err
	}

	// END TRANSFER
	if err := c.Client.EndTransfer(); err != nil {
		tErr := asTransferError(types.TeUnknownRemote, err)
		c.Pip.SetError(tErr)

		return tErr
	}

	if err := c.Pip.EndTransfer(); err != nil {
		tErr := asTransferError(types.TeInternal, err)

		return tErr
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
				c.Pip.Logger.Error("Failed to pause remote transfer: %v", err)

				return fmt.Errorf("failed to pause remote transfer: %w", err)
			}
		} else {
			c.Client.SendError(types.NewTransferError(types.TeStopped,
				"transfer paused by user"))
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
		c.Client.SendError(types.NewTransferError(types.TeShuttingDown,
			"transfer interrupted by service shutdown"))

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
				c.Pip.Logger.Error("Failed to cancel remote transfer: %v", err)

				return fmt.Errorf("failed to cancel remote transfer: %w", err)
			}
		} else {
			c.Client.SendError(types.NewTransferError(types.TeCanceled,
				"transfer canceled by user"))
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to cancel remote transfer: %w", err)
	}

	return nil
}
