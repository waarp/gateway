// Package snmp serves the "SNMP" screen of the web interface
// (Administration): one page with the SNMP monitors (the monitoring tools
// the gateway notifies when a transfer fails or a service breaks down) and
// the SNMP agent (the configuration with which the gateway answers the
// requests of a monitoring tool), the sheet of a monitor in reading, the
// pages that create, edit and delete a monitor, the pages that configure and
// disable the agent, and the action that sends the test notification.
//
// A monitor holds a name, a version (SNMPv2c or SNMPv3), a UDP address and
// its security settings: a community in SNMPv2c; in SNMPv3 a security level,
// a user and, as the level asks, an authentication and a privacy protocol
// with their passphrases. The agent holds a listening address, a community,
// whether it accepts SNMPv3 only, and an SNMPv3 user with its protocols.
// The passphrases and the communities are never rendered back. Until this
// screen, both were managed through the REST API and the command line only.
package snmp

import (
	"cmp"
	"context"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/liststate"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/protoform"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	gwsnmp "code.waarp.fr/apps/gateway/gateway/pkg/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	// The stylesheet of the screen, under the web interface.
	stylesheet = "/static_v2/snmp/snmp.css"

	docPage      = "user/webui/snmp"
	pagePath     = "/snmp"
	monitorsPath = "/snmp/monitors"
	agentPath    = "/snmp/agent"

	// The cookie that keeps the state of the list of the monitors for the
	// browser session.
	stateCookie = "webui_snmp_list"

	// The parameters of the list state and of the result of the previous
	// action, shared by the list screens.
	doneParam   = liststate.DoneParam
	failedParam = liststate.FailedParam
	nameParam   = liststate.NameParam
	reasonParam = liststate.ReasonParam
	dirDesc     = liststate.DirDesc

	sortName    = "name"
	sortAddress = "address"

	doneCreated = liststate.DoneCreated
	doneUpdated = liststate.DoneUpdated
	doneDeleted = liststate.DoneDeleted
	// The results of the actions of the agent and of the test.
	doneAgentSaved    = "agent_saved"
	doneAgentDisabled = "agent_disabled"
	doneTested        = "tested"

	// The failures: the test notification, the reload of the SNMP service
	// once a configuration is saved.
	failedTest   = "test"
	failedReload = "reload"

	// The versions a monitor speaks, as the product stores them (SNMPv2 is
	// SNMPv2c; the product accepts both spellings).
	versionV2  = gwsnmp.Version2
	versionV2c = gwsnmp.Version2c
	versionV3  = gwsnmp.Version3

	// The default community of the product.
	defaultCommunity = "public"

	// Prefixes of the catalogue keys chosen at run time.
	sortKeyPrefix     = "snmp.sort."
	toastKeyPrefix    = "snmp.toast."
	securityKeyPrefix = "snmp.security."

	// Kinds of a gui.Notice.
	stateOK = "ok"
	stateKO = "ko"
)

//nolint:gochecknoglobals // reference data of the screen and of the product
var (
	sortKeys  = []string{sortName, sortAddress}
	doneKinds = []string{
		doneCreated, doneUpdated, doneDeleted, doneAgentSaved, doneAgentDisabled, doneTested,
	}

	// The SNMPv3 security levels, from the weakest; the empty one of the
	// product reads as the first.
	securityLevels = []string{gwsnmp.V3SecurityNoAuthNoPriv, gwsnmp.V3SecurityAuthNoPriv, gwsnmp.V3SecurityAuthPriv}

	// The protocols the product accepts, in the order of its command line;
	// the historic ones are said so in the lists.
	authProtocols = []string{
		gwsnmp.AuthMD5, gwsnmp.AuthSHA, gwsnmp.AuthSHA224, gwsnmp.AuthSHA256, gwsnmp.AuthSHA384, gwsnmp.AuthSHA512,
	}
	privProtocols = []string{
		gwsnmp.PrivDES, gwsnmp.PrivAES, gwsnmp.PrivAES192, gwsnmp.PrivAES192C, gwsnmp.PrivAES256, gwsnmp.PrivAES256C,
	}
	historicProtocols = []string{gwsnmp.AuthMD5, gwsnmp.AuthSHA, gwsnmp.PrivDES}

	listSpec = liststate.Spec{Path: pagePath, Cookie: stateCookie, Sorts: sortKeys, Sizes: liststate.DefaultSizes}
)

