package transfers

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/format"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

// infoKeyPrefix prefixes the catalogue keys of the names of the reserved
// transfer information of the product ("transfers.info_key.__followID__"). A
// key without an entry keeps its technical name.
const infoKeyPrefix = "transfers.info_key."

type kv = protoform.KV

// infoRow is a transfer information: its name (the functional one of a
// reserved key, with the technical one under it) and its value.
type infoRow struct {
	Name  string
	Tech  string
	Value string
}

// errorView is the error of a transfer: the readable cause, the technical
// code, the message as the product keeps it, the step it stopped at.
type errorView struct {
	Cause   string
	Code    string
	Message string
	Step    string
}

// button is an action of the header of the sheet, a POST form.
type button struct {
	Label   string
	Icon    string
	Action  string
	Primary bool
}

type detailData struct {
	ID       int64
	File     string
	Status   gui.Status
	Number   string
	BackURL  string
	Buttons  []button
	Actions  []gui.Action // the menu: retry, cancel
	Error    *errorView
	Transfer []kv
	Files    []kv
	Run      []kv
	Progress progressView
	Retries  []kv // nil for a server transfer
	Infos    []infoRow
}

// Detail serves GET /webui/transfers/{id}: the sheet of a transfer.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	trans, ok := h.transfer(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	data := detailOf(translate, h.shell.Catalog().Has, trans, canWrite(r), carried, h.sheetLinks(trans))
	data.Files = append(data.Files, h.expectedRows(translate, trans)...)

	screen := h.screen(data.File, crumbs(translate, data.Number, carried), 0)
	screen.Notices = Notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, detailView(data))
}

// detailOf lays out the sheet of a transfer.
func detailOf(translate gui.Translator, has func(string) bool, trans *model.NormalizedTransferView,
	write bool, carried string, links sheetLinks,
) detailData {
	data := detailData{
		ID:       trans.ID,
		File:     fileName(trans),
		Status:   StatusOf(translate, trans.Status),
		Number:   translate("transfers.detail.number", "id", trans.ID),
		BackURL:  listURL(carried),
		Progress: progressOf(translate, trans),
		Transfer: transferRows(translate, trans, links),
		Files:    fileRows(translate, trans),
		Run:      runRows(translate, trans),
		Infos:    infoRows(translate, has, trans.TransferInfo),
	}

	if data.File == "" {
		data.File = data.Number
	}

	if label := ErrorLabel(translate, trans.ErrCode); label != "" {
		data.Error = &errorView{
			Cause: label, Code: trans.ErrCode.String(), Message: trans.ErrDetails, Step: stepLabel(translate, trans.Step),
		}
	}

	if !trans.IsServer && trans.IsTransfer {
		data.Retries = retryRows(translate, trans)
	}

	if write {
		data.Buttons, data.Actions = sheetActions(translate, trans, carried)
	}

	return data
}

// expectedRows are the paths a transfer not started yet will have, as the
// pipeline computes them at its start (pipeline.ExpectedPaths); none once it
// has started, or when they cannot be computed (logged).
func (h *Handlers) expectedRows(translate gui.Translator, view *model.NormalizedTransferView) []kv {
	if !view.IsTransfer || view.LocalPath != "" {
		return nil
	}

	var trans model.Transfer
	if err := h.db.Get(&trans, "id=?", view.ID).Run(); err != nil {
		h.logger.Warningf("failed to read the transfer %d for its expected paths: %v", view.ID, err)

		return nil
	}

	transCtx, err := model.GetTransferContext(h.db, h.logger, &trans)
	if err != nil {
		h.logger.Warningf("failed to read the context of the transfer %d: %v", view.ID, err)

		return nil
	}

	local, remote, err := pipeline.ExpectedPaths(transCtx)
	if err != nil {
		h.logger.Warningf("failed to compute the expected paths of the transfer %d: %v", view.ID, err)

		return nil
	}

	rows := []kv{{Label: translate("transfers.detail.expected_local"), Value: local, Mono: true}}
	if remote != "" {
		rows = append(rows, kv{Label: translate("transfers.detail.expected_remote"), Value: remote, Mono: true})
	}

	return rows
}

