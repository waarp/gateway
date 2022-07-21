// Package controller defines the controller module whose purpose is to
// periodically scan the database for new transfers to launch.
package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// Controller is the service responsible for checking the database for new
// transfers at regular intervals, and starting those new transfers.
type Controller struct {
	Action func(*sync.WaitGroup, log.Logger)

	ticker *time.Ticker
	logger *log.Logger
	state  service.State

	wg     *sync.WaitGroup
	done   chan struct{}
	ctx    context.Context //nolint:containedctx //FIXME move the context to a function parameter
	cancel context.CancelFunc
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
				c.Action(c.wg, *c.logger)
			}
		}
	}()
}

// Start starts the transfer controller service.
func (c *Controller) Start() error {
	c.logger = conf.GetLogger(service.ControllerServiceName)

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
