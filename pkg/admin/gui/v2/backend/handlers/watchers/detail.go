package watchers

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// filesShown is how many of the last files received the sheet shows; the
// transfers screen, filtered on the watcher, shows them all.
const filesShown = 20

// kv is a row of the identity list of the sheet.
type kv struct {
	Label string
	Value string
	URL   string // the value links there when set
	Code  bool   // the value is technical (the pattern)
}

// fileRow is a file the watcher received: the transfer it created, with
// its date and its state, linking to the sheet of the transfer.
type fileRow struct {
	Name   string
	Date   string
	Time   string
	Status gui.Status
	URL    string
}

type detailData struct {
	Name       string
	BackURL    string
	EditURL    string
	Actions    []gui.Action
	Identity   []kv
	State      gui.Status
	Reason     string
	Files      []fileRow
	FilesText  string
	FilesCount string // the summary of the folded card
	// TransfersURL leads to the transfers screen, filtered on the rule, the
	// partner and the account of the watcher.
	TransfersURL string
}

// Detail serves GET /webui/watchers/{id}: the sheet of a watcher: what it
// is, the state of its service, the last files it received.
func (h *Handlers) Detail(w http.ResponseWriter, r *http.Request) {
	item, ok := h.watcher(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)
	fw := item.fw
	title := translate("watchers.detail.title", "name", fw.Flow)

	files, err := h.filesOf(translate, fw, item.partnerName())
	if err != nil {
		h.logger.Errorf("failed to list the files of the partner watcher %q: %v", fw.Flow, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := detailData{
		Name:         fw.Flow,
		BackURL:      listURL(carried),
		Actions:      actions(translate, &item, user, true, carried),
		Identity:     identityOf(translate, &item),
		State:        status(translate, item.state),
		Reason:       item.reason,
		Files:        files,
		FilesText:    translate("watchers.detail.files_text", "n", filesShown),
		FilesCount:   filesCount(translate, len(files)),
		TransfersURL: transfersURL(fw, item.partnerName()),
	}

	if user.Write {
		data.EditURL = withRet(watcherURL(fw.ID, "/edit"), carried)
	}

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), detailView(data))
}

// filesCount says how many files the card shows, in its summary.
func filesCount(translate gui.Translator, n int) string {
	switch n {
	case 0:
		return translate("watchers.detail.files_none_short")
	case 1:
		return translate("watchers.detail.files_count_one")
	default:
		return translate("watchers.detail.files_count", "n", n)
	}
}

// identityOf is the identity list of the sheet: the partner and the
// account, the client, the rule (linked to its sheet), the pattern, the
// interval, the files already transferred, the start.
func identityOf(translate gui.Translator, item *entry) []kv {
	fw := item.fw
	yes, no := translate("watchers.detail.yes"), translate("watchers.detail.no")

	skip, enabled := yes, yes
	if fw.NoDuplicateCheck {
		skip = no
	}

	if fw.Disabled {
		enabled = no
	}

	return []kv{
		{Label: translate("watchers.detail.partner"), Value: item.partnerName() + " · " + fw.RemoteAccount.Login},
		{Label: translate("watchers.detail.client"), Value: fw.Client.Name + " · " + protoLabel(fw.Client.Protocol)},
		{
			Label: translate("watchers.detail.rule"), Value: fw.Rule.Name,
			URL: common.WebuiURL("/rules/" + strconv.FormatInt(fw.RuleID, 10)),
		},
		{Label: translate("watchers.detail.pattern"), Value: fw.Pattern, Code: true},
		{Label: translate("watchers.detail.interval"), Value: intervalText(translate, fw.Interval)},
		{Label: translate("watchers.detail.skip"), Value: skip},
		{Label: translate("watchers.detail.enabled"), Value: enabled},
	}
}

// filesOf is the last files the watcher received: the receive transfers of
// its rule, its partner and its account, the latest first, as the transfers
// screen shows them.
func (h *Handlers) filesOf(translate gui.Translator, fw *model.FileWatcher, partner string) ([]fileRow, error) {
	list, err := internal.StartTransferQuery(h.db, "start", false).
		RuleName(fw.Rule.Name).Direction(false).Requested(partner).Requester(fw.RemoteAccount.Login).
		Limit(filesShown, 0).Run()
	if err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	rows := make([]fileRow, 0, len(list))

	for _, trans := range list {
		local := trans.Start.Local()
		rows = append(rows, fileRow{
			Name:   trans.SrcFilename,
			Date:   local.Format(translate("transfers.date")),
			Time:   local.Format(time.TimeOnly),
			Status: transfers.StatusOf(translate, trans.Status),
			URL:    common.WebuiURL("/transfers/" + strconv.FormatInt(trans.ID, 10)),
		})
	}

	return rows, nil
}

// transfersURL is the transfers screen filtered on the watcher: the receive
// transfers of its rule, its partner and its account.
func transfersURL(fw *model.FileWatcher, partner string) string {
	v := url.Values{
		"rule": {fw.Rule.Name}, "direction": {"receive"}, "partner": {partner}, "account": {fw.RemoteAccount.Login},
	}

	return common.WebuiURL("/transfers") + "?" + v.Encode()
}
