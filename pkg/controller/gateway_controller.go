package controller

import (
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

type GatewayController struct {
	DB      *database.DB
	logger  *log.Logger
	wasDown bool
}

// Run checks the database for new planned transfers and starts
// them, as long as there are available transfer slots.
func (c *GatewayController) Run(wg *sync.WaitGroup) {
	if c.checkIsDBDown() {
		return
	}

	plannedTrans, err := c.retrieveTransfers()
	if err != nil {
		return
	}

	for i := range plannedTrans {
		trans := &plannedTrans[i]

		pip, err := pipeline.NewClientPipeline(c.DB, trans)
		if err != nil {
			continue
		}

		wg.Add(1)

		go func() {
			if err := pip.Run(); err != nil {
				c.logger.Errorf("Transfer n°%d failed: %v", trans.ID, err)
			}

			wg.Done()
		}()
	}
}

func (c *GatewayController) checkIsDBDown() bool {
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

func (c *GatewayController) retrieveTransfers() (model.Transfers, error) {
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
			// c.logger.Errorf("Failed to access database: %s", err.Error())

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
