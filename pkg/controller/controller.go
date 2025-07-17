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
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const ServiceName = "Controller"

// Controller is the service responsible for checking the database for new
// transfers at regular intervals, and starting those new transfers.
type Controller struct {
	DB *database.DB

	ticker *time.Ticker
	logger *log.Logger
	state  utils.State

	wg     *sync.WaitGroup
	done   chan struct{}
	ctx    context.Context
	cancel context.CancelFunc
}

func (c *Controller) listen() error {
	c.wg = &sync.WaitGroup{}
	c.done = make(chan struct{})
	c.ctx, c.cancel = context.WithCancel(context.Background())

	if conf.GlobalConfig.NodeID == "" {
		if err := c.DB.Exec("UPDATE transfers SET status=? WHERE owner=? AND status=?",
			types.StatusInterrupted, conf.GlobalConfig.GatewayName, types.StatusRunning,
		); err != nil {
			c.logger.Error("Failed to access database: %v", err)

			return fmt.Errorf("failed to access database: %w", err)
		}
	}

	go func() {
		for {
			select {
			case <-c.ctx.Done():
				c.wg.Wait()
				close(c.done)

				return
			case <-c.ticker.C:
				c.Run(c.wg, *c.logger)
			}
		}
	}()

	return nil
}

// Start starts the transfer controller service.
func (c *Controller) Start() error {
	if c.state.IsRunning() {
		return utils.ErrAlreadyRunning
	}

	c.logger = logging.NewLogger(ServiceName)

	config := &conf.GlobalConfig.Controller
	pipeline.List.SetLimits(config.MaxTransfersIn, config.MaxTransfersOut)
	c.ticker = time.NewTicker(config.Delay)

	if err := c.listen(); err != nil {
		c.state.Set(utils.StateError, err.Error())

		return err
	}

	c.logger.Info("Controller started")
	c.state.Set(utils.StateRunning, "")

	return nil
}

// Stop stops the transfer controller service.
func (c *Controller) Stop(ctx context.Context) error {
	if !c.state.IsRunning() {
		return utils.ErrNotRunning
	}

	defer c.ticker.Stop()
	c.logger.Info("Shutting down controller...")

	c.cancel()

	select {
	case <-c.done:
		c.logger.Info("Shutdown complete")
		c.state.Set(utils.StateOffline, "")

		return nil
	case <-ctx.Done():
		c.logger.Info("Shutdown failed, forcing exit")
		c.state.Set(utils.StateError, fmt.Sprintf("shutdown failed: %s", ctx.Err()))
		snmp.ReportServiceFailure(ServiceName, fmt.Errorf("shutdown failed: %w", ctx.Err()))

		return fmt.Errorf("shutdown failed: %w", ctx.Err())
	}
}

func (c *Controller) State() (utils.StateCode, string) {
	return c.state.Get()
}
