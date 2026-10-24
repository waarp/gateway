package servers

// The local accounts of a server: the "Local accounts" tab, the sheet of an
// account (its details; its means are served by creds.go), the form that
// creates one with its first authentication means or edits one, and the
// confirmation of a deletion, refused while running transfers use the
// account.

import (
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"code.waarp.fr/lib/gui"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/means"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const (
	fieldLogin = "login"
	fieldIPs   = "ips"
)

//nolint:gochecknoglobals // reference data of the screen
var accountsCounter = counter{
	One: "accounts.count_one", Many: "accounts.count",
	ShownOne: "accounts.count_shown_one", ShownMany: "accounts.count_shown",
}

// accountEntry is an account with the kinds of its means.
type accountEntry struct {
	acc   *model.LocalAccount
	types []string
}

// authText lists the kinds of means of the account, or nothing.
func (e accountEntry) authText(translate gui.Translator) string {
	labels := make([]string, 0, len(e.types))
	for _, typ := range e.types {
		labels = append(labels, means.TypeLabel(translate, typ))
	}

	return strings.Join(labels, ", ")
}

// accountsOf lists the accounts of a server with the kinds of their means,
// in two queries.
func (h *Handlers) accountsOf(serverID int64) ([]accountEntry, error) {
	var accounts model.LocalAccounts
	if err := h.db.Select(&accounts).Where("local_agent_id=?", serverID).OrderBy("login", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller
	}

	var creds model.Credentials
	if err := h.db.Select(&creds).Where("local_account_id IN (SELECT id FROM "+model.TableLocAccounts+
		" WHERE local_agent_id=?)", serverID).OrderBy("id", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller
	}

	kinds := map[int64][]string{}
	for _, c := range creds {
		kinds[c.LocalAccountID.Int64] = append(kinds[c.LocalAccountID.Int64], c.Type)
	}

	entries := make([]accountEntry, 0, len(accounts))
	for _, acc := range accounts {
		entries = append(entries, accountEntry{acc: acc, types: kinds[acc.ID]})
	}

	return entries, nil
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
func accountActions(translate gui.Translator, sid, aid int64, user rights, onDetail bool, carried string) []gui.Action {
	var acts []gui.Action

	if !onDetail {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("accounts.action.detail"), URL: withRet(accountURL(sid, aid, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("accounts.action.edit"), URL: withRet(accountURL(sid, aid, editSuffix), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("accounts.action.delete"),
			URL:    withRet(accountURL(sid, aid, deleteSuffix), carried),
			Danger: true,
		})
	}

	return acts
}

// buildAccountsList computes the "Local accounts" tab: the accounts searched
// by login, sorted by login or by their means, paged.
func buildAccountsList(translate gui.Translator, e entry, entries []accountEntry, query *listQuery,
	user rights, carried string,
) accountsListData {
	sid := e.agent.ID
	base := serverURL(sid, accountsSuffix)
	data := accountsListData{
		List:    liststate.NewTabList(base, query, carried),
		Title:   translate("servers.detail.title", "name", e.agent.Name),
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
	sortAccounts(translate, kept, query.Sort, query.Dir)

	data.List.Count = accountsCounter.Text(translate, len(entries), len(kept))

	switch {
	case len(entries) == 0:
		data.List.Empty = true

		return data
	case len(kept) == 0:
		data.List.SetNoMatch(translate("accounts.none.title", "query", query.Q), withRet(base, carried),
			translate("search.clear"))

		return data
	}

	b := query.Bounds(len(kept))
	data.List.Caption = translate("accounts.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate("accounts.sort."+query.Sort))

	for _, entry := range kept[b.From:b.To] {
		row := accountRow{
			ID:      entry.acc.ID,
			Login:   entry.acc.Login,
			Auth:    entry.authText(translate),
			URL:     withRet(accountURL(sid, entry.acc.ID, ""), carried),
			Actions: accountActions(translate, sid, entry.acc.ID, user, false, carried),
		}

		if len(entry.types) == 0 {
			row.NoAuth = gui.Status{State: stateWarn, Label: translate("accounts.auth_none")}
		}

		data.Rows = append(data.Rows, row)
	}

	data.List.Foot(translate("accounts.pager", "from", b.From+1, "to", b.To, "total", len(kept)), b, query, base, carried)

	return data
}

// sortAccounts sorts by login or by the kinds of means; ties keep the login order.
func sortAccounts(translate gui.Translator, entries []accountEntry, key, dir string) {
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

// Accounts serves GET /webui/servers/{id}/accounts.
func (h *Handlers) Accounts(w http.ResponseWriter, r *http.Request) {
	srv, ok := h.server(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	entries, err := h.accountsOf(srv.agent.ID)
	if err != nil {
		h.logger.Errorf("failed to list the accounts of the server %d: %v", srv.agent.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	data := buildAccountsList(translate, srv, entries, parseQuery(r.URL.Query(), accountSortKeys), rightsOf(r), carried)

	if data.Tabs, err = h.serverTabs(translate, srv, tabAccounts, carried); err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	own := owner{srv: srv}
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

// ipsText lists the allowed addresses of an account, or "all".
func ipsText(translate gui.Translator, acc *model.LocalAccount) string {
	if len(acc.IPAddresses) == 0 {
		return translate("account.detail.ips_all")
	}

	return strings.Join(acc.IPAddresses, ", ")
}

// AccountDetail serves GET /webui/servers/{id}/accounts/{aid}.
func (h *Handlers) AccountDetail(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)
	sid, aid := own.srv.agent.ID, own.acc.ID

	creds, err := own.acc.GetCredentials(h.db)
	if err != nil {
		h.logger.Errorf("failed to list the authentication means of the account %d: %v", aid, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	auth := translate("account.detail.auth_none")
	if len(creds) > 0 {
		labels := make([]string, 0, len(creds))
		for _, c := range creds {
			labels = append(labels, means.TypeLabel(translate, c.Type))
		}

		auth = strings.Join(labels, ", ")
	}

	data := accountDetailData{
		Title:   own.title(translate),
		Login:   own.acc.Login,
		BackURL: withRet(serverURL(sid, accountsSuffix), carried),
		Actions: accountActions(translate, sid, aid, user, true, carried),
		Identity: []kv{
			{Label: translate("account.detail.server"), Value: own.srv.agent.Name + " (" + own.srv.proto.Label + ")"},
			{Label: translate("account.detail.login"), Value: own.acc.Login, Mono: true},
			{Label: translate("account.detail.ips"), Value: ipsText(translate, own.acc), Mono: len(own.acc.IPAddresses) > 0},
			{Label: translate("account.detail.auth"), Value: auth},
		},
	}

	if user.Write {
		data.EditURL = withRet(accountURL(sid, aid, editSuffix), carried)
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

// accountFormData is the form of an account: its login, its allowed
// addresses and, at creation, its first means (Driss, 2026-10-18: the
// account and the means on the same page, so that the link between them is
// plain; the next means are added in the sheet of the account).
type accountFormData struct {
	Title     string
	Action    string
	CancelURL string
	Server    string
	Tabs      []tab
	Edit      bool
	HasErrors bool
	Refused   string
	Login     field
	IPs       field
	Means     *means.Form // nil on edition
}

func (h *Handlers) newAccountForm(translate gui.Translator, srv entry, acc *model.LocalAccount,
	carried string,
) *accountFormData {
	sid := srv.agent.ID

	if acc == nil {
		return &accountFormData{
			Title:     translate("accounts.new.title"),
			Action:    withRet(serverURL(sid, accountsSuffix+newSuffix), carried),
			CancelURL: withRet(serverURL(sid, accountsSuffix), carried),
			Server:    srv.agent.Name,
			Means:     means.FirstMeans(translate, means.LocalAccountTypes(srv.agent.Protocol), ""),
		}
	}

	return &accountFormData{
		Title:     translate("accounts.edit.title", "login", acc.Login),
		Action:    withRet(accountURL(sid, acc.ID, editSuffix), carried),
		CancelURL: withRet(accountURL(sid, acc.ID, ""), carried),
		Server:    srv.agent.Name,
		Edit:      true,
		Login:     field{Value: acc.Login},
		IPs:       field{Value: strings.Join(acc.IPAddresses, "\n")},
	}
}

// read takes the posted login and addresses into the form and the account.
func (f *accountFormData) read(translate gui.Translator, r *http.Request, acc *model.LocalAccount) {
	f.Login.Value = strings.TrimSpace(r.FormValue(fieldLogin))
	f.IPs.Value = r.FormValue(fieldIPs)

	if f.Login.Value == "" {
		f.Login.Error = translate("accounts.form.error.login")
	}

	ips, bad := parseIPs(f.IPs.Value)
	if bad != "" {
		f.IPs.Error = translate("accounts.form.error.ip", "value", bad)
	}

	acc.Login = f.Login.Value
	acc.IPAddresses = ips
	f.HasErrors = f.Login.Error != "" || f.IPs.Error != ""
}

// parseIPs reads the allowed addresses, one per line (commas and spaces
// also separate them), in their canonical form; the first value that is
// not an address is returned.
func parseIPs(text string) (types.IPList, string) {
	var list types.IPList

	for raw := range strings.FieldsFuncSeq(text, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ';' || r == ' ' || r == '\t'
	}) {
		addr, err := netip.ParseAddr(raw)
		if err != nil {
			return nil, raw
		}

		list.Add(addr.String())
	}

	return list, ""
}

// refuse marks the form with the refusal of the model.
func (f *accountFormData) refuse(translate gui.Translator, err error) {
	msg := err.Error()

	switch {
	case strings.Contains(msg, "same login"):
		f.Login.Error = translate("accounts.form.error.taken")
	case strings.Contains(msg, "invalid account IP address"):
		f.IPs.Error = msg
	default:
		f.Refused = msg
	}

	f.HasErrors = true
}

// NewAccountForm serves GET /webui/servers/{id}/accounts/new.
func (h *Handlers) NewAccountForm(w http.ResponseWriter, r *http.Request) {
	srv, ok := h.server(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	h.accountFormPage(w, r, owner{srv: srv}, h.newAccountForm(translate, srv, nil, ret(r)), 0)
}

// EditAccountForm serves GET /webui/servers/{id}/accounts/{aid}/edit.
func (h *Handlers) EditAccountForm(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	h.accountFormPage(w, r, own, h.newAccountForm(translate, own.srv, own.acc, ret(r)), 0)
}

// accountFormPage renders the form of an account: under the server tabs at
// creation, under the account tabs on edition.
func (h *Handlers) accountFormPage(w http.ResponseWriter, r *http.Request, own owner, f *accountFormData, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)

	tabs, err := h.tabsOf(translate, own, map[bool]string{true: tabDetails, false: tabAccounts}[f.Edit], carried)
	if err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	f.Tabs = tabs
	screen := h.sheetScreen(translate, f.Title, own, f.Title, carried, status, nil)
	h.shell.Render(w, h.logger, r, screen, accountFormView(f))
}

// CreateAccount serves POST /webui/servers/{id}/accounts/new: the account
// and its first means, if any, are saved together or not at all.
func (h *Handlers) CreateAccount(w http.ResponseWriter, r *http.Request) {
	srv, ok := h.server(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)

	if err := means.ParseForm(r); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)

		return
	}

	form := h.newAccountForm(translate, srv, nil, carried)
	acc := &model.LocalAccount{LocalAgentID: srv.agent.ID}
	form.read(translate, r, acc)

	own := owner{srv: srv, acc: acc}

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
			// means (Driss, 2026-10-18).
			redirectSheet(w, r, withRet(serverURL(srv.agent.ID, accountsSuffix), carried), kindAccount, doneCreated, acc.Login)

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

	form.Means.Types = means.TypeChoices(translate, means.LocalAccountTypes(srv.agent.Protocol), form.Means.Type, true)
	h.accountFormPage(w, r, owner{srv: srv}, form, http.StatusUnprocessableEntity)
}

// UpdateAccount serves POST /webui/servers/{id}/accounts/{aid}/edit.
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

	form := h.newAccountForm(translate, own.srv, own.acc, carried)
	form.read(translate, r, own.acc)

	if !form.HasErrors {
		err := h.db.Update(own.acc).Run()

		switch {
		case err == nil:
			target := withRet(accountURL(own.srv.agent.ID, own.acc.ID, ""), carried)
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
	Server       string
	IPs          string
	Auth         string
	Tabs         []tab
	CancelURL    string
	Action       string
	TransfersURL string
	Blocked      bool // running transfers use the account: the model refuses
}

// accountBlocked tells whether the model refuses the deletion of the account
// (running transfers use it), from the model's own rule.
func (h *Handlers) accountBlocked(acc *model.LocalAccount) (bool, error) {
	err := acc.BeforeDelete(h.db)
	if err == nil {
		return false, nil
	}

	if database.IsValidationError(err) {
		return true, nil
	}

	return false, err //nolint:wrapcheck // logged by the caller
}

func (h *Handlers) accountDeletePage(w http.ResponseWriter, r *http.Request, own owner, blocked bool, status int) {
	translate := h.shell.Translator(r)
	carried := ret(r)
	sid, aid := own.srv.agent.ID, own.acc.ID

	creds, err := own.acc.GetCredentials(h.db)
	if err != nil {
		h.logger.Errorf("failed to list the authentication means of the account %d: %v", aid, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	auth := translate("account.detail.auth_none")
	if len(creds) > 0 {
		labels := make([]string, 0, len(creds))
		for _, c := range creds {
			labels = append(labels, means.TypeLabel(translate, c.Type))
		}

		auth = strings.Join(labels, ", ")
	}

	data := accountDeleteData{
		Title:        translate("accounts.delete.title", "login", own.acc.Login),
		Login:        own.acc.Login,
		Server:       own.srv.agent.Name,
		IPs:          ipsText(translate, own.acc),
		Auth:         auth,
		CancelURL:    withRet(accountURL(sid, aid, ""), carried),
		Action:       withRet(accountURL(sid, aid, deleteSuffix), carried),
		TransfersURL: transfersURL(),
		Blocked:      blocked,
	}

	if data.Tabs, err = h.accountTabs(translate, own, tabDetails, carried); err != nil {
		h.logger.Errorf("%v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	screen := h.sheetScreen(translate, data.Title, own, data.Title, carried, status, nil)
	h.shell.Render(w, h.logger, r, screen, accountDeleteView(data))
}

// DeleteAccountForm serves GET /webui/servers/{id}/accounts/{aid}/delete:
// the confirmation page, or the refusal when the account is in use.
func (h *Handlers) DeleteAccountForm(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	blocked, err := h.accountBlocked(own.acc)
	if err != nil {
		h.logger.Errorf("failed to check the transfers of the account %d: %v", own.acc.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.accountDeletePage(w, r, own, blocked, 0)
}

// DeleteAccount serves POST /webui/servers/{id}/accounts/{aid}/delete. The
// means of the account go with it.
func (h *Handlers) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	own, ok := h.owner(w, r)
	if !ok {
		return
	}

	if err := h.db.Delete(own.acc).Run(); err != nil {
		if database.IsValidationError(err) {
			h.accountDeletePage(w, r, own, true, http.StatusUnprocessableEntity)

			return
		}

		h.logger.Errorf("failed to delete the account %d: %v", own.acc.ID, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	target := withRet(serverURL(own.srv.agent.ID, accountsSuffix), ret(r))
	redirectSheet(w, r, target, kindAccount, doneDeleted, own.acc.Login)
}
