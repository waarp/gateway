package controller

import (
	"fmt"
	"math"
	"sync"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

type GatewayController struct {
	DB      *database.DB
	wasDown bool
}

// Run checks the database for new planned transfers and starts
// them, as long as there are available transfer slots.
func (c *GatewayController) Run(wg *sync.WaitGroup, logger log.Logger) {
	if c.checkIsDBDown(logger) {
		return
	}

	plannedTrans, dbErr := c.retrieveTransfers()
	if dbErr != nil {
		logger.Error("Failed to retrieve the transfers to run: %v", dbErr)

		return
	}

	for _, trans := range plannedTrans {
		pip, pipErr := NewClientPipeline(c.DB, trans)
		if pipErr != nil {
			continue
		}

		wg.Add(1)

		go func(t *model.Transfer) {
			if err := pip.Run(); err != nil {
				logger.Error("Transfer n°%d failed: %v", t.ID, err)
			}

			wg.Done()
		}(trans)
	}
}

func (c *GatewayController) checkIsDBDown(logger log.Logger) bool {
	if !c.wasDown {
		return false
	}

	if err := c.DB.Exec("UPDATE transfers SET status=? WHERE owner=? AND status=?",
		types.StatusInterrupted, conf.GlobalConfig.GatewayName, types.StatusRunning,
	); err != nil {
		logger.Error("Failed to access database: %s", err.Error())

		return true
	}

	c.wasDown = false

	return false
}

func (c *GatewayController) retrieveTransfers() (model.Transfers, error) {
	var transfers model.Transfers

	if tErr := c.DB.Transaction(func(ses *database.Session) error {
		lim := pipeline.List.GetAvailableOut()
		if lim == 0 {
			return nil // cannot start more transfers, limit has been reached
		}

		query := ses.SelectForUpdate(&transfers).Where("owner=? AND status=? AND "+
			"remote_account_id IS NOT NULL AND start<?", conf.GlobalConfig.GatewayName,
			types.StatusPlanned, time.Now().UTC())

		if lim <= math.MaxInt {
			query.Limit(int(lim), 0)
		}

		if err := query.Run(); err != nil {
			return fmt.Errorf("failed to retrieve transfers to execute: %w", err)
		}

		for _, trans := range transfers {
			trans.Status = types.StatusRunning
			if err := ses.Update(trans).Cols("status").Run(); err != nil {
				return fmt.Errorf("failed to update transfer status: %w", err)
			}
		}

		return nil
	}); tErr != nil {
		return nil, fmt.Errorf("controller database error: %w", tErr)
	}

	return transfers, nil
}
