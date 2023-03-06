package pipeline

import (
	"context"
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/service/state"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// ClientPipeline associates a Pipeline with a TransferClient, allowing to run complete
// client transfers.
type ClientPipeline struct {
	Pip    *Pipeline
	Client TransferClient
}

// NewClientPipeline initializes and returns a new ClientPipeline for the given
// transfer.
func NewClientPipeline(db *database.DB, trans *model.Transfer,
) (*ClientPipeline, *types.TransferError) {
	logger := conf.GetLogger(fmt.Sprintf("Pipeline %d (client)", trans.ID))

	transCtx, err := model.GetTransferContext(db, logger, trans)
	if err != nil {
		return nil, err
	}

	cli, tErr := newClientPipeline(db, logger, transCtx)
	if tErr != nil {
		trans.Status = types.StatusError
		trans.Error = *tErr

		if dbErr := db.Update(transCtx.Transfer).Run(); dbErr != nil {
			logger.Error("Failed to update the transfer error: %s", dbErr)
		}

		return nil, tErr
	}

	if dbErr := cli.Pip.UpdateTrans(); dbErr != nil {
		logger.Error("Failed to update the transfer details: %s", dbErr)

		return nil, errDatabase
	}

	if iErr := cli.Pip.init(); iErr != nil {
		return nil, iErr
	}

	return cli, nil
}

func newClientPipeline(db *database.DB, logger *log.Logger,
	transCtx *model.TransferContext,
) (*ClientPipeline, *types.TransferError) {
	dbClient := transCtx.Client

	client, ok := Clients[dbClient.Name]
	if !ok {
		logger.Error("No client %q found", dbClient.Name)

		return nil, types.NewTransferError(types.TeInternal,
			fmt.Sprintf("no client %q found", dbClient.Name))
	}

	if code, _ := client.State().Get(); code != state.Running {
		logger.Error("Client %q is not active, cannot initiate transfer", dbClient.Name)

		return nil, types.NewTransferError(types.TeShuttingDown,
			fmt.Sprintf("client %q is not active", dbClient.Name))
	}

	pipeline, pipErr := newPipeline(db, logger, transCtx)
	if pipErr != nil {
		logger.Error("Failed to initialize the client transfer pipeline: %v", pipErr)

		return nil, pipErr
	}

	transferClient, err := client.InitTransfer(pipeline)
	if err != nil {
		logger.Error("Failed to instantiate the %q transfer client: %s",
			dbClient.Name, err)

		return nil, err
	}

	c := &ClientPipeline{
		Pip:    pipeline,
		Client: transferClient,
	}

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
	pt, ok := c.Client.(PreTasksHandler)
	if !ok {
		if err := c.Pip.PreTasks(); err != nil {
			c.Client.SendError(err)

			return err
		}

		return nil
	}

	// Extended pre-task handling
	if err := pt.BeginPreTasks(); err != nil {
		c.Pip.SetError(err)

		return err
	}

	if err := c.Pip.PreTasks(); err != nil {
		c.Client.SendError(err)

		return err
	}

	if err := pt.EndPreTasks(); err != nil {
		c.Pip.SetError(err)

		return err
	}

	return nil
}

//nolint:dupl // factorizing would hurt readability
func (c *ClientPipeline) postTasks() *types.TransferError {
	// Simple post-tasks
	pt, ok := c.Client.(PostTasksHandler)
	if !ok {
		if err := c.Pip.PostTasks(); err != nil {
			c.Client.SendError(err)

			return err
		}

		return nil
	}

	// Extended post-task handling
	if err := pt.BeginPostTasks(); err != nil {
		c.Pip.SetError(err)

		return err
	}

	if err := c.Pip.PostTasks(); err != nil {
		c.Client.SendError(err)

		return err
	}

	if err := pt.EndPostTasks(); err != nil {
		c.Pip.SetError(err)

		return err
	}

	return nil
}

// Run executes the full client transfer pipeline in order. If a transfer error
// occurs, it will be handled internally.
func (c *ClientPipeline) Run() *types.TransferError {
	// REQUEST
	if err := c.Client.Request(); err != nil {
		c.Pip.SetError(err)
		c.Client.SendError(err)

		return err
	}

	// PRE-TASKS
	if err := c.preTasks(); err != nil {
		return err
	}

	// DATA
	file, fErr := c.Pip.StartData()
	if fErr != nil {
		c.Client.SendError(fErr)

		return fErr
	}

	if err := c.Client.Data(file); err != nil {
		c.Pip.SetError(err)
		c.Client.SendError(err)

		return err
	}

	if err := c.Pip.EndData(); err != nil {
		c.Client.SendError(err)

		return err
	}

	// POST-TASKS
	if err := c.postTasks(); err != nil {
		return err
	}

	// END TRANSFER
	if err := c.Client.EndTransfer(); err != nil {
		c.Pip.SetError(err)

		return err
	}

	return c.Pip.EndTransfer()
}

// Pause stops the client pipeline and pauses the transfer.
func (c *ClientPipeline) Pause(ctx context.Context) error {
	handle := func() {
		done := make(chan struct{})

		go func() {
			defer close(done)

			if pa, ok := c.Client.(PauseHandler); ok {
				if err := pa.Pause(); err != nil {
					c.Pip.Logger.Warning("Failed to pause remote transfer: %v", err)
				}
			} else {
				c.Client.SendError(types.NewTransferError(types.TeStopped,
					"transfer paused by user"))
			}
		}()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	c.Pip.Pause(handle)

	return nil
}

// Interrupt stops the client pipeline and interrupts the transfer.
func (c *ClientPipeline) Interrupt(ctx context.Context) error {
	handle := func() {
		done := make(chan struct{})

		go func() {
			defer close(done)

			c.Client.SendError(types.NewTransferError(types.TeShuttingDown,
				"transfer interrupted by service shutdown"))
		}()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	c.Pip.Interrupt(handle)

	return nil
}

// Cancel stops the client pipeline and cancels the transfer.
func (c *ClientPipeline) Cancel(ctx context.Context) (err error) {
	handle := func() {
		done := make(chan struct{})

		go func() {
			defer close(done)

			if ca, ok := c.Client.(CancelHandler); ok {
				if err := ca.Cancel(); err != nil {
					c.Pip.Logger.Warning("Failed to cancel remote transfer: %v", err)
				}
			} else {
				c.Client.SendError(types.NewTransferError(types.TeCanceled,
					"transfer canceled by user"))
			}
		}()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	c.Pip.Cancel(handle)

	return nil
}

func (c *ClientPipeline) Pipeline() *Pipeline {
	return c.Pip
}
