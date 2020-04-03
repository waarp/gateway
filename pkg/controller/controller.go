// Package controller defines the controller module whose purpose is to
// periodically scan the database for new transfers to launch.
package controller

import (
	"context"
	"sync"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/pipeline"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/go-xorm/builder"
)

// ServiceName is the name of the controller service
const ServiceName = "executor"

// Controller is the service responsible for checking the database for new
// transfers at regular intervals, and starting those new transfers.
type Controller struct {
	Conf *conf.ServerConfig
	Db   *database.Db

	ticker *time.Ticker
	logger *log.Logger
	state  service.State

	shutdown chan bool
}

func (c *Controller) checkIsDbDown() bool {
	owner := builder.Eq{"owner": database.Owner}
	statusDown := builder.Eq{"status": model.StatusRunning}
	filtersDown := database.Filters{
		Conditions: builder.And(owner, statusDown),
	}

	if st, _ := c.Db.State().Get(); st != service.Running {
		return true
	}
	runningTrans := []model.Transfer{}
	if err := c.Db.Select(&runningTrans, &filtersDown); err != nil {
		c.logger.Errorf("Failed to access database: %s", err.Error())
		return true
	}
	for _, trans := range runningTrans {
		trans.Status = model.StatusInterrupted
		if err := trans.Update(c.Db); err != nil {
			c.logger.Errorf("Failed to access database: %s", err.Error())
			return false
		}
	}
	return true
}

func (c *Controller) listen() {
	owner := builder.Eq{"owner": database.Owner}
	status := builder.Eq{"status": model.StatusPlanned}
	client := builder.Eq{"is_server": false}
	wg := sync.WaitGroup{}
	c.shutdown = make(chan bool)

	go func() {
		isDbDown := false
		for {
			select {
			case <-c.shutdown:
				wg.Wait()
				close(c.shutdown)
				return
			case <-c.ticker.C:
			}

			if isDbDown {
				isDbDown = c.checkIsDbDown()
			}

			start := builder.Lte{"start": time.Now()}
			filters := database.Filters{
				Conditions: builder.And(owner, start, status, client),
			}

			plannedTrans := []model.Transfer{}
			if err := c.Db.Select(&plannedTrans, &filters); err != nil {
				c.logger.Errorf("Failed to access database: %s", err.Error())
				if err == database.ErrServiceUnavailable {
					isDbDown = true
				}
				continue
			}

			for _, trans := range plannedTrans {
				exe, err := c.getExecutor(trans)
				if err == pipeline.ErrLimitReached {
					break
				}
				if err != nil {
					continue
				}

				wg.Add(1)
				go func() {
					exe.Run()
					wg.Done()
				}()
			}
		}
	}()
}

func (c *Controller) getExecutor(trans model.Transfer) (*executor.Executor, error) {
	stream, err := pipeline.NewTransferStream(c.logger, c.Db, "", trans)
	if err != nil {
		c.logger.Errorf("Failed to create transfer stream: %s", err.Error())
		return nil, err
	}

	exe := &executor.Executor{TransferStream: stream}
	return exe, nil
}

// Start starts the transfer controller service.
func (c *Controller) Start() error {
	if c.logger == nil {
		c.logger = log.NewLogger(ServiceName, c.Conf.Log)
	}

	pipeline.TransferInCount.SetLimit(c.Conf.Controller.MaxTransfersIn)
	pipeline.TransferOutCount.SetLimit(c.Conf.Controller.MaxTransfersOut)
	c.ticker = time.NewTicker(c.Conf.Controller.Delay)
	c.state.Set(service.Running, "")

	c.listen()

	return nil
}

// Stop stops the transfer controller service.
func (c *Controller) Stop(ctx context.Context) error {
	defer c.ticker.Stop()
	c.state.Set(service.Offline, "")
	c.shutdown <- true

	var transfers []model.Transfer
	filters := &database.Filters{
		Conditions: builder.And(builder.Eq{"is_server": false},
			builder.NotIn("status", model.StatusInterrupted, model.StatusPaused)),
	}
	if err := c.Db.Select(&transfers, filters); err != nil {
		c.logger.Criticalf("Failed to retrieve ongoing transfers: %s", err)
		return err
	}
	for _, trans := range transfers {
		pipeline.Signals.SendSignal(trans.ID, model.SignalShutdown)
	}

	select {
	case <-c.shutdown:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// State returns the state of the transfer controller service.
func (c *Controller) State() *service.State {
	return &c.state
}
