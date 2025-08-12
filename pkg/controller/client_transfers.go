package controller

import (
	"fmt"
	"math"
	"slices"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// Run checks the database for new planned transfers and starts
// them, as long as there are available transfer slots.
func (c *Controller) Run() {
	plannedTrans, dbErr := c.retrieveTransfers()
	if dbErr != nil {
		c.logger.Errorf("Failed to retrieve the transfers to run: %v", dbErr)

		return
	}

	for _, trans := range plannedTrans {
		pip, pipErr := NewClientPipeline(c.DB, trans)
		if pipErr != nil {
			continue
		}

		c.wg.Add(1)

		go func(t *model.Transfer) {
			defer c.wg.Done()

			if t.IsServer() {
				pip.Pip.SetError(types.TeExpired, "Transfer expired")

				return
			}

			if err := pip.Run(); err != nil {
				c.logger.Errorf("Transfer n°%d failed: %v", t.ID, err)
			}
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
			In("status", types.StatusPlanned, types.StatusInterrupted, types.StatusError).
			Where("remote_account_id IS NOT NULL").
			Where("next_retry <= ?", time.Now().UTC())

		if lim <= math.MaxInt {
			query.Limit(int(lim), 0)
		}

		if err := query.Run(); err != nil {
			return fmt.Errorf("failed to retrieve transfers to execute: %w", err)
		}

		for _, trans := range transfers {
			trans.Status = types.StatusRunning

			if trans.RemainingTries > 0 {
				trans.RemainingTries--
				trans.NextRetry = time.Time{}
			}

			if err := ses.Update(trans).Cols("status", "remaining_retries",
				"next_retry").Run(); err != nil {
				c.logger.Errorf("Failed to update status of transfer %d: %v", trans.ID, err)
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
