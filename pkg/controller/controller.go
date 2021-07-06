// Package controller defines the controller module whose purpose is to
// periodically scan the database for new transfers to launch.
package controller

import (
	"context"
	"errors"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model/types"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
)

// ServiceName is the name of the controller service.
const ServiceName = "Controller"

// Controller is the service responsible for checking the database for new
// transfers at regular intervals, and starting those new transfers.
type Controller struct {
	Conf *conf.ServerConfig
	DB   *database.DB

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

func (c *Controller) retrieveTransfers() (model.Transfers, error) {
	var transfers model.Transfers
	if tErr := c.DB.Transaction(func(ses *database.Session) database.Error {
		query := ses.SelectForUpdate(&transfers).Where("owner=? AND status=? AND "+
			"is_server=? AND start<?", database.Owner, types.StatusPlanned, false,
			time.Now().UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano))
		lim := pipeline.TransferOutCount.GetLimit()
		if lim > 0 {
			query.Limit(int(lim-pipeline.TransferOutCount.Get()), 0)
		}

		if err := query.Run(); err != nil {
			c.logger.Errorf("Failed to access database: %s", err.Error())
			return err
		}

		for i := range transfers {
			transfers[i].Status = types.StatusRunning
			if err := ses.Update(&transfers[i]).Cols("status").Run(); err != nil {
				return err
			}
		}
		return nil
	}); tErr != nil {
		return nil, tErr
	}
	return transfers, nil
}

// startNewTransfers checks the database for new planned transfers and starts
// them, as long as there are available transfer slots.
func (c *Controller) startNewTransfers() {

	if c.checkIsDBDown() {
		return
	}

	plannedTrans, err := c.retrieveTransfers()
	if err != nil {
		return
	}

	for _, trans := range plannedTrans {
		exe, err := c.getExecutor(trans)
		if err != nil {
			if errors.Is(err, pipeline.ErrLimitReached) {
				break
			}
			continue
		}

		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			exe.Run()
		}()
	}
}

func (c *Controller) getExecutor(trans model.Transfer) (*executor.Executor, error) {
	paths := pipeline.Paths{PathsConfig: c.Conf.Paths}

	stream, err := pipeline.NewTransferStream(c.ctx, c.logger, c.DB, paths, &trans)
	if err != nil {
		c.logger.Errorf("Failed to create transfer stream: %s", err.Error())
		return nil, err
	}

	exe := &executor.Executor{TransferStream: stream}

	return exe, nil
}

// Start starts the transfer controller service.
func (c *Controller) Start() error {
	c.logger = log.NewLogger(ServiceName)

	pipeline.TransferInCount.SetLimit(c.Conf.Controller.MaxTransfersIn)
	pipeline.TransferOutCount.SetLimit(c.Conf.Controller.MaxTransfersOut)
	c.ticker = time.NewTicker(c.Conf.Controller.Delay)
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