// service is what the screen asks of the SNMP service of the gateway: its
// state, the reload of the configurations once saved (as the REST API
// does), the test notification. The tests replace it.
type service interface {
	State() (utils.StateCode, string)
	ReloadMonitorsConf() error
	ReloadServerConf(ctx context.Context) error
	SendTestNotification() error
}

// globalService is the SNMP service of the gateway, nil when it does not run.
func globalService() service {
	if gwsnmp.GlobalService == nil {
		return nil
	}

	return gwsnmp.GlobalService
}

// Handlers serves the pages and the actions of the screen.
type Handlers struct {
	db      *database.DB
	logger  *log.Logger
	shell   *common.Shell
	service func() service
}

func New(db *database.DB, logger *log.Logger) *Handlers {
	return &Handlers{db: db, logger: logger, shell: common.NewShell(), service: globalService}
}

// rights of the current user on SNMP: those of the administration. Reading
// is implied by reaching the screen; writing and deleting are checked again
// by the router on each action.
type rights struct{ Write, Delete bool }

func rightsOf(r *http.Request) rights {
	perms := common.GetUser(r).Permissions

	return rights{
		Write:  perms.HasPermission(model.PermAdminWrite),
		Delete: perms.HasPermission(model.PermAdminDelete),
	}
}

type (
	choice = protoform.Choice
	field  = protoform.Field
)

// isV3 tells whether a stored version is SNMPv3 (the product compares the
// versions regardless of case).
func isV3(version string) bool { return strings.EqualFold(version, versionV3) }

// isV2 tells whether a stored version is SNMPv2c, in either spelling.
func isV2(version string) bool {
	return strings.EqualFold(version, versionV2) || strings.EqualFold(version, versionV2c)
}

// versionLabel names a stored version: SNMPv2c or SNMPv3, the raw value of
// an unknown one.
func versionLabel(version string) string {
	switch {
	case isV2(version):
		return versionV2c
	case isV3(version):
		return versionV3
	default:
		return version
	}
}

// levelOf is the security level of an SNMPv3 monitor, the empty one of the
// product reading as "no authentication".
func levelOf(m *gwsnmp.MonitorConfig) string {
	if m.SNMPv3Security == "" {
		return gwsnmp.V3SecurityNoAuthNoPriv
	}

	return m.SNMPv3Security
}

func usesAuth(level string) bool {
	return level == gwsnmp.V3SecurityAuthNoPriv || level == gwsnmp.V3SecurityAuthPriv
}

func usesPriv(level string) bool { return level == gwsnmp.V3SecurityAuthPriv }

// securityText says how a monitor is secured: by its community in SNMPv2c,
// by its level in SNMPv3.
func securityText(translate gui.Translator, m *gwsnmp.MonitorConfig) string {
	if !isV3(m.Version) {
		return translate("snmp.security.community")
	}

	if slices.Contains(securityLevels, levelOf(m)) {
		return translate(securityKeyPrefix + levelOf(m))
	}

	return m.SNMPv3Security
}

// communityText says what community is set, never its value but the
// default one of the product, which everyone knows.
func communityText(translate gui.Translator, community string) string {
	switch community {
	case "":
		return translate("snmp.detail.unset")
	case defaultCommunity:
		return translate("snmp.detail.community_default")
	default:
		return translate("snmp.detail.set")
	}
}

// protocolLabel names a protocol, saying which ones are historic.
func protocolLabel(translate gui.Translator, protocol string) string {
	if slices.Contains(historicProtocols, protocol) {
		return translate("snmp.form.historic", "name", protocol)
	}

	return protocol
}

type listQuery = liststate.Query

func parseListQuery(v url.Values) *listQuery { return listSpec.Parse(v) }

func ret(r *http.Request) string   { return listSpec.Ret(r) }
func listURL(ret string) string    { return listSpec.ListURL(ret) }
func withRet(u, ret string) string { return liststate.WithRet(u, ret) }

func monitorURL(id int64, suffix string) string {
	return common.WebuiURL(monitorsPath + "/" + strconv.FormatInt(id, 10) + suffix)
}

type row struct {
	ID       int64
	Name     string
	URL      string
	Version  string
	Address  string
	Security string
	Actions  []gui.Action
}

// agentData is the card of the agent: its configuration, never a secret,
// and the state of the SNMP service.
type agentData struct {
	Configured bool
	HasState   bool
	Status     gui.Status
	Reason     string
	Address    string
	Versions   string
	User       string
	Auth       string
	Priv       string
	Community  string
	EditURL    string
	EditLabel  string
	Actions    []gui.Action
}

