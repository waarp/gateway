package internal

import (
	"context"
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func InsertNewTransfer(db *database.DB,
	srcFilename, dstFilename string,
	rule *model.Rule,
	account *model.RemoteAccount,
	client *model.Client,
	date time.Time,
	transferInfos map[string]any,
) error {
	trans := &model.Transfer{
		SrcFilename:     srcFilename,
		DestFilename:    dstFilename,
		RemoteAccountID: utils.NewNullInt64(account.ID),
		ClientID:        utils.NewNullInt64(client.ID),
		RuleID:          rule.ID,
		Start:           date,
	}

	if date.IsZero() {
		trans.Start = time.Now()
	}

	return db.Transaction(func(ses *database.Session) error {
		if err := ses.Insert(trans).Run(); err != nil {
			return fmt.Errorf("failed to insert transfer: %w", err)
		}

		if len(transferInfos) != 0 {
			if err := trans.SetTransferInfo(ses, transferInfos); err != nil {
				return fmt.Errorf("failed to set transfer info: %w", err)
			}
		}

		return nil
	})
}

func PauseTransfer(ctx context.Context, db database.Access, transfer *model.NormalizedTransferView,
) error {
	if transfer.Status != types.StatusRunning {
		return ErrPauseTransferNotRunning
	}

	if pip := pipeline.List.Get(transfer.ID); pip != nil {
		if err := pip.Pause(ctx); err != nil {
			return fmt.Errorf("failed to pause transfer: %w", err)
		}

		return nil
	}

	transfer.Status = types.StatusPaused
	if err := db.Update(transfer).Run(); err != nil {
		return fmt.Errorf("failed to update transfer: %w", err)
	}

	return nil
}

func ResumeTransfer(db database.Access, transfer *model.NormalizedTransferView) error {
	if !transfer.Status.IsOneOf(types.StatusPaused, types.StatusError, types.StatusInterrupted) {
		return ErrResumeTransferNotPaused
	}

	transfer.Status = types.StatusPlanned
	if err := db.Update(transfer).Run(); err != nil {
		return fmt.Errorf("failed to update transfer: %w", err)
	}

	return nil
}

func CancelTransfer(ctx context.Context, db *database.DB, transfer *model.NormalizedTransferView,
) error {
	if transfer.Status.IsOneOf(types.StatusCancelled, types.StatusDone) {
		return ErrCancelTransferFinished
	}

	if pip := pipeline.List.Get(transfer.ID); pip != nil {
		if err := pip.Cancel(ctx); err != nil {
			return fmt.Errorf("failed to cancel transfer: %w", err)
		}

		return nil
	}

	var trans model.Transfer
	if err := db.Get(&trans, "id=?", transfer.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve transfer: %w", err)
	}

	trans.Status = types.StatusCancelled
	if err := trans.MoveToHistory(db, logging.Discard(), time.Now()); err != nil {
		return fmt.Errorf("failed to move transfer to history: %w", err)
	}

	return nil
}

func ReprogramTransfer(db *database.DB, transfer *model.NormalizedTransferView,
	date time.Time,
) (*model.Transfer, error) {
	newTransfer, copyErr := transfer.Restart(db, date)
	if copyErr != nil {
		return nil, fmt.Errorf("failed to reprogram transfer: %w", copyErr)
	}

	infos, infErr := transfer.GetTransferInfo(db)
	if infErr != nil {
		return nil, fmt.Errorf("failed to get transfer info: %w", infErr)
	}

	return newTransfer, db.Transaction(func(ses *database.Session) error {
		if err := ses.Insert(newTransfer).Run(); err != nil {
			return fmt.Errorf("failed to insert transfer: %w", err)
		}

		if err := newTransfer.SetTransferInfo(ses, infos); err != nil {
			return fmt.Errorf("failed to set transfer info: %w", err)
		}

		return nil
	})
}
