// Package batch applies a gesture of the transfer monitoring to a set of
// transfers at once: the resumption, the retry and the cancellation the model
// does one transfer at a time. The web interface and the REST API enumerate
// the set their filters define, give its identifiers, and get back what was
// done and what was left aside, with its reason, for the notification or the
// answer to say it. The preconditions are those of the unit actions (the
// model: CheckResumable, CheckRetryable, CheckCancellable).
package batch

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// chunkSize is how many transfers a statement names at most (an IN clause),
// whatever the base.
const chunkSize = 500

// Gesture is an action of the transfer monitoring applied to a set.
type Gesture string

const (
	Resume Gesture = "resume"
	Retry  Gesture = "retry"
	Cancel Gesture = "cancel"
)

// Reason says why a transfer of the set was left aside.
type Reason string

const (
	// ReasonServer: a server transfer resumes, or is retried, on the
	// initiative of the partner.
	ReasonServer Reason = "server"
	// ReasonSync: the child of a synchronous transfer resumes through its
	// parent.
	ReasonSync Reason = "sync"
	// ReasonState: the state of the transfer does not allow the gesture (a
	// done transfer neither resumes nor cancels, a running one is not
	// retried).
	ReasonState Reason = "state"
	// ReasonMissing: the transfer no longer exists (another gesture took it
	// meanwhile).
	ReasonMissing Reason = "missing"
	// ReasonGone: the rule, the partner, the account or the client of the
	// transfer no longer exists (a retry).
	ReasonGone Reason = "gone"
	// ReasonElsewhere: the transfer runs on another node of the cluster,
	// which alone can cancel it.
	ReasonElsewhere Reason = "elsewhere"
	// ReasonBusy: the running transfer did not stop in time.
	ReasonBusy Reason = "busy"
)

// Reasons are the reasons in the order they are reported.
func Reasons() []Reason {
	return []Reason{ReasonServer, ReasonSync, ReasonState, ReasonMissing, ReasonGone, ReasonElsewhere, ReasonBusy}
}

// Refusal is a transfer of the set left aside, and why.
type Refusal struct {
	ID     int64
	Reason Reason
}

// Result is what a gesture did on a set: the transfers it was applied to
// (Done), the transfers a retry created (New, one for each of Done, in the
// same order) and the transfers left aside.
type Result struct {
	Done    []int64
	New     []int64
	Refused []Refusal
}

func (r *Result) refuse(id int64, reason Reason) {
	r.Refused = append(r.Refused, Refusal{ID: id, Reason: reason})
}

// missing counts the identifiers of the chunk the base did not give back.
func (r *Result) missing(chunk []int64, found map[int64]bool) {
	for _, id := range chunk {
		if !found[id] {
			r.refuse(id, ReasonMissing)
		}
	}
}

// Counts are the refusals by reason, the reasons without any left out.
func (r *Result) Counts() map[Reason]int {
	counts := map[Reason]int{}
	for _, refusal := range r.Refused {
		counts[refusal.Reason]++
	}

	return counts
}

// Preview is what a gesture would do on a set, counted without doing it:
// the transfers it accepts, those it would leave aside by reason, and, among
// the accepted, the running ones (a cancellation stops them). The transfers
// are read without their information: the child of a synchronous transfer
// counts as accepted here and is refused at the execution.
type Preview struct {
	Accepted int
	Running  int
	Refused  map[Reason]int
}

// Total is the size of the set.
func (p *Preview) Total() int {
	n := p.Accepted
	for _, count := range p.Refused {
		n += count
	}

	return n
}

// ids sorts and deduplicates the identifiers of a set.
func ids(set []int64) []int64 {
	sorted := slices.Clone(set)
	slices.Sort(sorted)

	return slices.Compact(sorted)
}

// check applies the precondition of a gesture to a transfer, as the unit
// action does.
func check(gesture Gesture, view *model.NormalizedTransferView) error {
	switch gesture {
	case Resume:
		return view.CheckResumable() //nolint:wrapcheck // the error of the model, read by reasonOf
	case Retry:
		return view.CheckRetryable() //nolint:wrapcheck // the error of the model, read by reasonOf
	case Cancel:
		return view.CheckCancellable() //nolint:wrapcheck // the error of the model, read by reasonOf
	default:
		return fmt.Errorf("unknown gesture %q", gesture) //nolint:err113 // a programming error
	}
}

