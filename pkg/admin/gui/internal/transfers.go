package internal

import (
	"errors"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrPauseTransferNotRunning = errors.New("cannot pause a non-running transfer")
	ErrResumeTransferNotPaused = errors.New("cannot resume a non-interrupted transfer")
	ErrCancelTransferFinished  = errors.New("cannot cancel a finished transfer")
)

type TransferListQuery struct {
	query     *database.SelectQuery
	transfers model.NormalizedTransfers
}

func (q *TransferListQuery) Status(statuses ...types.TransferStatus) *TransferListQuery {
	q.query.In("status", utils.AsAny(statuses)...)

	return q
}

func (q *TransferListQuery) Requester(name string) *TransferListQuery {
	q.query.Where("account=?", name)

	return q
}

func (q *TransferListQuery) Requested(name string) *TransferListQuery {
	q.query.Where("agent=?", name)

	return q
}

func (q *TransferListQuery) Rule(name string, isSend bool) *TransferListQuery {
	q.query.Where("rule=?", name).Where("is_send=?", isSend)

	return q
}

func (q *TransferListQuery) Date(from, to time.Time) *TransferListQuery {
	q.query.Where("start>=?", from.UTC()).Where("start<=?", to.UTC())

	return q
}

func (q *TransferListQuery) File(file string) *TransferListQuery {
	q.query.Where("src_filename=? OR dest_filename=?", file, file)

	return q
}

// FilePattern adds a condition which will match any transfer whose filename
// contains the provided substring.
func (q *TransferListQuery) FilePattern(substring string) *TransferListQuery {
	const escapeChar = "\033"

	replacer := strings.NewReplacer(
		"%", escapeChar+"%",
		"_", escapeChar+"_",
	)

	substring = replacer.Replace(substring)
	substring = "%" + substring + "%"

	q.query.Where("src_filename LIKE ? ESCAPE ? OR dest_filename LIKE ? ESCAPE ?",
		substring, escapeChar, substring, escapeChar)

	return q
}

func (q *TransferListQuery) Count() (uint64, error) {
	return q.query.Count()
}

func (q *TransferListQuery) Limit(limit, offset int) *TransferListQuery {
	q.query.Limit(limit, offset)

	return q
}

func (q *TransferListQuery) Run() ([]*model.NormalizedTransferView, error) {
	return q.transfers, q.query.Run()
}

func GetTransfer(db database.ReadAccess, id int64) (*model.Transfer, error) {
	var transfer model.Transfer

	return &transfer, db.Get(&transfer, "id=?", id).Run()
}

func StartTransferQuery(db database.ReadAccess, orderBy string, asc bool) *TransferListQuery {
	list := &TransferListQuery{}
	list.query = db.Select(&list.transfers).Owner().OrderBy(orderBy, asc)

	return list
}
