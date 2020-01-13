// Package controller defines the controller module whose purpose is to
// periodically scan the database for new transfers to launch.
package controller

import (
	"context"
	"time"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/executor"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
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

	ticker   time.Ticker
	logger   *log.Logger
	state    service.State
	shutdown chan bool
}

func (c *Controller) listen(run func(model.Transfer)) {
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
				go run(trans)
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

	exe := executor.Executor{
		Db:       c.Db,
		Logger:   log.NewLogger("executor", c.Conf.Log),
		R66Home:  c.Conf.Controller.R66Home,
		Shutdown: c.shutdown,
	}
	c.listen(exe.Run)

	return nil
}

// Stop stops the transfer controller service.
func (c *Controller) Stop(_ context.Context) error {
	close(c.shutdown)
	c.state.Set(service.Offline, "")
	c.ticker.Stop()
	return nil
}

// State returns the state of the transfer controller service.
func (c *Controller) State() *service.State {
	return &c.state
}
