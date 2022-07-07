package pipeline

import (
	"context"
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// ClientTransfers is a synchronized map containing the pipelines of all currently
// running client transfers. It can be used to interrupt transfers using the various
// functions exposed by the TransferInterrupter interface.
//nolint:gochecknoglobals // global var is necessary so that transfers can be managed from admin
var ClientTransfers = service.NewTransferMap()

// ClientPipeline associates a Pipeline with a Client, allowing to run complete
// client transfers.
type ClientPipeline struct {
	Pip    *Pipeline
	Client Client
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

	cli, cols, tErr := newClientPipeline(db, logger, transCtx)
	if tErr != nil {
		trans.Status = types.StatusError
		trans.Error = *tErr

		cols = append(cols, "status", "error_code", "error_details")

		if dbErr := db.Update(transCtx.Transfer).Cols(cols...).Run(); dbErr != nil {
			logger.Error("Failed to update the transfer error: %s", dbErr)
		}

		return nil, tErr
	}

	if dbErr := db.Update(transCtx.Transfer).Cols(cols...).Run(); dbErr != nil {
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
) (*ClientPipeline, []string, *types.TransferError) {
	proto := transCtx.RemoteAgent.Protocol

	constr, ok := ClientConstructors[proto]
	if !ok {
		logger.Error("No client found for protocol %s", proto)

		return nil, nil, types.NewTransferError(types.TeInternal,
			fmt.Sprintf("no client found for protocol %s", proto))
	}

	pipeline, cols, pErr := NewPipeline(db, logger, transCtx)
	if pErr != nil {
		return nil, cols, pErr
	}

	client, err := constr(pipeline)
	if err != nil {
		logger.Error("Failed to instantiate the %s transfer client: %s", proto, err)

		return nil, cols, err
	}

	c := &ClientPipeline{
		Pip:    pipeline,
		Client: client,
	}

	ClientTransfers.Add(transCtx.Transfer.ID, c)

	return c, cols, nil
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
	defer ClientTransfers.Delete(c.Pip.TransCtx.Transfer.ID)

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
				_ = pa.Pause() //nolint:errcheck // error is irrelevant at this point
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
				_ = ca.Cancel() //nolint:errcheck // error is irrelevant at this point
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
