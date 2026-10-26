package internal

import (
	"errors"
	"fmt"
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

// Client keeps the transfers made through a local client, by its name.
func (q *TransferListQuery) Client(name string) *TransferListQuery {
	q.query.Where("client=?", name)

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

// LikeEscape is the escape character of the LIKE patterns of the file
// searches.
const LikeEscape = "\033"

// likePattern is the LIKE pattern that matches any text containing the
// substring, its wildcards escaped.
func likePattern(substring string) string {
	replacer := strings.NewReplacer(
		"%", LikeEscape+"%",
		"_", LikeEscape+"_",
	)

	return "%" + replacer.Replace(substring) + "%"
}

// FilePattern adds a condition which will match any transfer whose filename
// contains the provided substring.
func (q *TransferListQuery) FilePattern(substring string) *TransferListQuery {
	pattern := likePattern(substring)

	q.query.Where("src_filename LIKE ? ESCAPE ? OR dest_filename LIKE ? ESCAPE ?",
		pattern, LikeEscape, pattern, LikeEscape)

	return q
}

// Search adds a condition which will match any transfer whose file name or
// path (the local and the remote ones, which the lists show) matches the
// LIKE pattern (escaped with LikeEscape), case aside, or whose follow ID is
// the text (the condition of the followID filter of the REST API).
func (q *TransferListQuery) Search(pattern, text string) *TransferListQuery {
	pattern = strings.ToLower(pattern)

	q.query.Where("(LOWER(src_filename) LIKE ? ESCAPE ? OR LOWER(dest_filename) LIKE ? ESCAPE ? OR "+
		"LOWER(local_path) LIKE ? ESCAPE ? OR LOWER(remote_path) LIKE ? ESCAPE ? OR "+
		"id IN (SELECT COALESCE(transfer_id, history_id) FROM transfer_info WHERE name=? AND value=?))",
		pattern, LikeEscape, pattern, LikeEscape, pattern, LikeEscape, pattern, LikeEscape, model.FollowID, text)

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

// Protocols adds a condition on the protocol of the transfers: one of the
// given ones.
func (q *TransferListQuery) Protocols(protocols ...string) *TransferListQuery {
	q.query.In("protocol", utils.AsAny(protocols)...)

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

// CountByStatus counts the transfers of the query for each status, in one
// reading of the transfers.
func (q *TransferListQuery) CountByStatus() (map[types.TransferStatus]uint64, error) {
	counts, err := q.query.CountBy("status")
	if err != nil {
		return nil, fmt.Errorf("failed to count the transfers by status: %w", err)
	}

	byStatus := make(map[types.TransferStatus]uint64, len(counts))
	for status, n := range counts {
		byStatus[types.TransferStatus(status)] = n
	}

	return byStatus, nil
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
	list.query = db.Select(&list.transfers).OrderBy(orderBy, asc)

	return list
}