type pageData struct {
	AddURL      string
	TestURL     string
	Base        string
	Sort        string
	Dir         string
	Rows        []row
	Caption     string
	Pager       *liststate.Pager
	PagerHidden map[string]string
	Sizes       []int
	Size        int
	Empty       bool
	Error       bool
	RetryURL    string
	Agent       agentData
}

// monitors reads the monitors, by name.
func (h *Handlers) monitors() ([]*gwsnmp.MonitorConfig, error) {
	var list model.Slice[*gwsnmp.MonitorConfig]
	if err := h.db.Select(&list).OrderBy("name", true).Run(); err != nil {
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	}

	return list, nil
}

// agent reads the configuration of the agent, nil when there is none.
func (h *Handlers) agent() (*gwsnmp.ServerConfig, error) {
	var conf gwsnmp.ServerConfig

	err := h.db.Get(&conf, "").Run()

	switch {
	case database.IsNotFound(err):
		return nil, nil //nolint:nilnil // nil is "no configuration", as in the neighbouring code
	case err != nil:
		return nil, err //nolint:wrapcheck // logged by the caller with its context
	default:
		return &conf, nil
	}
}

// Page serves GET /webui/snmp: the monitors, then the agent.
func (h *Handlers) Page(w http.ResponseWriter, r *http.Request) {
	if listSpec.Restore(w, r) {
		return
	}

	translate := h.shell.Translator(r)
	query := parseListQuery(r.URL.Query())
	screen := h.screen(translate, translate("snmp.title"), "", 0)
	screen.Crumbs = crumbs(translate, "", "")
	screen.Notices = notices(translate, r.URL.Query())

	list, err := h.monitors()

	var conf *gwsnmp.ServerConfig
	if err == nil {
		conf, err = h.agent()
	}

	if err != nil {
		h.logger.Errorf("failed to read the SNMP configuration: %v", err)

		screen.Status = http.StatusInternalServerError
		h.shell.Render(w, h.logger, r, screen,
			pageView(pageData{Error: true, RetryURL: listSpec.ListURLOf(query.WithPage(query.Page))}))

		return
	}

	data := buildPage(translate, list, query, rightsOf(r))
	data.Agent = h.agentCard(translate, conf, rightsOf(r), query.Encode(query.Page))
	h.shell.Render(w, h.logger, r, screen, pageView(data))
}

// buildPage computes the table of the monitors: sort, page, row actions
// according to the rights.
func buildPage(translate gui.Translator, all []*gwsnmp.MonitorConfig, query *listQuery, user rights) pageData {
	data := pageData{
		Base: listSpec.ListURLOf(query.FilterValues()),
		Sort: query.Sort,
		Dir:  query.Dir,
		Size: query.Size,
	}

	carried := query.Encode(query.Page)
	if user.Write {
		data.AddURL = withRet(common.WebuiURL(monitorsPath+"/new"), carried)
	}

	if len(all) == 0 {
		data.Empty = true

		return data
	}

	if user.Write {
		data.TestURL = withRet(common.WebuiURL(pagePath+"/test"), carried)
	}

	sorted := slices.Clone(all)
	sortMonitors(sorted, query.Sort, query.Dir)

	b := query.Bounds(len(sorted))
	carried = query.Encode(b.Page)
	data.Caption = translate("snmp.caption",
		"page", b.Page, "pages", b.Pages, "sort", translate(sortKeyPrefix+query.Sort))

	for _, m := range sorted[b.From:b.To] {
		data.Rows = append(data.Rows, row{
			ID:       m.ID,
			Name:     m.Name,
			URL:      withRet(monitorURL(m.ID, ""), carried),
			Version:  versionLabel(m.Version),
			Address:  m.UDPAddress,
			Security: securityText(translate, m),
			Actions:  monitorActions(translate, m, user, false, carried),
		})
	}

	data.Sizes = liststate.DefaultSizes
	data.PagerHidden = query.PagerHidden()
	data.Pager = listSpec.Pager(query, translate("snmp.pager", "from", b.From+1, "to", b.To, "total", len(sorted)), b)

	return data
}

// sortMonitors sorts by the asked column; ties keep the name order
// (regardless of case).
func sortMonitors(list []*gwsnmp.MonitorConfig, key, dir string) {
	byName := func(a, b *gwsnmp.MonitorConfig) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	}

	slices.SortStableFunc(list, func(left, right *gwsnmp.MonitorConfig) int {
		order := byName(left, right)
		if key == sortAddress {
			order = strings.Compare(left.UDPAddress, right.UDPAddress)
		}

		if dir == dirDesc {
			order = -order
		}

		return cmp.Or(order, byName(left, right))
	})
}

