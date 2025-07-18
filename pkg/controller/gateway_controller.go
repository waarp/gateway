package controller

import (
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// Run checks the database for new planned transfers and starts
// them, as long as there are available transfer slots.
func (c *Controller) Run(wg *sync.WaitGroup, logger log.Logger) {
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

func (c *Controller) retrieveTransfers() (model.Transfers, error) {
	var transfers model.Transfers

	if tErr := c.DB.Transaction(func(ses *database.Session) error {
		lim := pipeline.List.GetAvailableOut()
		if lim == 0 {
			return nil // cannot start more transfers, limit has been reached
		}

		query := ses.SelectForUpdate(&transfers).Owner().
			Where("status=?", types.StatusPlanned).
			Where("remote_account_id IS NOT NULL").
			Where("start<?", time.Now().UTC())

		if lim <= math.MaxInt {
			query.Limit(int(lim), 0)
		}

		if err := query.Run(); err != nil {
			return fmt.Errorf("failed to retrieve transfers to execute: %w", err)
		}

		for _, trans := range transfers {
			trans.Status = types.StatusRunning
			if err := ses.Update(trans).Cols("status").Run(); err != nil {
				c.logger.Error("Failed to update status of transfer %d: %v", trans.ID, err)
				trans.Status = types.StatusError
			}
		}

		return nil
	}); tErr != nil {
		return nil, fmt.Errorf("controller database error: %w", tErr)
	}

	return slices.DeleteFunc(transfers, func(t *model.Transfer) bool {
		return t.Status != types.StatusRunning
	}), nil
}