// reasonOf is the reason a precondition of the model gives.
func reasonOf(err error) Reason {
	var syncErr *model.ResumeSyncError

	switch {
	case errors.Is(err, model.ErrResumeServer), errors.Is(err, model.ErrRetryServer):
		return ReasonServer
	case errors.As(err, &syncErr):
		return ReasonSync
	default:
		return ReasonState
	}
}

// readViews reads the transfers of a chunk, live or in the history, with
// their information when eager.
func readViews(db database.ReadAccess, chunk []int64, eager bool) (model.NormalizedTransfers, error) {
	var views model.NormalizedTransfers

	query := db.Select(&views).In("id", utils.AsAny(chunk)...)
	if eager {
		query.Eager()
	}

	if err := query.Run(); err != nil {
		return nil, fmt.Errorf("failed to read the transfers of the set: %w", err)
	}

	return views, nil
}

// PreviewOf counts what a gesture would do on a set.
func PreviewOf(db database.ReadAccess, gesture Gesture, set []int64) (*Preview, error) {
	preview := &Preview{Refused: map[Reason]int{}}

	for chunk := range slices.Chunk(ids(set), chunkSize) {
		views, err := readViews(db, chunk, false)
		if err != nil {
			return nil, err
		}

		found := make(map[int64]bool, len(views))

		for _, view := range views {
			found[view.ID] = true

			if checkErr := check(gesture, view); checkErr != nil {
				preview.Refused[reasonOf(checkErr)]++

				continue
			}

			preview.Accepted++

			if view.Status == types.StatusRunning {
				preview.Running++
			}
		}

		for _, id := range chunk {
			if !found[id] {
				preview.Refused[ReasonMissing]++
			}
		}
	}

	return preview, nil
}