// monitorActions: the menu of a row, or of the sheet (no "view" there).
func monitorActions(translate gui.Translator, m *gwsnmp.MonitorConfig, user rights, onSheet bool, carried string,
) []gui.Action {
	var acts []gui.Action

	if !onSheet {
		acts = append(acts, gui.Action{
			Icon: "eye", Label: translate("snmp.action.detail"), URL: withRet(monitorURL(m.ID, ""), carried),
		})

		if user.Write {
			acts = append(acts, gui.Action{
				Icon: "pen", Label: translate("snmp.action.edit"), URL: withRet(monitorURL(m.ID, "/edit"), carried),
			})
		}
	}

	if user.Delete {
		acts = append(acts, gui.Action{
			Icon:   "trash",
			Label:  translate("snmp.action.delete"),
			URL:    withRet(monitorURL(m.ID, "/delete"), carried),
			Danger: true,
		})
	}

	return acts
}

// agentCard builds the card of the agent: what is configured (never a
// secret), the state of the SNMP service, the actions the rights allow.
func (h *Handlers) agentCard(translate gui.Translator, conf *gwsnmp.ServerConfig, user rights, carried string,
) agentData {
	card := agentData{EditLabel: translate("snmp.agent.action.configure")}

	if svc := h.service(); svc != nil {
		code, reason := svc.State()
		card.HasState, card.Status, card.Reason = true, services.Status(translate, code.String()), reason
	}

	if user.Write {
		card.EditURL = withRet(common.WebuiURL(agentPath+"/edit"), carried)
	}

	if conf == nil {
		return card
	}

	card.Configured = true
	card.EditLabel = translate("snmp.agent.action.edit")
	card.Address = conf.LocalUDPAddress
	card.Versions = translate("snmp.agent.versions.all")

	if conf.SNMPv3Only {
		card.Versions = translate("snmp.agent.versions.v3")
	}

	card.User = conf.SNMPv3Username
	card.Auth = protocolText(translate, conf.SNMPv3AuthProtocol, conf.SNMPv3AuthPassphrase != "")
	card.Priv = protocolText(translate, conf.SNMPv3PrivProtocol, conf.SNMPv3PrivPassphrase != "")
	card.Community = communityText(translate, conf.Community)

	if user.Write {
		card.Actions = []gui.Action{{
			Icon:   "trash",
			Label:  translate("snmp.action.disable"),
			URL:    withRet(common.WebuiURL(agentPath+"/delete"), carried),
			Danger: true,
		}}
	}

	return card
}

// protocolText says a protocol and whether its passphrase is set, never the
// passphrase; "none" without a protocol.
func protocolText(translate gui.Translator, protocol string, set bool) string {
	switch {
	case protocol == "":
		return translate("snmp.detail.none")
	case set:
		return translate("snmp.detail.protocol_set", "name", protocolLabel(translate, protocol))
	default:
		return translate("snmp.detail.protocol_unset", "name", protocolLabel(translate, protocol))
	}
}

// crumbs is the breadcrumb: the screen alone, or the screen (as the user
// left it) then the page.
func crumbs(translate gui.Translator, last, carried string) []gui.Crumb {
	c := []gui.Crumb{
		{Label: translate("nav.home"), URL: common.WebuiURL("/home")},
		{Label: translate("nav.administration")},
	}

	if last == "" {
		return append(c, gui.Crumb{Label: translate("snmp.title")})
	}

	return append(c,
		gui.Crumb{Label: translate("snmp.title"), URL: listURL(carried)},
		gui.Crumb{Label: last})
}

// notices turns the result of the previous action into a notification: a
// success, or a failure with its cause (the test, the reload of the
// service).
func notices(translate gui.Translator, values url.Values) []gui.Notice {
	if done := values.Get(doneParam); slices.Contains(doneKinds, done) {
		return []gui.Notice{{Kind: stateOK, Title: translate(toastKeyPrefix+done, "name", values.Get(nameParam))}}
	}

	switch values.Get(failedParam) {
	case failedTest:
		return []gui.Notice{{Kind: stateKO, Title: translate("snmp.toast.test_failed"), Text: values.Get(reasonParam)}}
	case failedReload:
		return []gui.Notice{{Kind: stateKO, Title: translate("snmp.toast.reload_failed"), Text: values.Get(reasonParam)}}
	default:
		return nil
	}
}

