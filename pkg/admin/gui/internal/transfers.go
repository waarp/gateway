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

// FilePattern adds a condition on the transfer file name using the provided
// pattern. The pattern must be in Unix glob format, which accepts the following
// wildcards:
// - "?" for matching any character exactly once
// - "*" for matching a string of zero or more characters
//
// That glob will then be converted to an SQL pattern, with the equivalent SQL
// wildcards.
func (q *TransferListQuery) FilePattern(glob string) *TransferListQuery {
	const escapeChar = "!"

	// First, escape SQL wildcard characters
	pattern := strings.ReplaceAll(glob, "%", escapeChar+"%")
	pattern = strings.ReplaceAll(pattern, "_", escapeChar+"_")
	// Then replace glob wildcards with SQL wildcards
	pattern = strings.ReplaceAll(pattern, "*", "%")
	pattern = strings.ReplaceAll(pattern, "?", "_")

	q.query.Where("src_filename LIKE ? ESCAPE ? OR dest_filename LIKE ? ESCAPE ?",
		pattern, escapeChar, pattern, escapeChar)

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