// sheetLinks are the sheets a transfer leads to: its rule, its partner or
// server, its account, its client (#783). Empty when the object no longer
// exists: the name stays, without a link.
type sheetLinks struct{ Rule, Agent, Account, Client string }

// sheetLinks finds the objects of the transfer by their names, as the REST
// API does: the view of a transfer carries names, a history entry outlives
// the objects it names.
func (h *Handlers) sheetLinks(trans *model.NormalizedTransferView) sheetLinks {
	var links sheetLinks

	var rule model.Rule
	if h.found(h.db.Get(&rule, "name=? AND is_send=?", trans.Rule, trans.IsSend).Run()) {
		links.Rule = common.WebuiURL("/rules/" + strconv.FormatInt(rule.ID, 10))
	}

	if trans.IsServer {
		var server model.LocalAgent
		if h.found(h.db.Get(&server, "name=?", trans.Agent).Run()) {
			links.Agent = common.WebuiURL("/servers/" + strconv.FormatInt(server.ID, 10))

			var account model.LocalAccount
			if h.found(h.db.Get(&account, "local_agent_id=? AND login=?", server.ID, trans.Account).Run()) {
				links.Account = links.Agent + "/accounts/" + strconv.FormatInt(account.ID, 10)
			}
		}
	} else {
		var partner model.RemoteAgent
		if h.found(h.db.Get(&partner, "name=?", trans.Agent).Run()) {
			links.Agent = common.WebuiURL("/partners/" + strconv.FormatInt(partner.ID, 10))

			var account model.RemoteAccount
			if h.found(h.db.Get(&account, "remote_agent_id=? AND login=?", partner.ID, trans.Account).Run()) {
				links.Account = links.Agent + "/accounts/" + strconv.FormatInt(account.ID, 10)
			}
		}
	}

	if trans.Client != "" {
		var client model.Client
		if h.found(h.db.Get(&client, "name=?", trans.Client).Run()) {
			links.Client = common.WebuiURL("/clients/" + strconv.FormatInt(client.ID, 10))
		}
	}

	return links
}

// found says whether a lookup by name succeeded; an absent object is the
// ordinary case of a history entry, any other error is logged.
func (h *Handlers) found(err error) bool {
	if err != nil && !database.IsNotFound(err) {
		h.logger.Warningf("failed to find an object of the transfer: %v", err)
	}

	return err == nil
}

// transferRows are the transfer: the rule and its direction, the partner or
// the server and the account, the client, the protocol, the identifiers;
// each object a link to its sheet while it exists.
func transferRows(translate gui.Translator, trans *model.NormalizedTransferView, links sheetLinks) []kv {
	agent := "transfers.detail.partner"
	if trans.IsServer {
		agent = "transfers.detail.server"
	}

	rows := []kv{
		{Label: translate("transfers.detail.rule"), Value: trans.Rule, Mono: true, URL: links.Rule},
		{Label: translate("transfers.detail.direction"), Value: direction(translate, trans)},
		{Label: translate(agent), Value: trans.Agent, URL: links.Agent},
		{Label: translate("transfers.detail.account"), Value: trans.Account, Mono: true, URL: links.Account},
	}

	if trans.Client != "" {
		rows = append(rows,
			kv{Label: translate("transfers.detail.client"), Value: trans.Client, Mono: true, URL: links.Client})
	}

	rows = append(rows, kv{Label: translate("transfers.detail.protocol"), Value: protocolLabel(trans.Protocol)})

	if trans.RemoteTransferID != "" {
		rows = append(rows, kv{Label: translate("transfers.detail.remote_id"), Value: trans.RemoteTransferID, Mono: true})
	}

	if followID, ok := trans.TransferInfo[model.FollowID]; ok {
		rows = append(rows, kv{Label: translate("transfers.detail.follow_id"), Value: fmt.Sprint(followID), Mono: true})
	}

	return rows
}

