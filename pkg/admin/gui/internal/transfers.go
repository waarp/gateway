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

// likeEscape is the escape character of the LIKE patterns of the file
// searches.
const likeEscape = "\033"

// likePattern is the LIKE pattern that matches any text containing the
// substring, its wildcards escaped.
func likePattern(substring string) string {
	replacer := strings.NewReplacer(
		"%", likeEscape+"%",
		"_", likeEscape+"_",
	)

	return "%" + replacer.Replace(substring) + "%"
}

// FilePattern adds a condition which will match any transfer whose filename
// contains the provided substring.
func (q *TransferListQuery) FilePattern(substring string) *TransferListQuery {
	pattern := likePattern(substring)

	q.query.Where("src_filename LIKE ? ESCAPE ? OR dest_filename LIKE ? ESCAPE ?",
		pattern, likeEscape, pattern, likeEscape)

	return q
}

// Search adds a condition which will match any transfer whose filename
// contains the text, or whose follow ID is the text (the condition of the
// followID filter of the REST API).
func (q *TransferListQuery) Search(text string) *TransferListQuery {
	pattern := likePattern(text)

	q.query.Where("(src_filename LIKE ? ESCAPE ? OR dest_filename LIKE ? ESCAPE ? OR "+
		"id IN (SELECT COALESCE(transfer_id, history_id) FROM transfer_info WHERE name=? AND value=?))",
		pattern, likeEscape, pattern, likeEscape, model.FollowID, text)

	return q
}

// RuleName adds a condition on the name of the rule, in both directions.
func (q *TransferListQuery) RuleName(name string) *TransferListQuery {
	q.query.Where("rule=?", name)

	return q
}

// Direction adds a condition on the direction of the transfers.
func (q *TransferListQuery) Direction(isSend bool) *TransferListQuery {
	q.query.Where("is_send=?", isSend)

	return q
}

// Protocol adds a condition on the protocol of the transfers.
func (q *TransferListQuery) Protocol(protocol string) *TransferListQuery {
	q.query.Where("protocol=?", protocol)

	return q
}

// StartedFrom adds a condition on the start date: at or after the given time.
func (q *TransferListQuery) StartedFrom(from time.Time) *TransferListQuery {
	q.query.Where("start>=?", from.UTC())

	return q
}

// StartedUntil adds a condition on the start date: at or before the given
// time.
func (q *TransferListQuery) StartedUntil(to time.Time) *TransferListQuery {
	q.query.Where("start<=?", to.UTC())

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

func GetNormalizedTransferView(db database.ReadAccess, id int64) (*model.NormalizedTransferView, error) {
	var transfer model.NormalizedTransferView

	return &transfer, db.Get(&transfer, "id=?", id).Run()
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
