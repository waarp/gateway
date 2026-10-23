package transfers

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

const (
	// The parameter that says an action was asked from the sheet of the
	// transfer, where the user comes back; the list otherwise.
	fromParam = "from"
	fromSheet = "sheet"

	// How long the pause and the cancellation of a running transfer may
	// take, as the REST API allows.
	actionTimeout = time.Second

	// The field of the retry page: when the new transfer starts, now when
	// empty.
	fieldDate = "date"
)

// allowed are the actions the state of a transfer allows (screen sheet: the
// table of the actions by state), checked again by each action.
type allowed struct{ Pause, Resume, Cancel, Retry bool }

func allowedOf(trans *model.NormalizedTransferView) allowed {
	return allowed{
		// A planned transfer is paused by the gateway that would start it: a
		// client transfer.
		Pause: trans.Status == types.StatusRunning || (trans.Status == types.StatusPlanned && !trans.IsServer),
		// A paused, interrupted or failed client transfer which is not the
		// child of a synchronous transfer (the model says so).
		Resume: trans.IsTransfer && trans.CheckResumable() == nil,
		Cancel: !trans.Status.IsOneOf(types.StatusDone, types.StatusCancelled),
		// A new transfer from a client transfer that no longer runs nor waits
		// to: the gateway starts client transfers only.
		Retry: !trans.IsServer && trans.Status.IsOneOf(types.StatusPaused, types.StatusInterrupted, types.StatusError,
			types.StatusDone, types.StatusCancelled),
	}
}

// elsewhere tells whether a running transfer runs on another node of the
// cluster: this node cannot pause nor cancel it (the REST API answers with an
// internal error in that case).
func (h *Handlers) elsewhere(trans *model.NormalizedTransferView) bool {
	return trans.Status == types.StatusRunning && pipeline.List.Get(trans.ID) == nil && h.db.Config.NodeID != ""
}

// actionURL is the address of an action on a transfer, carrying the list
// state and, from the sheet, where to come back.
func actionURL(id int64, action, carried string, onSheet bool) string {
	u := withRet(transferURL(id, "/"+action), carried)
	if onSheet {
		u = liststate.WithQuery(u, url.Values{fromParam: {fromSheet}})
	}

	return u
}

// rowActions is the "⋯" menu of a transfer of the list: its sheet, then the
// actions its state allows to whoever may write; the cancellation last.
func rowActions(translate gui.Translator, trans *model.NormalizedTransferView, write bool, carried string,
) []gui.Action {
	acts := []gui.Action{{
		Icon: "eye", Label: translate("transfers.action.detail"), URL: withRet(transferURL(trans.ID, ""), carried),
	}}
	if !write {
		return acts
	}

	can := allowedOf(trans)

	if can.Pause {
		acts = append(acts, gui.Action{
			Icon: "stop", Label: translate("transfers.action.pause"),
			Action: actionURL(trans.ID, "pause", carried, false),
		})
	}

	if can.Resume {
		acts = append(acts, gui.Action{
			Icon: "play", Label: translate("transfers.action.resume"),
			Action: actionURL(trans.ID, "resume", carried, false),
		})
	}

	if can.Retry {
		acts = append(acts, gui.Action{
			Icon: "rotate", Label: translate("transfers.action.retry"),
			URL: actionURL(trans.ID, "retry", carried, false),
		})
	}

	if can.Cancel {
		acts = append(acts, gui.Action{
			Icon: "xmark", Label: translate("transfers.action.cancel"),
			URL: actionURL(trans.ID, "cancel", carried, false), Danger: true,
		})
	}

	return acts
}

// sheetActions are the actions of the sheet: resuming (the main action of a
// transfer that stopped) or pausing as buttons, retrying and cancelling in
// the menu.
func sheetActions(translate gui.Translator, trans *model.NormalizedTransferView, carried string,
) ([]button, []gui.Action) {
	can := allowedOf(trans)

	var (
		buttons []button
		acts    []gui.Action
	)

	if can.Resume {
		buttons = append(buttons, button{
			Label: translate("transfers.action.resume"), Icon: "play",
			Action: actionURL(trans.ID, "resume", carried, true), Primary: true,
		})
	}

	if can.Pause {
		buttons = append(buttons, button{
			Label: translate("transfers.action.pause"), Icon: "stop",
			Action: actionURL(trans.ID, "pause", carried, true),
		})
	}

	if can.Retry {
		acts = append(acts, gui.Action{
			Icon: "rotate", Label: translate("transfers.action.retry"),
			URL: actionURL(trans.ID, "retry", carried, true),
		})
	}

	if can.Cancel {
		acts = append(acts, gui.Action{
			Icon: "xmark", Label: translate("transfers.action.cancel"),
			URL: actionURL(trans.ID, "cancel", carried, true), Danger: true,
		})
	}

	return buttons, acts
}

