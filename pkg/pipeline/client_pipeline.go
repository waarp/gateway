package pipeline

import (
	"context"
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
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
	pip    *Pipeline
	client Client
}

// NewClientPipeline initializes and returns a new ClientPipeline for the given
// transfer.
func NewClientPipeline(db *database.DB, trans *model.Transfer) (*ClientPipeline, *types.TransferError) {
	logger := log.NewLogger(fmt.Sprintf("Pipeline %d (client)", trans.ID))

	transCtx, err := model.GetTransferContext(db, logger, trans)
	if err != nil {
		return nil, err
	}

	constr, ok := ClientConstructors[transCtx.RemoteAgent.Protocol]
	if !ok {
		logger.Errorf("No client found for protocol %s", transCtx.RemoteAgent.Protocol)

		return nil, types.NewTransferError(types.TeInternal, "no client found for protocol %s",
			transCtx.RemoteAgent.Protocol)
	}

	pipeline, err := newPipeline(db, logger, transCtx)
	if err != nil {
		return nil, err
	}

	client, err := constr(pipeline)
	if err != nil {
		return nil, err
	}

	c := &ClientPipeline{
		pip:    pipeline,
		client: client,
	}
	ClientTransfers.Add(trans.ID, c)

	return c, nil
}

//nolint:dupl // factorizing would hurt readability
func (c *ClientPipeline) preTasks() error {
	// Simple pre-tasks
	pt, ok := c.client.(PreTasksHandler)
	if !ok {
		if err := c.pip.PreTasks(); err != nil {
			c.client.SendError(err)

			return err
		}

		return nil
	}

	// Extended pre-task handling
	if err := pt.BeginPreTasks(); err != nil {
		c.pip.SetError(err)

		return err
	}

	if err := c.pip.PreTasks(); err != nil {
		c.client.SendError(err)

		return err
	}

	if err := pt.EndPreTasks(); err != nil {
		c.pip.SetError(err)

		return err
	}

	return nil
}

//nolint:dupl // factorizing would hurt readability
func (c *ClientPipeline) postTasks() error {
	// Simple post-tasks
	pt, ok := c.client.(PostTasksHandler)
	if !ok {
		if err := c.pip.PostTasks(); err != nil {
			c.client.SendError(err)

			return err
		}

		return nil
	}

	// Extended post-task handling
	if err := pt.BeginPostTasks(); err != nil {
		c.pip.SetError(err)

		return err
	}

	if err := c.pip.PostTasks(); err != nil {
		c.client.SendError(err)

		return err
	}

	if err := pt.EndPostTasks(); err != nil {
		c.pip.SetError(err)

		return err
	}

	return nil
}

// Run executes the full client transfer pipeline in order. If a transfer error
// occurs, it will be handled internally.
func (c *ClientPipeline) Run() {
	defer ClientTransfers.Delete(c.pip.TransCtx.Transfer.ID)

	// REQUEST
	if err := c.client.Request(); err != nil {
		c.pip.SetError(err)
		c.client.SendError(err)

		return
	}

	// PRE-TASKS
	if c.preTasks() != nil {
		return
	}

	// DATA
	file, fErr := c.pip.StartData()
	if fErr != nil {
		return
	}

	if err := c.client.Data(file); err != nil {
		c.pip.SetError(err)
		c.client.SendError(err)

		return
	}

	if err := c.pip.EndData(); err != nil {
		c.client.SendError(err)

		return
	}

	// POST-TASKS
	if c.postTasks() != nil {
		return
	}

	// END TRANSFER
	if err := c.client.EndTransfer(); err != nil {
		c.pip.SetError(err)

		return
	}

	//nolint:errcheck // error is irrelevant at this point
	_ = c.pip.EndTransfer()
}

// Pause stops the client pipeline and pauses the transfer.
func (c *ClientPipeline) Pause(ctx context.Context) error {
	handle := func() {
		done := make(chan struct{})

		go func() {
			defer close(done)

			if pa, ok := c.client.(PauseHandler); ok {
				_ = pa.Pause() //nolint:errcheck // error is irrelevant at this point
			} else {
				c.client.SendError(types.NewTransferError(types.TeStopped,
					"transfer paused by user"))
			}
		}()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	c.pip.Pause(handle)

	return nil
}

// Interrupt stops the client pipeline and interrupts the transfer.
func (c *ClientPipeline) Interrupt(ctx context.Context) error {
	handle := func() {
		done := make(chan struct{})

		go func() {
			defer close(done)

			c.client.SendError(types.NewTransferError(types.TeShuttingDown,
				"transfer interrupted by service shutdown"))
		}()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	c.pip.Interrupt(handle)

	return nil
}

// Cancel stops the client pipeline and cancels the transfer.
func (c *ClientPipeline) Cancel(ctx context.Context) (err error) {
	handle := func() {
		done := make(chan struct{})

		go func() {
			defer close(done)

			if ca, ok := c.client.(CancelHandler); ok {
				_ = ca.Cancel() //nolint:errcheck // error is irrelevant at this point
			} else {
				c.client.SendError(types.NewTransferError(types.TeCanceled,
					"transfer canceled by user"))
			}
		}()
		select {
		case <-done:
		case <-ctx.Done():
		}
	}

	c.pip.Cancel(handle)

	return nil
}
