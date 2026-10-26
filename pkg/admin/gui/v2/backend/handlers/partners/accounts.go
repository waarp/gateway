package partners

// The remote accounts of a partner: the "Remote accounts" tab, the sheet of
// an account (its details; its means are served by creds.go), the form that
// creates one with its first authentication means or edits its login, and
// the confirmation of a deletion, refused while running transfers or a
// folder watcher use the account.

import (
	"net/http"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const fieldLogin = "login"

//nolint:gochecknoglobals // reference data of the screen
var accountsCounter = liststate.Counter{
	One: "remote_accounts.count_one", Many: "remote_accounts.count",
	ShownOne: "remote_accounts.count_shown_one", ShownMany: "remote_accounts.count_shown",
}

// accountEntry is an account with the kinds of its means.
type accountEntry struct {
	acc   *model.RemoteAccount
	types []string
}

// authText lists the kinds of means of the account, or nothing.
func (e accountEntry) authText(translate gui.Translator) string {
	return kindsText(translate, e.types)
}

// kindsText lists kinds of means, in the order given.
func kindsText(translate gui.Translator, types []string) string {
	labels := make([]string, 0, len(types))
	for _, typ := range types {
		labels = append(labels, means.TypeLabel(translate, typ))
	}

	return strings.Join(labels, ", ")
}

// accountsOf lists the accounts of a partner with the kinds of their means,
// in two queries.
func (h *Handlers) accountsOf(partnerID int64) ([]accountEntry, error) {
	var accounts model.RemoteAccounts
	if err := h.db.Select(&accounts).Where("remote_agent_id=?", partnerID).OrderBy("login", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller
	}

	var creds model.Credentials
	if err := h.db.Select(&creds).Where("remote_account_id IN (SELECT id FROM "+model.TableRemAccounts+
		" WHERE remote_agent_id=?)", partnerID).OrderBy("id", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller
	}

	kinds := map[int64][]string{}
	for _, c := range creds {
		kinds[c.RemoteAccountID.Int64] = append(kinds[c.RemoteAccountID.Int64], c.Type)
	}

	entries := make([]accountEntry, 0, len(accounts))
	for _, acc := range accounts {
		entries = append(entries, accountEntry{acc: acc, types: kinds[acc.ID]})
	}

	return entries, nil
}

// authOf lists the kinds of means of an account, or that it has none.
func (h *Handlers) authOf(translate gui.Translator, acc *model.RemoteAccount) (string, error) {
	creds, err := acc.GetCredentials(h.db)
	if err != nil {
		return "", err //nolint:wrapcheck // logged by the caller
	}

	if len(creds) == 0 {
		return translate("remote_account.detail.auth_none"), nil
	}

	types := make([]string, 0, len(creds))
	for _, c := range creds {
		types = append(types, c.Type)
	}

	return kindsText(translate, types), nil
}

// ---------------------------------------------------------------- list ----

type accountRow struct {
	ID      int64
	Login   string
	Auth    string     // the kinds of means
	NoAuth  gui.Status // the warning of an account without any means
	URL     string
	Actions []gui.Action
}

type accountsListData struct {
	List    tabList
	Title   string
	Tabs    []tab
	BackURL string
	AddURL  string
	Rows    []accountRow
}

// accountActions is the "⋯" menu of an account: view and edit in the list,
// delete (delete right) everywhere.
func accountActions(translate gui.Translator, agent string, pid int64, acc *model.RemoteAccount, user rights,
	onDetail bool, carried string,
) []gui.Action {
	aid := acc.ID
	var acts []gui.Action

	if !onDetail {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("remote_accounts.action.detail"), URL: withRet(accountURL(pid, aid, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon:  "pen",
				Label: translate("remote_accounts.action.edit"),
				URL:   withRet(accountURL(pid, aid, editSuffix), carried),
			})
		}
	}

	if user.Transfers {
		acts = append(acts, gui.Action{
			Icon: "clock", Label: translate("remote_accounts.action.transfers"), URL: transfersOf(agent, acc.Login),
		})
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("remote_accounts.action.delete"),
			URL:    withRet(accountURL(pid, aid, deleteSuffix), carried),
			Danger: true,
		})
	}

	return acts
}

