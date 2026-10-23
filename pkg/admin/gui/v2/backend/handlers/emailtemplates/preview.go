package emailtemplates

import (
	"cmp"
	"html"
	"io"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"
	xhtml "golang.org/x/net/html"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/ruleuse"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

const (
	// The parameters of the preview: the values of the markers (exampleValues
	// or the id of a transfer) and the highlighting of the values.
	valuesParam   = "values"
	markParam     = "mark"
	exampleValues = "example"

	// How many recent transfers of the rules that use a template the preview
	// offers.
	recentTransfers = 5

	// The example transfer.
	exampleID       = 1234
	exampleRemoteID = "1729871530123"
	exampleSize     = 48213
	exampleAge      = 2 * time.Minute

	// Around a marker before the substitution, around its value after: the
	// highlighting of the values (private use characters, never in a
	// template).
	markOpen  = ""
	markClose = ""
)

// markerPattern finds the markers of a template to highlight their values: a
// name with an optional parameter, or the key of a transfer info.
var markerPattern = regexp.MustCompile(`#(?:[A-Za-z][A-Za-z0-9_]*(?:\([^)#\n]*\))?|TI_[^#\s]+)#`)

type valueOption struct {
	Value    string
	Label    string
	Selected bool
}

// segment is a piece of a text of the preview: a value of a marker
// (highlighted) or the text around it.
type segment struct {
	Text string
	Mark bool
}

// previewData is the preview of a template: the message as the EMAIL task
// sends it, the markers substituted with the values of a transfer or with
// example values.
type previewData struct {
	// Action: on the sheet, the page itself (a GET form chooses the values);
	// in the form, empty (the script of the screen updates the preview).
	Action  string
	Ret     string
	ID      int64 // the template, in the form of an existing one
	Options []valueOption
	Note    string
	Mark    bool
	HTML    bool   // an HTML body, rendered as HTML
	Failed  string // the error the task would fail with
	Subject []segment
	Files   [][]segment
	Body    string // the document of the body, for the isolated frame
}

// preview substitutes the markers of a template with the values of one of
// the recent transfers of the rules that use it (values is its id), or with
// example values, by the substitution of the EMAIL task.
func (h *Handlers) preview(translate gui.Translator, tpl *model.EmailTemplate, uses []ruleuse.Use, values string,
	mark bool,
) *previewData {
	data := &previewData{ID: tpl.ID, Mark: mark, HTML: isHTML(tpl.MIMEType)}
	recent := h.recent(uses)

	data.Options = append(data.Options, valueOption{
		Value: exampleValues, Label: translate("email_templates.preview.example"), Selected: true,
	})

	var chosen *model.NormalizedTransferView

	for _, v := range recent {
		opt := valueOption{Value: strconv.FormatInt(v.ID, 10), Label: transferLabel(translate, v)}
		if opt.Value == values {
			chosen, opt.Selected, data.Options[0].Selected = v, true, false
		}

		data.Options = append(data.Options, opt)
	}

	var transCtx *model.TransferContext

	switch {
	case chosen != nil:
		transCtx = h.transferContext(chosen)
		data.Note = translate("email_templates.preview.note_transfer")
	case len(recent) == 0:
		transCtx = h.exampleContext(translate, uses)
		data.Note = translate("email_templates.preview.note_none")
	default:
		transCtx = h.exampleContext(translate, uses)
		data.Note = translate("email_templates.preview.note_example")
	}

	rendered := *tpl
	rendered.Attachments = slices.Clone(tpl.Attachments)

	if mark {
		rendered.Subject = wrapMarkers(rendered.Subject)
		rendered.Body = wrapMarkers(rendered.Body)

		for i, a := range rendered.Attachments {
			rendered.Attachments[i] = wrapMarkers(a)
		}
	}

	msg, err := tasks.RenderEmail(&rendered, transCtx)
	if err != nil {
		data.Failed = strings.NewReplacer(markOpen, "", markClose, "").Replace(err.Error())

		return data
	}

	data.Subject = segments(msg.Subject)
	for _, a := range msg.Attachments {
		data.Files = append(data.Files, segments(a))
	}

	data.Body = bodyDocument(msg.Body, data.HTML)

	return data
}

// recent reads the latest transfers, running or finished, of the rules that
// use a template, the most recent first.
func (h *Handlers) recent(uses []ruleuse.Use) []*model.NormalizedTransferView {
	var all []*model.NormalizedTransferView

	for _, u := range uses {
		var list model.NormalizedTransfers
		if err := h.db.Select(&list).Where("rule=? AND is_send=?", u.Rule, u.Send).
			OrderBy("start", false).Limit(recentTransfers, 0).Run(); err != nil {
			h.logger.Errorf("failed to read the transfers of the rule %q: %v", u.Rule, err)

			return nil
		}

		all = append(all, list...)
	}

	slices.SortStableFunc(all, func(a, b *model.NormalizedTransferView) int {
		return cmp.Or(b.Start.Compare(a.Start), cmp.Compare(b.ID, a.ID))
	})

	return all[:min(len(all), recentTransfers)]
}

// transferLabel names a transfer in the list of the values: its id, its rule,
// its start, and its error if any.
func transferLabel(translate gui.Translator, v *model.NormalizedTransferView) string {
	key := "email_templates.preview.transfer"
	if v.Status == types.StatusError {
		key = "email_templates.preview.transfer_error"
	}

	return translate(key, "id", v.ID, "rule", v.Rule,
		"date", v.Start.Local().Format(translate("email_templates.preview.date_layout")))
}

// transferContext rebuilds what the tasks read of a transfer: its values as
// the transfer or the history kept them, its rule as it is now (for its
// directories), and the names of its partners. A rule or a server deleted
// since is kept by name.
func (h *Handlers) transferContext(view *model.NormalizedTransferView) *model.TransferContext {
	transCtx := &model.TransferContext{
		Transfer: &model.Transfer{
			ID: view.ID, RemoteTransferID: view.RemoteTransferID,
			SrcFilename: view.SrcFilename, DestFilename: view.DestFilename,
			LocalPath: view.LocalPath, RemotePath: view.RemotePath, Filesize: view.Filesize,
			Start: view.Start, Stop: view.Stop, Status: view.Status, Step: view.Step,
			ErrCode: view.ErrCode, ErrDetails: view.ErrDetails, TransferInfo: view.TransferInfo,
		},
		Rule:          &model.Rule{Name: view.Rule, IsSend: view.IsSend},
		Client:        &model.Client{Name: view.Client},
		RemoteAgent:   &model.RemoteAgent{},
		RemoteAccount: &model.RemoteAccount{},
		LocalAgent:    &model.LocalAgent{},
		LocalAccount:  &model.LocalAccount{},
		Paths:         &h.db.Config.Paths,
	}

	var rule model.Rule
	if h.db.Get(&rule, "name=? AND is_send=?", view.Rule, view.IsSend).Run() == nil {
		transCtx.Rule = &rule
	}

	if !view.IsServer {
		transCtx.RemoteAgent.Name, transCtx.RemoteAccount.Login = view.Agent, view.Account

		return transCtx
	}

	transCtx.Transfer.LocalAccountID.Valid = true
	transCtx.LocalAgent.Name, transCtx.LocalAccount.Login = view.Agent, view.Account

	var agent model.LocalAgent
	if h.db.Get(&agent, "name=?", view.Agent).Run() == nil {
		transCtx.LocalAgent = &agent
	}

	return transCtx
}

// exampleContext is a fictitious transfer in error, received by a rule that
// uses the template (or an example rule), with the directories of the
// gateway.
func (h *Handlers) exampleContext(translate gui.Translator, uses []ruleuse.Use) *model.TransferContext {
	paths := &h.db.Config.Paths
	file := translate("email_templates.example.file")
	partner := translate("email_templates.example.partner")

	rule := &model.Rule{Name: translate("email_templates.example.rule")}
	if len(uses) > 0 {
		var stored model.Rule
		if h.db.Get(&stored, "id=?", uses[0].RuleID).Run() == nil {
			rule = &stored
		}
	}

	return &model.TransferContext{
		Transfer: &model.Transfer{
			ID: exampleID, RemoteTransferID: exampleRemoteID, SrcFilename: strings.ToUpper(file), DestFilename: file,
			LocalPath: path.Join(paths.GatewayHome, paths.DefaultInDir, file), RemotePath: "/" + strings.ToUpper(file),
			Filesize: exampleSize, Start: time.Now().Add(-exampleAge), Status: types.StatusError,
			ErrCode: types.TeConnection, ErrDetails: translate("email_templates.example.error"),
			TransferInfo: map[string]any{},
		},
		Rule:          rule,
		Client:        &model.Client{Name: translate("email_templates.example.client")},
		RemoteAgent:   &model.RemoteAgent{Name: partner},
		RemoteAccount: &model.RemoteAccount{Login: translate("email_templates.example.account")},
		LocalAgent:    &model.LocalAgent{},
		LocalAccount:  &model.LocalAccount{},
		Paths:         paths,
	}
}

// wrapMarkers surrounds every marker of a text with the highlighting
// characters, which the substitution leaves around its value.
func wrapMarkers(text string) string {
	return markerPattern.ReplaceAllString(text, markOpen+"$0"+markClose)
}

// segments splits a substituted text into its values (highlighted) and the
// text around them.
func segments(text string) []segment {
	var out []segment

	for text != "" {
		before, rest, found := strings.Cut(text, markOpen)
		if before != "" {
			out = append(out, segment{Text: before})
		}

		if !found {
			break
		}

		value, after, _ := strings.Cut(rest, markClose)
		out = append(out, segment{Text: value, Mark: true})
		text = after
	}

	return out
}

// previewStyle is the style of the document of the body: the font an e-mail
// client shows by default, the colours of the kit (tokens.css, linked).
const previewStyle = "body{margin:0;padding:12px 16px;font:14px/1.5 Arial,Helvetica,sans-serif;color:var(--tp)}" +
	"p{margin:0 0 .7em}h2{font-size:18px;margin:.6em 0 .3em}.plain{white-space:pre-wrap;overflow-wrap:anywhere}" +
	"mark{padding:0 4px;border:1px solid var(--wo-aa);border-radius:var(--r-xs);" +
	"background:var(--wo-bg);color:var(--wo-aa);font-weight:700}"

// bodyDocument is the document the isolated frame of the preview shows: an
// HTML body as it is (its values highlighted in its text only), a text body
// as text, with its line breaks.
func bodyDocument(body string, isHTMLBody bool) string {
	var b strings.Builder

	b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><link rel="stylesheet" href="`)
	b.WriteString(html.EscapeString(common.Options().StaticURL("tokens.css")))
	b.WriteString(`"><style>` + previewStyle + `</style></head><body>`)

	if isHTMLBody {
		markHTML(&b, body)
	} else {
		b.WriteString(`<div class="plain">`)
		writeMarked(&b, body)
		b.WriteString(`</div>`)
	}

	b.WriteString(`</body></html>`)

	return b.String()
}

// writeMarked writes a text escaped, its values highlighted.
func writeMarked(b *strings.Builder, text string) {
	for _, s := range segments(text) {
		if s.Mark {
			b.WriteString("<mark>" + html.EscapeString(s.Text) + "</mark>")
		} else {
			b.WriteString(html.EscapeString(s.Text))
		}
	}
}

// markHTML writes an HTML body with its values highlighted in its text; in
// the attributes, and in a style or a script, the highlighting is dropped.
func markHTML(out *strings.Builder, body string) {
	unmark := strings.NewReplacer(markOpen, "", markClose, "")
	tokens := xhtml.NewTokenizer(strings.NewReader(body))
	raw := false

	for {
		kind := tokens.Next()
		if kind == xhtml.ErrorToken {
			if tokens.Err() != io.EOF {
				out.WriteString(html.EscapeString(unmark.Replace(string(tokens.Raw()))))
			}

			return
		}

		tok := tokens.Token()

		switch kind {
		case xhtml.TextToken:
			if raw {
				out.WriteString(unmark.Replace(tok.Data))
			} else {
				writeMarked(out, tok.Data)
			}
		case xhtml.StartTagToken, xhtml.EndTagToken, xhtml.SelfClosingTagToken:
			raw = kind == xhtml.StartTagToken && (tok.Data == "style" || tok.Data == "script")
			for i := range tok.Attr {
				tok.Attr[i].Val = unmark.Replace(tok.Attr[i].Val)
			}

			out.WriteString(tok.String())
		default:
			out.WriteString(unmark.Replace(tok.String()))
		}
	}
}

// Preview serves POST /webui/email-templates/preview: the preview of the
// template the form holds, for the script of the screen, which updates it as
// the user types. It writes nothing.
func (h *Handlers) Preview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	translate := h.shell.Translator(r)
	form := newForm(translate, nil, "")
	form.read(r.PostForm)

	var uses []ruleuse.Use

	if id, err := strconv.ParseInt(r.PostForm.Get(fieldID), 10, 64); err == nil {
		var stored model.EmailTemplate
		if h.db.Get(&stored, "id=?", id).Run() == nil {
			all, usesErr := h.uses()
			if usesErr != nil {
				h.logger.Errorf("failed to read the rules that use the email template %q: %v", stored.Name, usesErr)
			}

			uses = all[stored.Name]
		}
	}

	tpl := form.template()
	data := h.preview(translate, &tpl, uses, r.PostForm.Get(valuesParam), r.PostForm.Get(markParam) != "")
	h.shell.RenderFragment(w, h.logger, r, previewOutput(data))
}
