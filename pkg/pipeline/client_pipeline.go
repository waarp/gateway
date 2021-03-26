package pipeline

import (
	"fmt"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline/internal"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
)

type ClientPipeline struct {
	pip    *Pipeline
	client Client
}

func NewClientPipeline(db *database.DB, paths *conf.PathsConfig,
	trans *model.Transfer) (*ClientPipeline, error) {

	logger := log.NewLogger(fmt.Sprintf("Pipeline %d", trans.ID))

	info, err := internal.GetTransferInfo(db, logger, trans, paths)
	if err != nil {
		return nil, err
	}

	constr, ok := ClientConstructors[info.RemoteAgent.Protocol]
	if !ok {
		logger.Errorf("No client found for protocol %s", info.RemoteAgent.Protocol)
		return nil, err
	}

	client, err := constr(logger, info)
	if err != nil {
		return nil, err
	}

	pipeline, err := newPipeline(db, logger, info)
	if err != nil {
		return nil, err
	}

	c := &ClientPipeline{
		pip:    pipeline,
		client: client,
	}
	RunningTransfers.Store(trans.ID, c)
	return c, nil
}

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

func (c *ClientPipeline) Run() {
	// REQUEST
	if err := c.client.Request(); err != nil {
		c.pip.SetError(err)
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
		return
	}
	if c.pip.EndData() != nil {
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

	_ = c.pip.EndTransfer()
}

func (c *ClientPipeline) Pause() {
	if pa, ok := c.client.(PauseHandler); ok {
		_ = pa.Pause()
	} else {
		c.client.SendError(types.NewTransferError(types.TeStopped,
			"transfer paused by user"))
	}
	c.pip.Pause()
}

func (c *ClientPipeline) Interrupt() {
	c.client.SendError(types.NewTransferError(types.TeShuttingDown,
		"transfer interrupted by service shutdown"))
	c.pip.interrupt()
}

func (c *ClientPipeline) Cancel() {
	if ca, ok := c.client.(CancelHandler); ok {
		_ = ca.Cancel()
	} else {
		c.client.SendError(types.NewTransferError(types.TeCanceled,
			"transfer cancelled by user"))
	}
	c.pip.Cancel()
}