// screen is a page of the screen: the chrome around the data.
func (h *Handlers) screen(translate gui.Translator, title, carried string, status int) common.Screen {
	return common.Screen{
		Title:       title,
		Section:     constants.SidebarSectionAdministration,
		Menu:        constants.SidebarMenuSNMP,
		Doc:         docPage,
		Crumbs:      crumbs(translate, title, carried),
		Status:      status,
		Stylesheets: []string{stylesheet},
	}
}

// reloaded redirects to the screen once a configuration is saved: the
// success, or the failure of the SNMP service to take it (the configuration
// stays saved, as with the REST API).
func (h *Handlers) reloaded(w http.ResponseWriter, r *http.Request, target, done, name string, err error) {
	result := url.Values{doneParam: {done}, nameParam: {name}}

	if err != nil {
		h.logger.Errorf("failed to reload the SNMP service: %v", err)

		result = url.Values{failedParam: {failedReload}, reasonParam: {err.Error()}}
	}

	liststate.RedirectTo(w, r, target, result)
}

// reloadMonitors makes the SNMP service read the monitors again, as the
// REST API does after each change.
func (h *Handlers) reloadMonitors() error {
	if svc := h.service(); svc != nil {
		return svc.ReloadMonitorsConf() //nolint:wrapcheck // shown as is in the notification
	}

	return nil
}

// reloadAgent restarts the SNMP service on the new configuration of the
// agent, as the REST API does.
func (h *Handlers) reloadAgent(ctx context.Context) error {
	if svc := h.service(); svc != nil {
		return svc.ReloadServerConf(ctx) //nolint:wrapcheck // shown as is in the notification
	}

	return nil
}

// -------------------------------------------------------------- pages ----

type notFoundData struct {
	Title     string
	Text      string
	BackLabel string
	BackURL   string
}

// monitor reads the {id} of the route; a missing monitor renders the
// localised "not found" page and returns false.
func (h *Handlers) monitor(w http.ResponseWriter, r *http.Request) (*gwsnmp.MonitorConfig, bool) {
	id, parseErr := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

	var m gwsnmp.MonitorConfig

	err := h.db.Get(&m, "id=?", id).Run()
	if parseErr == nil && err == nil {
		return &m, true
	}

	if err != nil && !database.IsNotFound(err) {
		h.logger.Errorf("failed to retrieve the SNMP monitor %d: %v", id, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return nil, false
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("snmp.not_found.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, http.StatusNotFound), notFoundView(notFoundData{
		Title:     title,
		Text:      translate("snmp.not_found.text"),
		BackLabel: translate("snmp.back"),
		BackURL:   listURL(carried),
	}))

	return nil, false
}

type monitorData struct {
	Name          string
	Version       string
	Address       string
	V3            bool
	Community     string
	Security      string
	User          string
	EngineID      string
	Auth          string
	Priv          string
	ContextName   string
	ContextEngine string
	ShowAuth      bool
	ShowPriv      bool
	BackURL       string
	EditURL       string
	Actions       []gui.Action
}

// Monitor serves GET /webui/snmp/monitors/{id}: the monitor in reading,
// never a secret.
func (h *Handlers) Monitor(w http.ResponseWriter, r *http.Request) {
	item, ok := h.monitor(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	user := rightsOf(r)
	level := levelOf(item)

	data := monitorData{
		Name:          item.Name,
		Version:       versionLabel(item.Version),
		Address:       item.UDPAddress,
		V3:            isV3(item.Version),
		Community:     communityText(translate, item.Community),
		Security:      securityText(translate, item),
		User:          item.AuthUsername,
		EngineID:      item.AuthEngineID,
		Auth:          protocolText(translate, item.AuthProtocol, item.AuthPassphrase != ""),
		Priv:          protocolText(translate, item.PrivProtocol, item.PrivPassphrase != ""),
		ContextName:   item.ContextName,
		ContextEngine: item.ContextEngineID,
		ShowAuth:      usesAuth(level),
		ShowPriv:      usesPriv(level),
		BackURL:       listURL(carried),
		Actions:       monitorActions(translate, item, user, true, carried),
	}

	if user.Write {
		data.EditURL = withRet(monitorURL(item.ID, "/edit"), carried)
	}

	title := translate("snmp.detail.title", "name", item.Name)
	screen := h.screen(translate, title, carried, 0)
	screen.Notices = notices(translate, r.URL.Query())
	h.shell.Render(w, h.logger, r, screen, monitorView(data))
}

type deleteData struct {
	Title     string
	Text      string
	Name      string
	Version   string
	Address   string
	Agent     bool
	Label     string // the verb of the button, when it is not "Delete"
	BackLabel string
	CancelURL string
	Action    string
}

// DeleteMonitorForm serves GET /webui/snmp/monitors/{id}/delete: the
// confirmation page, with the consequence.
func (h *Handlers) DeleteMonitorForm(w http.ResponseWriter, r *http.Request) {
	m, ok := h.monitor(w, r)
	if !ok {
		return
	}

	translate := h.shell.Translator(r)
	carried := ret(r)
	title := translate("snmp.delete.title", "name", m.Name)

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), deleteView(deleteData{
		Title:     title,
		Text:      translate("snmp.delete.text"),
		Name:      m.Name,
		Version:   versionLabel(m.Version),
		Address:   m.UDPAddress,
		BackLabel: translate("snmp.back_monitor"),
		CancelURL: withRet(monitorURL(m.ID, ""), carried),
		Action:    withRet(monitorURL(m.ID, "/delete"), carried),
	}))
}

