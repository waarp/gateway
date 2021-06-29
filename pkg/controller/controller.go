// Package controller defines the controller module whose purpose is to
// periodically scan the database for new transfers to launch.
package controller

import (
	"context"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

// Controller is the service responsible for checking the database for new
// transfers at regular intervals, and starting those new transfers.
type Controller struct {
	DB *database.DB

	ticker *time.Ticker
	logger *log.Logger
	state  service.State

	wg     *sync.WaitGroup
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Controller) checkIsDBDown() bool {
	if st, _ := c.DB.State().Get(); st != service.Running {
		return true
	}

	query := c.DB.UpdateAll(&model.Transfer{}, database.UpdVals{"status": types.StatusInterrupted},
		"owner=? AND status=?", database.Owner, types.StatusRunning)
	if err := query.Run(); err != nil {
		c.logger.Errorf("Failed to access database: %s", err.Error())
		return true
	}
	return false
}

func (c *Controller) listen() {
	c.wg = &sync.WaitGroup{}
	c.done = make(chan struct{})
	c.ctx, c.cancel = context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				c.wg.Wait()
				close(c.done)
				return
			case <-c.ticker.C:
				c.startNewTransfers()
			}
		}
	}()
}

// startNewTransfers checks the database for new planned transfers and starts
// them, as long as there are available transfer slots.
func (c *Controller) startNewTransfers() {

	if c.checkIsDBDown() {
		return
	}

	var plannedTrans model.Transfers
	query := c.DB.Select(&plannedTrans).Where("owner=? AND status=? AND "+
		"is_server=? AND start<?", database.Owner, types.StatusPlanned, false,
		time.Now().UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)).
		Limit(int(pipeline.TransferOutCount.GetAvailable()), 0)

	if err := query.Run(); err != nil {
		c.logger.Errorf("Failed to access database: %s", err.Error())
		return
	}

	for i := range plannedTrans {
		pip, err := pipeline.NewClientPipeline(c.DB, &plannedTrans[i])
		if err != nil {
			continue
		}
		c.wg.Add(1)
		go func() {
			pip.Run()
			c.wg.Done()
		}()
	}
}

// Start starts the transfer controller service.
func (c *Controller) Start() error {
	c.logger = log.NewLogger(service.ControllerServiceName)

	pipeline.TransferInCount.SetLimit(c.DB.Conf.Controller.MaxTransfersIn)
	pipeline.TransferOutCount.SetLimit(c.DB.Conf.Controller.MaxTransfersOut)
	c.ticker = time.NewTicker(c.DB.Conf.Controller.Delay)
	c.state.Set(service.Running, "")

	c.listen()
	c.logger.Info("Controller started")

	return nil
}

// Stop stops the transfer controller service.
func (c *Controller) Stop(ctx context.Context) error {
	defer func() {
		c.state.Set(service.Offline, "")
		c.ticker.Stop()
	}()
	c.logger.Info("Shutting down controller...")

	c.cancel()

	select {
	case <-c.done:
		c.logger.Info("Shutdown complete")
		return nil
	case <-ctx.Done():
		c.logger.Info("Shutdown failed, forcing exit")
		return ctx.Err()
	}
}

// State returns the state of the transfer controller service.
func (c *Controller) State() *service.State {
	return &c.state
}