// fileRows are the file: its names at the source and at the destination,
// its paths on the gateway and at the partner, its size.
func fileRows(translate gui.Translator, trans *model.NormalizedTransferView) []kv {
	size := translate("transfers.detail.size_unknown")
	if trans.Filesize >= 0 {
		size = translate("transfers.detail.size_bytes", "bytes", format.GroupDigits(translate, uint64(trans.Filesize)),
			"size", format.Bytes(translate, trans.Filesize))
	}

	rows := []kv{}

	for _, item := range []struct{ key, value string }{
		{"transfers.detail.src", trans.SrcFilename},
		{"transfers.detail.dest", trans.DestFilename},
		{"transfers.detail.local_path", trans.LocalPath},
		{"transfers.detail.remote_path", trans.RemotePath},
	} {
		if item.value != "" {
			rows = append(rows, kv{Label: translate(item.key), Value: item.value, Mono: true})
		}
	}

	return append(rows, kv{Label: translate("transfers.detail.size"), Value: size})
}

// runRows are the run of the transfer: its start, its end, the step and the
// task it is at.
func runRows(translate gui.Translator, trans *model.NormalizedTransferView) []kv {
	startKey := "transfers.detail.start"

	switch trans.Status {
	case types.StatusPlanned:
		startKey = "transfers.detail.start_planned"
	case types.StatusAvailable:
		startKey = "transfers.detail.due"
	default:
	}

	rows := []kv{{Label: translate(startKey), Value: dateTime(translate, trans.Start)}}

	if !trans.Stop.IsZero() {
		rows = append(rows, kv{Label: translate("transfers.detail.stop"), Value: dateTime(translate, trans.Stop)})
	}

	if step := stepLabel(translate, trans.Step); step != "" {
		rows = append(rows, kv{Label: translate("transfers.detail.step"), Value: step})
	}

	if trans.TaskNumber > 0 && (trans.Step == types.StepPreTasks || trans.Step == types.StepPostTasks ||
		trans.Step == types.StepErrorTasks) {
		rows = append(rows, kv{Label: translate("transfers.detail.task"), Value: strconv.Itoa(int(trans.TaskNumber))})
	}

	return rows
}

// retryRows are the automatic retries of a client transfer.
func retryRows(translate gui.Translator, trans *model.NormalizedTransferView) []kv {
	if trans.RemainingTries <= 0 {
		return []kv{{Label: translate("transfers.retries.remaining"), Value: translate("transfers.retries.none")}}
	}

	rows := []kv{{Label: translate("transfers.retries.remaining"), Value: strconv.Itoa(int(trans.RemainingTries))}}

	if trans.Status == types.StatusError && !trans.NextRetry.IsZero() {
		rows = append(rows, kv{Label: translate("transfers.retries.next"), Value: dateTime(translate, trans.NextRetry)})
	}

	rows = append(rows, kv{
		Label: translate("transfers.retries.delay"),
		Value: translate("transfers.retries.seconds", "n",
			format.GroupDigits(translate, uint64(max(trans.NextRetryDelay, 0)))),
	})

	if trans.RetryIncrementFactor > 0 {
		factor := strings.Replace(strconv.FormatFloat(float64(trans.RetryIncrementFactor), 'f', -1, 32), ".",
			translate("transfers.decimal"), 1)
		rows = append(rows, kv{Label: translate("transfers.retries.factor"), Value: factor})
	}

	return rows
}

// stepLabel is the label of a step, "" for none.
func stepLabel(translate gui.Translator, step types.TransferStep) string {
	if step == types.StepNone || !step.IsValid() {
		return ""
	}

	return translate(stepKeyPrefix + step.String())
}

// infoRows are the transfer information, in the order of their names: the
// reserved keys of the product by their functional name.
func infoRows(translate gui.Translator, has func(string) bool, infos map[string]any) []infoRow {
	rows := make([]infoRow, 0, len(infos))

	for name, value := range infos {
		line := infoRow{Name: name, Value: fmt.Sprint(value)}
		if key := infoKeyPrefix + name; has(key) {
			line.Name, line.Tech = translate(key), name
		}

		rows = append(rows, line)
	}

	slices.SortFunc(rows, func(a, b infoRow) int { return strings.Compare(a.Name, b.Name) })

	return rows
}