// done sends back where the action was asked, with its result: the sheet of
// the transfer, or the list as the user left it.
func done(w http.ResponseWriter, r *http.Request, trans *model.NormalizedTransferView, result string) {
	values := url.Values{liststate.DoneParam: {result}, liststate.NameParam: {fileName(trans)}}

	if r.URL.Query().Get(fromParam) == fromSheet {
		liststate.RedirectTo(w, r, withRet(transferURL(trans.ID, ""), ret(r)), values)

		return
	}

	listSpec.RedirectList(w, r, ret(r), values)
}

// backURL is where an action page comes back to: the sheet or the list.
func backURL(r *http.Request, trans *model.NormalizedTransferView) string {
	if r.URL.Query().Get(fromParam) == fromSheet {
		return withRet(transferURL(trans.ID, ""), ret(r))
	}

	return listURL(ret(r))
}

// refusedData is the page of an action the state of the transfer does not
// allow: what was asked, why it cannot be done, the way back.
type refusedData struct {
	Title   string
	Reason  string
	File    string
	Status  gui.Status
	Number  string
	BackURL string
}

// refuse renders the page of an action refused by the state of the transfer
// (409), never a raw error.
func (h *Handlers) refuse(w http.ResponseWriter, r *http.Request, trans *model.NormalizedTransferView,
	titleKey, reason string,
) {
	translate := h.shell.Translator(r)
	title := translate(titleKey, "name", fileName(trans))
	number := translate("transfers.detail.number", "id", trans.ID)

	h.shell.Render(w, h.logger, r, h.screen(title, crumbs(translate, number, ret(r)), http.StatusConflict),
		refusedView(refusedData{
			Title: title, Reason: reason, File: fileName(trans), Status: statusOf(translate, trans.Status),
			Number: number, BackURL: backURL(r, trans),
		}))
}

// stateReason says why the state of a transfer forbids an action: the
// sentence of the action, completed with the state the transfer is in.
func stateReason(translate gui.Translator, key string, trans *model.NormalizedTransferView) string {
	return translate(key, "state", translate(stateKeyPrefix+specOfStatus(trans.Status).key))
}

