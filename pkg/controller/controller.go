// Package controller defines the controller module whose purpose is to
// periodically scan the database for new transfers to launch.
package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// Controller is the service responsible for checking the database for new
// transfers at regular intervals, and starting those new transfers.
type Controller struct {
	DB *database.DB

	ticker *time.Ticker
	logger *log.Logger
	state  service.State

	wg      *sync.WaitGroup
	done    chan struct{}
	ctx     context.Context //nolint:containedctx //FIXME move the context to a function parameter
	cancel  context.CancelFunc
	wasDown bool
}

func (c *Controller) checkIsDBDown() bool {
	if st, _ := c.DB.State().Get(); st != service.Running {
		c.wasDown = true

		return true
	}

	if !c.wasDown {
		return false
	}

	query := c.DB.UpdateAll(&model.Transfer{}, database.UpdVals{"status": types.StatusInterrupted},
		"owner=? AND status=?", conf.GlobalConfig.GatewayName, types.StatusRunning)
	if err := query.Run(); err != nil {
		c.logger.Errorf("Failed to access database: %s", err.Error())

		return true
	}

	c.wasDown = false

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
		lim, hasLimit := pipeline.TransferOutCount.GetAvailable()
		if hasLimit && lim == 0 {
			return nil // cannot start more transfers, limit has been reached
		}

		query := ses.SelectForUpdate(&transfers).Where("owner=? AND status=? AND "+
			"is_server=? AND start<?", conf.GlobalConfig.GatewayName,
			types.StatusPlanned, false, time.Now().UTC().Truncate(time.Microsecond).
				Format(time.RFC3339Nano)).Limit(lim, 0)

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

	for i := range plannedTrans {
		pip, err := pipeline.NewClientPipeline(c.DB, &plannedTrans[i])
		if err != nil {
			continue
		}

		c.wg.Add(1)

		go func() {
			pip.Run() //nolint:errcheck //error is irrelevant here
			c.wg.Done()
		}()
	}
}

// Start starts the transfer controller service.
func (c *Controller) Start() error {
	c.logger = log.NewLogger(service.ControllerServiceName)

	config := &conf.GlobalConfig.Controller
	pipeline.TransferInCount.SetLimit(config.MaxTransfersIn)
	pipeline.TransferOutCount.SetLimit(config.MaxTransfersOut)
	c.ticker = time.NewTicker(config.Delay)
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

		if err := ctx.Err(); err != nil {
			return fmt.Errorf("shutdown done with error: %w", err)
		}

		return nil
	}
}

// State returns the state of the transfer controller service.
func (c *Controller) State() *service.State {
	return &c.state
}
