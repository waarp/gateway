package means

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type row struct {
	ID      int64
	Name    string
	Type    string
	Summary string
	Expiry  gui.Status
	URL     string
	Actions []gui.Action
}

type listData struct {
	List         liststate.TabList
	Title        string
	Intro        string
	Tabs         []Tab
	BackURL      string
	BackLabel    string
	TabsLabel    string
	AddURL       string // empty without the write right or when nothing can be added
	NonePossible bool   // the protocol asks no means from the owner
	NoneTitle    string
	NoneText     string
	EmptyText    string
	Rows         []row
}

// actions is the "⋯" menu of a means: view, edit (write right), delete
// (delete right).
func actions(translate gui.Translator, own *Owner, id int64) []gui.Action {
	acts := []gui.Action{{Icon: "eye", Label: translate("auth.action.detail"), URL: own.WithRet(own.CredURL(id, ""))}}

	if own.Rights.Write {
		acts = append(acts, gui.Action{
			Icon: "pen", Label: translate("auth.action.edit"), URL: own.WithRet(own.CredURL(id, EditSuffix)),
		})
	}

	if own.Rights.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("auth.action.delete"),
			URL:    own.WithRet(own.CredURL(id, DeleteSuffix)),
			Danger: true,
		})
	}

	return acts
}

// buildList computes the "Authentication" tab of an owner: the means
// searched by name, sorted, paged, each one with its non-secret summary.
func buildList(translate gui.Translator, own *Owner, creds []*model.Credential, query *liststate.Query,
	now time.Time,
) listData {
	base := own.AuthURL
	data := listData{
		List:         liststate.NewTabList(base, query, own.Carried),
		Title:        own.Title,
		Intro:        translate(own.Texts.Intro),
		BackURL:      own.BackURL,
		BackLabel:    translate(own.Texts.Back),
		TabsLabel:    translate(own.Texts.Tabs),
		NonePossible: len(own.Allowed) == 0,
		NoneTitle:    translate(own.Texts.NoneTitle),
		NoneText:     translate(own.Texts.NoneText),
		EmptyText:    translate(own.Texts.Empty),
	}

	if own.Rights.Write && len(own.Remaining(creds)) > 0 {
		data.AddURL = own.WithRet(base + NewSuffix)
	}

	views := make([]view, 0, len(creds))
	needle := strings.ToLower(query.Q)

	for _, c := range creds {
		data.List.Names = append(data.List.Names, c.Name)

		if needle == "" || strings.HasPrefix(strings.ToLower(c.Name), needle) {
			views = append(views, viewOf(c))
		}
	}

	slices.Sort(data.List.Names)
	sortViews(translate, views, query.Sort, query.Dir)

	data.List.Count = counter.Text(translate, len(creds), len(views))

	switch {
	case len(creds) == 0:
		data.List.Empty = true

		return data
	case len(views) == 0:
		data.List.SetNoMatch(translate("auth.none.title", "query", query.Q), own.WithRet(base), translate("search.clear"))

		return data
	}

	b := query.Bounds(len(views))
	data.List.Caption = translate("auth.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate("auth.sort."+query.Sort))

	for i := b.From; i < b.To; i++ {
		v := &views[i]
		data.Rows = append(data.Rows, row{
			ID:      v.ID,
			Name:    v.Name,
			Type:    TypeLabel(translate, v.Type),
			Summary: v.summary(translate),
			Expiry:  v.expiry(translate, now),
			URL:     own.WithRet(own.CredURL(v.ID, "")),
			Actions: actions(translate, own, v.ID),
		})
	}

	data.List.Foot(translate("auth.pager", "from", b.From+1, "to", b.To, "total", len(views)), b, query, base, own.Carried)

	return data
}

// sortViews sorts by name or by kind; ties keep the name order.
func sortViews(translate gui.Translator, views []view, key, dir string) {
	byName := func(a, b view) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) }

	slices.SortStableFunc(views, func(a, b view) int {
		order := byName(a, b)
		if key == sortType {
			order = strings.Compare(TypeLabel(translate, a.Type), TypeLabel(translate, b.Type))
		}

		if dir == liststate.DirDesc {
			order = -order
		}

		if order == 0 {
			return byName(a, b)
		}

		return order
	})
}

// Auth serves the "Authentication" tab of the owner: its means searched,
// sorted and paged, with the notifications of the previous action.
func (p *Pages) Auth(w http.ResponseWriter, r *http.Request, own *Owner, notices []gui.Notice) {
	translate := p.Shell.Translator(r)

	creds, err := own.Table.GetCredentials(p.DB)
	if err != nil {
		p.Logger.Errorf("failed to list the authentication means: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	query := liststate.Parse(r.URL.Query(), sortKeys, liststate.DefaultSizes)
	data := buildList(translate, own, creds, query, time.Now())

	tabs, ok := p.tabs(w, own)
	if !ok {
		return
	}

	data.Tabs = tabs
	p.Shell.Render(w, p.Logger, r, own.screen(data.Title, "", 0, notices), authView(data))
}
