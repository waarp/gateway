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
	"github.com/go-xorm/builder"
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
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Controller) checkIsDBDown() bool {
	owner := builder.Eq{"owner": database.Owner}
	statusDown := builder.Eq{"status": types.StatusRunning}
	filtersDown := database.Filters{
		Conditions: builder.And(owner, statusDown),
	}

	if st, _ := c.DB.State().Get(); st != service.Running {
		return true
	}

	var runningTrans []model.Transfer
	if err := c.DB.Select(&runningTrans, &filtersDown); err != nil {
		c.logger.Errorf("Failed to access database: %s", err.Error())
		return true
	}

	for _, t := range runningTrans {
		trans := t
		trans.Status = types.StatusInterrupted
		if err := c.DB.Update(&trans); err != nil {
			c.logger.Errorf("Failed to access database: %s", err.Error())
			return true
		}
	}

	return false
}

func (c *Controller) listen() {
	c.wg = &sync.WaitGroup{}
	c.ctx, c.cancel = context.WithCancel(context.Background())

	go func() {
		for {
			select {
			case <-c.ctx.Done():
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

	owner := builder.Eq{"owner": database.Owner}
	status := builder.Eq{"status": types.StatusPlanned}
	client := builder.Eq{"is_server": false}
	start := builder.Lte{"start": time.Now()}
	filters := database.Filters{
		Conditions: builder.And(owner, start, status, client),
	}

	plannedTrans := []model.Transfer{}
	if err := c.DB.Select(&plannedTrans, &filters); err != nil {
		c.logger.Errorf("Failed to access database: %s", err.Error())
		return
	}

	for _, trans := range plannedTrans {
		exe, err := c.getExecutor(trans)
		if errors.Is(err, pipeline.ErrLimitReached) {
			break
		}

		if err != nil {
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

	exe := &executor.Executor{TransferStream: stream, R66Home: c.Conf.Controller.R66Home}

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

	finished := make(chan struct{})

	go func() {
		c.wg.Wait()
		close(finished)
	}()

	select {
	case <-finished:
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