// ResumeSet resumes the transfers of the set where they stopped, their next
// attempt at when, in the transaction of db (one of its own when db is the
// base): a transfer the model refuses (a server transfer, the child of a
// synchronous transfer, a state that does not resume, a transfer gone to the
// history) or that no longer exists is counted without stopping the set; an
// error of the base resumes nothing.
func ResumeSet(db database.Access, set []int64, when time.Time) (*Result, error) {
	result := &Result{}

	err := db.Transaction(func(ses *database.Session) error {
		for chunk := range slices.Chunk(ids(set), chunkSize) {
			views, err := readViews(ses, chunk, true)
			if err != nil {
				return err
			}

			found := make(map[int64]bool, len(views))
			resumable := make([]int64, 0, len(views))

			for _, view := range views {
				found[view.ID] = true

				if checkErr := view.CheckResumable(); checkErr != nil {
					result.refuse(view.ID, reasonOf(checkErr))

					continue
				}

				resumable = append(resumable, view.ID)
			}

			result.missing(chunk, found)

			err = model.ResumeTransfers(ses, resumable, when)
			if err != nil {
				return err //nolint:wrapcheck // wrapped once, below
			}

			result.Done = append(result.Done, resumable...)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to resume the set: %w", err)
	}

	return result, nil
}

// RetrySet creates a new transfer for each transfer of the set, now or at
// date, as the unit retry does: the same rule, partner, account, client and
// file; the transfers of the set stay as they are. A server transfer, one
// that has not stopped, one whose rule, partner, account or client no longer
// exists, or one that no longer exists itself, is counted without stopping
// the set. In the transaction of db: an error of the base creates nothing.
func RetrySet(db database.Access, set []int64, date time.Time) (*Result, error) {
	result := &Result{}

	err := db.Transaction(func(ses *database.Session) error {
		for chunk := range slices.Chunk(ids(set), chunkSize) {
			views, err := readViews(ses, chunk, false)
			if err != nil {
				return err
			}

			found := make(map[int64]bool, len(views))

			for _, view := range views {
				found[view.ID] = true

				err = retryOne(ses, result, view, date)
				if err != nil {
					return err
				}
			}

			result.missing(chunk, found)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retry the set: %w", err)
	}

	return result, nil
}

// retryOne makes the new transfer of one transfer of the set, or counts why
// it cannot; an error of the base is returned.
func retryOne(ses *database.Session, result *Result, view *model.NormalizedTransferView, date time.Time) error {
	if err := view.CheckRetryable(); err != nil {
		result.refuse(view.ID, reasonOf(err))

		return nil
	}

	trans, err := view.Restart(ses, date)

	switch {
	case database.IsNotFound(err):
		result.refuse(view.ID, ReasonGone)

		return nil
	case err != nil:
		return fmt.Errorf("failed to prepare the retry of the transfer %d: %w", view.ID, err)
	}

	if err = ses.Insert(trans).Run(); err != nil {
		return fmt.Errorf("failed to insert the retry of the transfer %d: %w", view.ID, err)
	}

	result.Done = append(result.Done, view.ID)
	result.New = append(result.New, trans.ID)

	return nil
}

// CancelSet cancels the transfers of the set: a running one is stopped
// through its pipeline (within ctx), the others go to the history, cancelled,
// by chunks, each chunk in a transaction. The set goes on after a transfer
// left aside (one already done or cancelled, one gone meanwhile, one running
// on another node of the cluster, one that did not stop in time) and stops at
// the first error of the base: what was cancelled stays so, the result says
// how far it went.
func CancelSet(ctx context.Context, db *database.DB, logger *log.Logger, set []int64) (*Result, error) {
	result := &Result{}

	for chunk := range slices.Chunk(ids(set), chunkSize) {
		views, err := readViews(db, chunk, false)
		if err != nil {
			return result, err
		}

		found := make(map[int64]bool, len(views))
		inBase := make([]int64, 0, len(views))

		for _, view := range views {
			found[view.ID] = true

			if stopped, ok := cancelRunning(ctx, db, logger, result, view); ok {
				if stopped {
					result.Done = append(result.Done, view.ID)
				}

				continue
			}

			inBase = append(inBase, view.ID)
		}

		result.missing(chunk, found)

		if err = cancelInBase(db, inBase); err != nil {
			return result, fmt.Errorf("failed to cancel the set: %w", err)
		}

		result.Done = append(result.Done, inBase...)
	}

	return result, nil
}

// cancelRunning handles a transfer of the set that the base alone does not
// cancel: one that no longer cancels (counted), one running here (stopped
// through its pipeline, which cancels it; counted when it does not stop in
// time), one running on another node (counted). It says whether it dealt with
// the transfer, and whether the transfer is cancelled; otherwise the base
// cancels the transfer: a running transfer without its pipeline on a single
// node is cancelled in the base, as the unit action does.
func cancelRunning(ctx context.Context, db *database.DB, logger *log.Logger, result *Result,
	view *model.NormalizedTransferView,
) (stopped, handled bool) {
	if err := view.CheckCancellable(); err != nil {
		result.refuse(view.ID, ReasonState)

		return false, true
	}

	if view.Status != types.StatusRunning {
		return false, false
	}

	pip := pipeline.List.Get(view.ID)

	switch {
	case pip != nil:
		if err := pip.Cancel(ctx); err != nil {
			logger.Warningf("Failed to stop the running transfer %d of the set: %v", view.ID, err)
			result.refuse(view.ID, ReasonBusy)

			return false, true
		}

		return true, true
	case db.Config.NodeID != "":
		result.refuse(view.ID, ReasonElsewhere)

		return false, true
	default:
		return false, false
	}
}

// cancelInBase moves the given live transfers to the history, cancelled, in
// one transaction: their history entries inserted together, the transfers
// deleted together (their information follows each of them).
func cancelInBase(db *database.DB, set []int64) error {
	if len(set) == 0 {
		return nil
	}

	return db.Transaction(func(ses *database.Session) error {
		var transfers model.Transfers
		if err := ses.Select(&transfers).In("id", utils.AsAny(set)...).Eager().Run(); err != nil {
			return fmt.Errorf("failed to read the transfers to cancel: %w", err)
		}

		entries := make([]*model.HistoryEntry, 0, len(transfers))

		for _, trans := range transfers {
			trans.Status = types.StatusCancelled

			entry, err := trans.MakeHistoryEntry(ses, time.Time{})
			if err != nil {
				return fmt.Errorf("failed to move the transfer %d to the history: %w", trans.ID, err)
			}

			entries = append(entries, entry)
		}

		if err := database.InsertBatch(ses, entries...); err != nil {
			return fmt.Errorf("failed to insert the history entries: %w", err)
		}

		if err := ses.DeleteAll(&model.Transfer{}).In("id", utils.AsAny(set)...).Run(); err != nil {
			return fmt.Errorf("failed to delete the cancelled transfers: %w", err)
		}

		return nil
	})
}
