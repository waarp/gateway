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
	remainingTries int8,
	nextRetryDelay int32,
	retryIncrementFactor float32,
	transferInfos map[string]any,
) (*model.Transfer, error) {
	trans := &model.Transfer{
		SrcFilename:          srcFilename,
		DestFilename:         dstFilename,
		RemoteAccountID:      utils.NewNullInt64(account.ID),
		ClientID:             utils.NewNullInt64(client.ID),
		RuleID:               rule.ID,
		Start:                date,
		RemainingTries:       remainingTries,
		NextRetryDelay:       nextRetryDelay,
		RetryIncrementFactor: retryIncrementFactor,
		TransferInfo:         transferInfos,
	}

	if date.IsZero() {
		trans.Start = time.Now()
	}

	if err := db.Insert(trans).Run(); err != nil {
		return nil, fmt.Errorf("failed to insert transfer: %w", err)
	}

	return trans, nil
}

func RegisterNewTransfer(db *database.DB,
	filename string,
	rule *model.Rule,
	account *model.LocalAccount,
	dueDate time.Time,
	transferInfos map[string]any,
) (*model.Transfer, error) {
	trans := &model.Transfer{
		Status:         types.StatusAvailable,
		LocalAccountID: utils.NewNullInt64(account.ID),
		RuleID:         rule.ID,
		Start:          dueDate,
		TransferInfo:   transferInfos,
	}

	if rule.IsSend {
		trans.SrcFilename = filename
	} else {
		trans.DestFilename = filename
	}

	if err := db.Insert(trans).Run(); err != nil {
		return nil, fmt.Errorf("failed to insert transfer: %w", err)
	}

	return trans, nil
}

func PauseTransfer(ctx context.Context, db database.Access, view *model.NormalizedTransferView,
) error {
	if view.Status != types.StatusRunning {
		return ErrPauseTransferNotRunning
	}

	if pip := pipeline.List.Get(view.ID); pip != nil {
		if err := pip.Pause(ctx); err != nil {
			return fmt.Errorf("failed to pause transfer: %w", err)
		}

		return nil
	}

	var transfer model.Transfer
	if err := db.Get(&transfer, "id=?", view.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve transfer: %w", err)
	}

	transfer.Status = types.StatusPaused
	if err := db.Update(&transfer).Run(); err != nil {
		return fmt.Errorf("failed to update transfer: %w", err)
	}

	return nil
}

func ResumeTransfer(db database.Access, view *model.NormalizedTransferView) error {
	if err := view.Resume(db, time.Now()); err != nil {
		return fmt.Errorf("failed to resume transfer: %w", err)
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
	if err := trans.MoveToHistory(db, logging.Discard(), time.Time{}); err != nil {
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

	if err := db.Insert(newTransfer).Run(); err != nil {
		return nil, fmt.Errorf("failed to insert transfer: %w", err)
	}

	return newTransfer, nil
}