// buildAccountsList computes the "Remote accounts" tab: the accounts
// searched by login, sorted by login or by their means, paged.
func buildAccountsList(translate gui.Translator, item entry, entries []accountEntry, query *listQuery,
	user rights, carried string,
) accountsListData {
	pid := item.agent.ID
	base := partnerURL(pid, accountsSuffix)
	data := accountsListData{
		List:    liststate.NewTabList(base, query, carried),
		Title:   translate("partners.detail.title", "name", item.agent.Name),
		BackURL: listURL(carried),
	}

	if user.Write {
		data.AddURL = withRet(base+newSuffix, carried)
	}

	kept := make([]accountEntry, 0, len(entries))

	for _, entry := range entries {
		data.List.Names = append(data.List.Names, entry.acc.Login)

		if query.Search.Matches(entry.acc.Login) {
			kept = append(kept, entry)
		}
	}

	slices.Sort(data.List.Names)
	sortAccountEntries(translate, kept, query.Sort, query.Dir)

	data.List.Count = accountsCounter.Text(translate, len(entries), len(kept))

	switch {
	case len(entries) == 0:
		data.List.Empty = true

		return data
	case len(kept) == 0:
		data.List.SetNoMatch(translate("remote_accounts.none.title", "query", query.Q), withRet(base, carried),
			translate("search.clear"))

		return data
	}

	b := query.Bounds(len(kept))
	data.List.Caption = translate("remote_accounts.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate("remote_accounts.sort."+query.Sort))

	for _, entry := range kept[b.From:b.To] {
		row := accountRow{
			ID:      entry.acc.ID,
			Login:   entry.acc.Login,
			Auth:    entry.authText(translate),
			URL:     withRet(accountURL(pid, entry.acc.ID, ""), carried),
			Actions: accountActions(translate, item.agent.Name, pid, entry.acc, user, false, carried),
		}

		if len(entry.types) == 0 {
			row.NoAuth = gui.Status{State: stateWarn, Label: translate("remote_accounts.auth_none")}
		}

		data.Rows = append(data.Rows, row)
	}

	foot := translate("remote_accounts.pager", "from", b.From+1, "to", b.To, "total", len(kept))
	data.List.Foot(foot, b, query, base, carried)

	return data
}

// sortAccountEntries sorts by login or by the kinds of means; ties keep the login order.
func sortAccountEntries(translate gui.Translator, entries []accountEntry, key, dir string) {
	byLogin := func(a, b accountEntry) int {
		return strings.Compare(strings.ToLower(a.acc.Login), strings.ToLower(b.acc.Login))
	}

	slices.SortStableFunc(entries, func(a, b accountEntry) int {
		order := byLogin(a, b)
		if key == sortAuth {
			order = strings.Compare(a.authText(translate), b.authText(translate))
		}

		if dir == dirDesc {
			order = -order
		}

		if order == 0 {
			return byLogin(a, b)
		}

		return order
	})
}

// Accounts serves GET /webui/partners/{id}/accounts.
func (h *Handlers) Accounts(w http.ResponseWriter, r *http.Request) {
	e, ok := h.partner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	entries, err := h.accountsOf(e.agent.ID)
	if err != nil {
		h.logger.Errorf("failed to list the accounts of the partner %d: %v", e.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := buildAccountsList(translate, e, entries, parseQuery(r.URL.Query(), accountSortKeys), rightsOf(r), carried)

	if data.Tabs, err = h.partnerTabs(translate, e, tabAccounts, carried); err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	own := owner{ptn: e}
	if liststate.IsListFragment(r) {
		h.shell.RenderFragment(w, h.logger, r, accountsFragment(data))

		return
	}

	screen := h.sheetScreen(translate, data.Title, own, "", carried, 0, notices(translate, r.URL.Query()))
	screen.Head = gui.LiveAssets()
	h.shell.Render(w, h.logger, r, screen, accountsView(data))
}

// -------------------------------------------------------------- detail ----

type accountDetailData struct {
	Title    string
	Login    string
	Tabs     []tab
	BackURL  string
	EditURL  string
	Actions  []gui.Action
	Identity []kv
}

// AccountDetail serves GET /webui/partners/{id}/accounts/{aid}.
func (h *Handlers) AccountDetail(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)
	pid, aid := own.ptn.agent.ID, own.acc.ID

	auth, err := h.authOf(translate, own.acc)
	if err != nil {
		h.logger.Errorf("failed to list the authentication means of the account %d: %v", aid, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := accountDetailData{
		Title:   own.title(translate),
		Login:   own.acc.Login,
		BackURL: withRet(partnerURL(pid, accountsSuffix), carried),
		Actions: accountActions(translate, own.ptn.agent.Name, pid, own.acc, user, true, carried),
		Identity: []kv{
			{Label: translate("remote_account.detail.partner"), Value: own.ptn.agent.Name + " (" + own.ptn.proto.Label + ")"},
			{Label: translate("remote_account.detail.login"), Value: own.acc.Login, Mono: true},
			{Label: translate("remote_account.detail.auth"), Value: auth},
		},
	}

	if user.Write {
		data.EditURL = withRet(accountURL(pid, aid, editSuffix), carried)
	}

	if data.Tabs, err = h.accountTabs(translate, own, tabDetails, carried); err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	screen := h.sheetScreen(translate, data.Title, own, "", carried, 0, notices(translate, r.URL.Query()))
	h.shell.Render(w, h.logger, r, screen, accountDetailView(data))
}

// ---------------------------------------------------------------- form ----

// accountFormData is the form of an account: its login and, at creation,
// its first means (Driss, 2026-10-18, for the local accounts: the account
// and the means on the same page, so that the link between them is plain;
// the next means are added in the sheet of the account).
type accountFormData struct {
	Title     string
	Action    string
	CancelURL string
	Partner   string
	Tabs      []tab
	Edit      bool
	HasErrors bool
	Refused   string
	Login     field
	Means     *means.Form // nil on edition
}

func (h *Handlers) newAccountForm(translate gui.Translator, e entry, acc *model.RemoteAccount,
	carried string,
) *accountFormData {
	pid := e.agent.ID

	if acc == nil {
		return &accountFormData{
			Title:     translate("remote_accounts.new.title"),
			Action:    withRet(partnerURL(pid, accountsSuffix+newSuffix), carried),
			CancelURL: withRet(partnerURL(pid, accountsSuffix), carried),
			Partner:   e.agent.Name,
			Means:     means.FirstMeans(translate, means.RemoteAccountTypes(e.agent.Protocol), ""),
		}
	}

	return &accountFormData{
		Title:     translate("remote_accounts.edit.title", "login", acc.Login),
		Action:    withRet(accountURL(pid, acc.ID, editSuffix), carried),
		CancelURL: withRet(accountURL(pid, acc.ID, ""), carried),
		Partner:   e.agent.Name,
		Edit:      true,
		Login:     field{Value: acc.Login},
	}
}

// read takes the posted login into the form and the account.
func (f *accountFormData) read(translate gui.Translator, r *http.Request, acc *model.RemoteAccount) {
	f.Login.Value = strings.TrimSpace(r.FormValue(fieldLogin))

	if f.Login.Value == "" {
		f.Login.Error = translate("remote_accounts.form.error.login")
	}

	acc.Login = f.Login.Value
	f.HasErrors = f.Login.Error != ""
}

// refuse marks the form with the refusal of the model.
func (f *accountFormData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	if strings.Contains(msg, "same login") {
		f.Login.Error = translate("remote_accounts.form.error.taken")
	} else {
		f.Refused = msg
	}

	f.HasErrors = true
}

// NewAccountForm serves GET /webui/partners/{id}/accounts/new.
func (h *Handlers) NewAccountForm(w http.ResponseWriter, r *http.Request) {
	e, ok := h.partner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	h.accountFormPage(w, r, owner{ptn: e}, h.newAccountForm(translate, e, nil, ret(r)), 0)
}

// EditAccountForm serves GET /webui/partners/{id}/accounts/{aid}/edit.
func (h *Handlers) EditAccountForm(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	h.accountFormPage(w, r, own, h.newAccountForm(translate, own.ptn, own.acc, ret(r)), 0)
}

// accountFormPage renders the form of an account: under the partner tabs at
// creation, under the account tabs on edition.
func (h *Handlers) accountFormPage(w http.ResponseWriter, r *http.Request, own owner, f *accountFormData, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	current := tabAccounts
	if f.Edit {
		current = tabDetails
	}

	tabs, err := h.tabsOf(translate, own, current, carried)
	if err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	f.Tabs = tabs
	screen := h.sheetScreen(translate, f.Title, own, f.Title, carried, status, nil)
	h.shell.Render(w, h.logger, r, screen, accountFormView(f))
}

// CreateAccount serves POST /webui/partners/{id}/accounts/new: the account
// and its first means, if any, are saved together or not at all.
func (h *Handlers) CreateAccount(w http.ResponseWriter, r *http.Request) {
	ptn, ok := h.partner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := means.ParseForm(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newAccountForm(translate, ptn, nil, carried)
	acc := &model.RemoteAccount{RemoteAgentID: ptn.agent.ID}
	form.read(translate, r, acc)

	own := owner{ptn: ptn, acc: acc}

	var cred *model.Credential

	if form.Means.Chosen(r) {
		cred = &model.Credential{}

		if err := form.Means.Read(translate, r, h.meansOwner(translate, own, carried, rightsOf(r)), cred, nil); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)

			return
		}

		form.HasErrors = form.HasErrors || form.Means.HasErrors
	}

	if !form.HasErrors {
		credRefused := false

		err := h.db.Transaction(func(ses *database.Session) error {
			if err := ses.Insert(acc).Run(); err != nil {
				return err //nolint:wrapcheck // the cause is mapped onto the form
			}

			if cred == nil {
				return nil
			}

			acc.SetCredOwner(cred)

			if err := ses.Insert(cred).Run(); err != nil {
				credRefused = true

				return err //nolint:wrapcheck // the cause is mapped onto the form
			}

			return nil
		})

		switch {
		case err == nil:
			// Back to the accounts tab, where the new account shows with its
			// means.
			redirectSheet(w, r, withRet(partnerURL(ptn.agent.ID, accountsSuffix), carried), kindAccount, doneCreated, acc.Login)

			return
		case database.IsValidationError(err) && credRefused:
			form.Means.Refuse(translate, err)
			form.HasErrors = true
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to create the account %q: %v", acc.Login, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	form.Means.Types = means.TypeChoices(translate, means.RemoteAccountTypes(ptn.agent.Protocol), form.Means.Type, true)
	h.accountFormPage(w, r, owner{ptn: ptn}, form, http.StatusUnprocessableEntity)
}

// UpdateAccount serves POST /webui/partners/{id}/accounts/{aid}/edit.
func (h *Handlers) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := means.ParseForm(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newAccountForm(translate, own.ptn, own.acc, carried)
	form.read(translate, r, own.acc)

	if !form.HasErrors {
		err := h.db.Update(own.acc).Run()

		switch {
		case err == nil:
			target := withRet(accountURL(own.ptn.agent.ID, own.acc.ID, ""), carried)
			redirectSheet(w, r, target, kindAccount, doneUpdated, own.acc.Login)

			return
		case database.IsValidationError(err):
			form.refuse(translate, err)
		default:
			h.logger.Errorf("failed to save the account %q: %v", own.acc.Login, err)
			http.Error(w, "Internal error", http.StatusInternalServerError)

			return
		}
	}

	h.accountFormPage(w, r, own, form, http.StatusUnprocessableEntity)
}

// -------------------------------------------------------------- delete ----

type accountDeleteData struct {
	Title        string
	Login        string
	Partner      string
	Auth         string
	Tabs         []tab
	CancelURL    string
	Action       string
	TransfersURL string
	// The model refuses the deletion: running transfers use the account
	// (Transfers), or a folder watcher does (Watchers).
	Transfers bool
	Watchers  bool
}

func (d *accountDeleteData) blocked() bool { return d.Transfers || d.Watchers }

// accountBlockers tells what refuses the deletion of the account, from the
// model's own rule (running transfers) and from the folder watchers that
// name it (the foreign key restricts the deletion).
func (h *Handlers) accountBlockers(acc *model.RemoteAccount) (transfers, watchers bool, err error) {
	if delErr := acc.BeforeDelete(h.db); delErr != nil {
		if !database.IsValidationError(delErr) {
			return false, false, delErr //nolint:wrapcheck // logged by the caller
		}

		transfers = true
	}

	n, err := h.db.Count(&model.FileWatcher{}).Where("remote_account_id=?", acc.ID).Run()
	if err != nil {
		return false, false, err //nolint:wrapcheck // logged by the caller
	}

	return transfers, n > 0, nil
}

func (h *Handlers) accountDeletePage(w http.ResponseWriter, r *http.Request, own owner, transfers, watchers bool,
	status int,
) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	pid, aid := own.ptn.agent.ID, own.acc.ID

	auth, err := h.authOf(translate, own.acc)
	if err != nil {
		h.logger.Errorf("failed to list the authentication means of the account %d: %v", aid, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := accountDeleteData{
		Title:        translate("remote_accounts.delete.title", "login", own.acc.Login),
		Login:        own.acc.Login,
		Partner:      own.ptn.agent.Name,
		Auth:         auth,
		CancelURL:    withRet(accountURL(pid, aid, ""), carried),
		Action:       withRet(accountURL(pid, aid, deleteSuffix), carried),
		TransfersURL: transfersURL(),
		Transfers:    transfers,
		Watchers:     watchers,
	}

	if data.Tabs, err = h.accountTabs(translate, own, tabDetails, carried); err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	screen := h.sheetScreen(translate, data.Title, own, data.Title, carried, status, nil)
	h.shell.Render(w, h.logger, r, screen, accountDeleteView(data))
}

// DeleteAccountForm serves GET /webui/partners/{id}/accounts/{aid}/delete:
// the confirmation page, or the refusal when the account is in use.
func (h *Handlers) DeleteAccountForm(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	transfers, watchers, err := h.accountBlockers(own.acc)
	if err != nil {
		h.logger.Errorf("failed to check the use of the account %d: %v", own.acc.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.accountDeletePage(w, r, own, transfers, watchers, 0)
}

// DeleteAccount serves POST /webui/partners/{id}/accounts/{aid}/delete. The
// means of the account go with it.
func (h *Handlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	transfers, watchers, err := h.accountBlockers(own.acc)
	if err == nil && !transfers && !watchers {
		err = h.db.Delete(own.acc).Run()
	}

	switch {
	case err == nil && !transfers && !watchers:
		target := withRet(partnerURL(own.ptn.agent.ID, accountsSuffix), ret(r))
		redirectSheet(w, r, target, kindAccount, doneDeleted, own.acc.Login)
	case err == nil || database.IsValidationError(err):
		h.accountDeletePage(w, r, own, transfers || database.IsValidationError(err), watchers, http.StatusUnprocessableEntity)
	default:
		h.logger.Errorf("failed to delete the account %d: %v", own.acc.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}