// Pause serves POST /webui/transfers/{id}/pause.
func (h *Handlers) Pause(w http.ResponseWriter, r *http.Request) {
	trans, ok := h.transfer(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)

	switch {
	case !allowedOf(trans).Pause:
		h.refuse(w, r, trans, "transfers.refused.pause_title", stateReason(translate, "transfers.refused.pause", trans))

		return
	case h.elsewhere(trans):
		h.refuse(w, r, trans, "transfers.refused.pause_title", translate("transfers.refused.elsewhere"))

		return
	}

	var err error

	if trans.Status == types.StatusPlanned {
		err = h.pausePlanned(trans.ID)
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), actionTimeout)
		defer cancel()

		err = internal.PauseTransfer(ctx, h.db, trans)
	}

	if err != nil {
		h.logger.Errorf("failed to pause the transfer %d: %v", trans.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	done(w, r, trans, "paused")
}

// pausePlanned pauses a planned transfer before it starts, as the REST API
// does.
func (h *Handlers) pausePlanned(id int64) error {
	var trans model.Transfer
	if err := h.db.Get(&trans, "id=?", id).Run(); err != nil {
		return err //nolint:wrapcheck // logged by the caller with the transfer
	}

	trans.Status = types.StatusPaused

	return h.db.Update(&trans).Cols("status").Run() //nolint:wrapcheck // logged by the caller with the transfer
}

// Resume serves POST /webui/transfers/{id}/resume.
func (h *Handlers) Resume(w http.ResponseWriter, r *http.Request) {
	trans, ok := h.transfer(w, r)
	if !ok {
		return
	}

	if reason := h.resumeRefusal(h.shell.Translator(r), trans); reason != "" {
		h.refuse(w, r, trans, "transfers.refused.resume_title", reason)

		return
	}

	if err := internal.ResumeTransfer(h.db, trans); err != nil {
		h.logger.Errorf("failed to resume the transfer %d: %v", trans.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	done(w, r, trans, "resumed")
}

// resumeRefusal says why a transfer cannot resume, "" when it can: the
// reason the model gives, in words.
func (h *Handlers) resumeRefusal(translate gui.Translator, trans *model.NormalizedTransferView) string {
	if !trans.IsTransfer {
		return stateReason(translate, "transfers.refused.resume", trans)
	}

	err := trans.CheckResumable()

	var syncErr *model.ResumeSyncError

	switch {
	case err == nil:
		return ""
	case errors.Is(err, model.ErrResumeServer):
		return translate("transfers.refused.resume_server")
	case errors.As(err, &syncErr):
		return translate("transfers.refused.resume_sync", "parent", syncErr.ParentID)
	default:
		return stateReason(translate, "transfers.refused.resume", trans)
	}
}

// cancelData is the confirmation page of the cancellation.
type cancelData struct {
	Title     string
	Action    string
	CancelURL string
	Running   bool
	File      string
	Number    string
	Status    gui.Status
	Rule      string
	Direction string
	Partner   string
}

// CancelForm serves GET /webui/transfers/{id}/cancel: the confirmation page
// (decision D10: a page, not a dialog), which names the transfer, its file
// and its state.
func (h *Handlers) CancelForm(w http.ResponseWriter, r *http.Request) {
	trans, ok := h.transfer(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	if !allowedOf(trans).Cancel {
		h.refuse(w, r, trans, "transfers.refused.cancel_title", stateReason(translate, "transfers.refused.cancel", trans))

		return
	}

	title := translate("transfers.cancel.title", "name", fileName(trans))
	number := translate("transfers.detail.number", "id", trans.ID)

	h.shell.Render(w, h.logger, r, h.screen(title, crumbs(translate, number, ret(r)), 0),
		cancelView(cancelData{
			Title:     title,
			Action:    r.URL.RequestURI(),
			CancelURL: backURL(r, trans),
			Running:   trans.Status == types.StatusRunning,
			File:      fileName(trans),
			Number:    number,
			Status:    statusOf(translate, trans.Status),
			Rule:      trans.Rule,
			Direction: direction(translate, trans),
			Partner:   trans.Agent,
		}))
}

// Cancel serves POST /webui/transfers/{id}/cancel: the transfer goes to the
// history, cancelled; a running one is stopped first.
func (h *Handlers) Cancel(w http.ResponseWriter, r *http.Request) {
	trans, ok := h.transfer(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)

	switch {
	case !allowedOf(trans).Cancel:
		h.refuse(w, r, trans, "transfers.refused.cancel_title", stateReason(translate, "transfers.refused.cancel", trans))

		return
	case h.elsewhere(trans):
		h.refuse(w, r, trans, "transfers.refused.cancel_title", translate("transfers.refused.elsewhere"))

		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), actionTimeout)
	defer cancel()

	if err := internal.CancelTransfer(ctx, h.db, trans); err != nil {
		h.logger.Errorf("failed to cancel the transfer %d: %v", trans.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	done(w, r, trans, "cancelled")
}

// retryData is the retry page: now, or at a date.
type retryData struct {
	Title     string
	Action    string
	CancelURL string
	File      string
	Number    string
	Date      field
	Refused   string // the retry the product refuses (a rule, a partner... gone)
}

// RetryForm serves GET /webui/transfers/{id}/retry.
func (h *Handlers) RetryForm(w http.ResponseWriter, r *http.Request) {
	trans, ok := h.transfer(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	if !allowedOf(trans).Retry {
		h.refuse(w, r, trans, "transfers.refused.retry_title", h.retryRefusal(translate, trans))

		return
	}

	h.renderRetry(w, r, trans, &retryData{}, 0)
}

// retryRefusal says why a transfer cannot be retried.
func (h *Handlers) retryRefusal(translate gui.Translator, trans *model.NormalizedTransferView) string {
	if trans.IsServer {
		return translate("transfers.refused.retry_server")
	}

	return stateReason(translate, "transfers.refused.retry", trans)
}

func (h *Handlers) renderRetry(w http.ResponseWriter, r *http.Request, trans *model.NormalizedTransferView,
	data *retryData, status int,
) {
	translate := h.shell.Translator(r)
	data.Title = translate("transfers.retry.title", "name", fileName(trans))
	data.Action = r.URL.RequestURI()
	data.CancelURL = backURL(r, trans)
	data.File = fileName(trans)
	data.Number = translate("transfers.detail.number", "id", trans.ID)

	h.shell.Render(w, h.logger, r, h.screen(data.Title, crumbs(translate, data.Number, ret(r)), status),
		retryView(data))
}

// Retry serves POST /webui/transfers/{id}/retry: a new transfer, with the
// same rule, partner, account, client and file, now or at the given date.
// The sheet of the new transfer follows.
func (h *Handlers) Retry(w http.ResponseWriter, r *http.Request) {
	trans, ok := h.transfer(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	if !allowedOf(trans).Retry {
		h.refuse(w, r, trans, "transfers.refused.retry_title", h.retryRefusal(translate, trans))

		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	data := &retryData{Date: field{Value: strings.TrimSpace(r.PostForm.Get(fieldDate))}}
	date := time.Now()

	if data.Date.Value != "" {
		parsed, valid := parseLocal(data.Date.Value)
		if !valid {
			data.Date.Error = translate("transfers.form.error.date")
			h.renderRetry(w, r, trans, data, http.StatusUnprocessableEntity)

			return
		}

		date = parsed
	}

	newTrans, err := internal.ReprogramTransfer(h.db, trans, date)

	switch {
	case err == nil:
		liststate.RedirectTo(w, r, withRet(transferURL(newTrans.ID, ""), ret(r)), url.Values{
			liststate.DoneParam: {"retried"}, liststate.NameParam: {strconv.FormatInt(newTrans.ID, 10)},
		})
	case database.IsNotFound(err):
		data.Refused = translate("transfers.retry.gone")
		h.renderRetry(w, r, trans, data, http.StatusUnprocessableEntity)
	default:
		h.logger.Errorf("failed to retry the transfer %d: %v", trans.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}
