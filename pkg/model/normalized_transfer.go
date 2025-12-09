package model

import (
	"errors"
	"fmt"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrResumeDone    = errors.New("cannot resume a transfer that is already done")
	ErrResumeRunning = errors.New("cannot resume a transfer that is already running")
	ErrResumeServer  = errors.New("cannot resume server transfers")
)

type NormalizedTransferView struct {
	HistoryEntry `xorm:"extends"`

	IsTransfer           bool      `xorm:"BOOL 'is_transfer'"`
	RemainingTries       int8      `xorm:"remaining_tries"`
	NextRetryDelay       int32     `xorm:"next_retry_delay"`
	RetryIncrementFactor float32   `xorm:"retry_increment_factor"`
	NextRetry            time.Time `xorm:"next_retry DATETIME(6) UTC"`
}

func (*NormalizedTransferView) TableName() string   { return ViewNormalizedTransfers }
func (*NormalizedTransferView) Appellation() string { return "normalized transfer" }
func (n *NormalizedTransferView) GetID() int64      { return n.ID }

// BeforeWrite always returns an error because writing is not allowed on views.
func (n *NormalizedTransferView) BeforeWrite(database.Access) error {
	return database.NewInternalError(errWriteOnView)
}

// BeforeDelete always returns an error because deleting is not allowed on views.
func (n *NormalizedTransferView) BeforeDelete(database.Access) error {
	return database.NewInternalError(errWriteOnView)
}

func (n *NormalizedTransferView) AfterRead(db database.ReadAccess) error {
	var owner transferInfoOwner = &n.HistoryEntry
	if n.IsTransfer {
		owner = &Transfer{ID: n.ID}
	}

	infos, err := getTransferInfo(db, owner)
	if err != nil {
		return err
	}

	n.TransferInfo = infos

	return nil
}

func (n *NormalizedTransferView) getTransInfoCondition() (string, int64) {
	if n.IsTransfer {
		return (&Transfer{ID: n.ID}).getTransInfoCondition()
	}

	return n.HistoryEntry.getTransInfoCondition()
}

func (n *NormalizedTransferView) setTransInfoOwner(info *TransferInfo) {
	if n.IsTransfer {
		(&Transfer{ID: n.ID}).setTransInfoOwner(info)
	} else {
		(&HistoryEntry{ID: n.ID}).setTransInfoOwner(info)
	}
}

func (n *NormalizedTransferView) CheckResumable() error {
	if !n.IsTransfer {
		return ErrResumeDone
	}

	trans := &Transfer{
		ID:           n.ID,
		Status:       n.Status,
		TransferInfo: n.TransferInfo,
	}

	if n.IsServer {
		trans.LocalAccountID = utils.NewNullInt64(-1)
	} else {
		trans.ClientID = utils.NewNullInt64(-1)
		trans.RemoteAccountID = utils.NewNullInt64(-1)
	}

	return trans.CheckResumable()
}

func (n *NormalizedTransferView) Resume(db database.Access, when time.Time) error {
	if !n.IsTransfer {
		return ErrResumeDone
	}

	var dbTrans Transfer
	if err := db.Get(&dbTrans, "id=?", n.ID).Run(); err != nil {
		return fmt.Errorf("failed to retrieve transfer: %w", err)
	}

	if err := dbTrans.Resume(db, when); err != nil {
		return err
	}

	n.Status = dbTrans.Status
	n.NextRetry = dbTrans.NextRetry
	n.ErrCode = dbTrans.ErrCode
	n.ErrDetails = dbTrans.ErrDetails

	return nil
}