// DeleteMonitor serves POST /webui/snmp/monitors/{id}/delete: the monitor
// no longer receives the notifications; the SNMP service reads the monitors
// again.
func (h *Handlers) DeleteMonitor(w http.ResponseWriter, r *http.Request) {
	m, ok := h.monitor(w, r)
	if !ok {
		return
	}

	if err := h.db.Delete(m).Run(); err != nil {
		h.logger.Errorf("failed to delete the SNMP monitor %q: %v", m.Name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	h.reloaded(w, r, listURL(ret(r)), doneDeleted, m.Name, h.reloadMonitors())
}

// DisableAgentForm serves GET /webui/snmp/agent/delete: the confirmation
// page of the disabling of the agent (its configuration is deleted, as with
// the REST API); without a configuration, back to the screen.
func (h *Handlers) DisableAgentForm(w http.ResponseWriter, r *http.Request) {
	conf, err := h.agent()
	if err != nil {
		h.logger.Errorf("failed to read the SNMP agent configuration: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	carried := ret(r)
	if conf == nil {
		http.Redirect(w, r, listURL(carried), http.StatusSeeOther)

		return
	}

	translate := h.shell.Translator(r)
	title := translate("snmp.agent.delete.title")

	h.shell.Render(w, h.logger, r, h.screen(translate, title, carried, 0), deleteView(deleteData{
		Title:     title,
		Text:      translate("snmp.agent.delete.text"),
		Address:   conf.LocalUDPAddress,
		Agent:     true,
		Label:     translate("snmp.agent.delete.action"),
		BackLabel: translate("snmp.back"),
		CancelURL: listURL(carried),
		Action:    withRet(common.WebuiURL(agentPath+"/delete"), carried),
	}))
}

// DisableAgent serves POST /webui/snmp/agent/delete: the gateway no longer
// answers SNMP requests; the notifications go on. The SNMP service restarts
// without the agent, as with the REST API.
func (h *Handlers) DisableAgent(w http.ResponseWriter, r *http.Request) {
	conf, err := h.agent()
	if err == nil && conf != nil {
		err = h.db.Delete(conf).Run()
	}

	if err != nil {
		h.logger.Errorf("failed to disable the SNMP agent: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)

		return
	}

	carried := ret(r)
	if conf == nil {
		http.Redirect(w, r, listURL(carried), http.StatusSeeOther)

		return
	}

	h.reloaded(w, r, listURL(carried), doneAgentDisabled, "", h.reloadAgent(r.Context()))
}

// Test serves POST /webui/snmp/test: the SNMP service sends the test
// notification (a transfer error) to every monitor; the screen says the
// result or the cause of the failure.
func (h *Handlers) Test(w http.ResponseWriter, r *http.Request) {
	result := url.Values{doneParam: {doneTested}}

	svc := h.service()
	if svc == nil {
		result = url.Values{failedParam: {failedTest}, reasonParam: {h.shell.Translator(r)("snmp.test_not_running")}}
	} else if err := svc.SendTestNotification(); err != nil {
		h.logger.Warningf("failed to send the SNMP test notification: %v", err)

		result = url.Values{failedParam: {failedTest}, reasonParam: {err.Error()}}
	}

	liststate.RedirectTo(w, r, listURL(ret(r)), result)
}
