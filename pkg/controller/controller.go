// Package controller defines the controller module whose purpose is to
// periodically scan the database for new transfers to launch.
package controller

import (
	"context"
	"fmt"
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

	ticker time.Ticker
	logger *log.Logger
	state  service.State

	wg        *sync.WaitGroup
	pool      chan model.Transfer
	executors []executor.Executor
}

func (c *Controller) listen() {
	owner := builder.Eq{"owner": database.Owner}
	start := builder.Lte{"start": time.Now()}
	planned := builder.Eq{"status": model.StatusPlanned}
	client := builder.Eq{"is_server": false}
	filters := database.Filters{
		Conditions: builder.And(owner, start, planned, client),
	}

	go func() {
		for {
			if s, _ := c.state.Get(); s != service.Running {
				return
			}

			<-c.ticker.C
			newTrans := []model.Transfer{}
			if err := c.Db.Select(&newTrans, &filters); err != nil {
				c.logger.Error(err.Error())
				continue
			}
			for _, trans := range newTrans {
				c.pool <- trans
			}
		}
	}()
}

// Start starts the transfer controller service.
func (c *Controller) Start() error {
	if c.logger == nil {
		c.logger = log.NewLogger(ServiceName, c.Conf.Log)
	}

	c.ticker = *time.NewTicker(c.Conf.Controller.Delay)
	c.state.Set(service.Running, "")

	c.wg = new(sync.WaitGroup)
	c.executors = make([]executor.Executor, 10)
	for i := range c.executors {
		c.executors[i] = executor.Executor{
			Db:        c.Db,
			Logger:    log.NewLogger(fmt.Sprintf("executor%d", i), c.Conf.Log),
			R66Home:   c.Conf.Controller.R66Home,
			Transfers: c.pool,
		}
		c.executors[i].Run(c.wg)
	}
	c.listen()

	return nil
}

// Stop stops the transfer controller service.
func (c *Controller) Stop(ctx context.Context) error {
	close(c.pool)
	c.state.Set(service.Offline, "")
	c.ticker.Stop()

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
	case <-func() chan bool {
		c.wg.Wait()
		b := make(chan bool)
		close(b)
		return b
	}():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// State returns the state of the transfer controller service.
func (c *Controller) State() *service.State {
	return &c.state
}
