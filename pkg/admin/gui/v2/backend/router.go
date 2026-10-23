package backend

import (
	"net/http"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/authorities"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/clients"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/clouds"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/emailtemplates"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/home"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/keys"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/me"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/overrides"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/partners"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rules"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/servers"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/smtp"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/transfers"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/users"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// permAuthenticated is the permission mask of a page any authenticated user
// may read, whatever their rights.
const permAuthenticated model.PermsMask = 0

func MakeRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	router.PathPrefix(constants.StaticPrefix).Handler(
		http.StripPrefix(
			constants.WebuiPrefix+constants.StaticPrefix,
			http.FileServerFS(webfs.WebFS),
		),
	)

	MakeHomeRouter(router, db, logger)
	MakeServicesRouter(router, logger)
	MakeServersRouter(router, db, logger)
	MakeClientsRouter(router, db, logger)
	MakePartnersRouter(router, db, logger)
	MakeUsersRouter(router, db, logger)
	MakeCloudsRouter(router, db, logger)
	MakeAuthoritiesRouter(router, db, logger)
	MakeSNMPRouter(router, db, logger)
	MakeOverridesRouter(router, db, logger)
	MakeRulesRouter(router, db, logger)
	MakeKeysRouter(router, db, logger)
	MakeSMTPCredentialsRouter(router, db, logger)
	MakeEmailTemplatesRouter(router, db, logger)
	MakeTransfersRouter(router, db, logger)
}

// adminRoutes registers the routes of a screen behind the rights on the
// administration: the pages in reading, the forms behind the write right,
// the deletion behind the delete right. Every action is a POST.
type adminRoutes struct{ router *mux.Router }

func (a adminRoutes) read(path string, handler http.HandlerFunc) {
	a.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermAdminRead, handler))
}

func (a adminRoutes) write(path string, form, action http.HandlerFunc) {
	a.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermAdminWrite, form))
	a.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermAdminWrite, action))
}

func (a adminRoutes) del(path string, form, action http.HandlerFunc) {
	a.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermAdminDelete, form))
	a.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermAdminDelete, action))
}

// act registers an action without a page of its own (a button of a page),
// behind the write right.
func (a adminRoutes) act(path string, action http.HandlerFunc) {
	a.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermAdminWrite, action))
}

// MakeCloudsRouter registers the "Cloud instances" screen: the list and the
// sheet in reading, the creation, the edition, the deletion.
func MakeCloudsRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := clouds.New(db, logger)
	routes := adminRoutes{router}
	const cloudPath = "/clouds/{id:[0-9]+}"

	routes.read("/clouds", handlers.List)
	routes.write("/clouds/new", handlers.NewForm, handlers.Create)
	routes.read(cloudPath, handlers.Detail)
	routes.write(cloudPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(cloudPath+"/delete", handlers.DeleteForm, handlers.Delete)
}

// userRoutes registers the routes of the users screen behind the rights on
// the users: the pages in reading, the forms behind the write right, the
// deletion behind the delete right. Every action is a POST.
type userRoutes struct{ router *mux.Router }

func (u userRoutes) read(path string, handler http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermUsersRead, handler))
}

func (u userRoutes) write(path string, form, action http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermUsersWrite, form))
	u.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermUsersWrite, action))
}

func (u userRoutes) del(path string, form, action http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermUsersDelete, form))
	u.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermUsersDelete, action))
}

// MakeUsersRouter registers the "Users" screen: the list and the sheet in
// reading, the creation, the edition, the deletion.
func MakeUsersRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := users.New(db, logger)
	routes := userRoutes{router}
	const userPath = "/users/{id:[0-9]+}"

	routes.read("/users", handlers.List)
	routes.write("/users/new", handlers.NewForm, handlers.Create)
	routes.read(userPath, handlers.Detail)
	routes.write(userPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(userPath+"/delete", handlers.DeleteForm, handlers.Delete)
}

// MakeHomeRouter registers the home page, the "About" page and the "My
// account" page: any authenticated user; GET only, except the change of the
// user's own password, a POST.
func MakeHomeRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	router.Path("/home").Methods(http.MethodGet).Handler(withPermissions(permAuthenticated, home.Page(logger)))
	router.Path("/about").Methods(http.MethodGet).Handler(withPermissions(permAuthenticated, home.About(logger)))

	account := me.New(db, logger)
	router.Path("/account").Methods(http.MethodGet).Handler(
		withPermissions(permAuthenticated, http.HandlerFunc(account.Page)),
	)
	router.Path("/account/password").Methods(http.MethodPost).Handler(
		withPermissions(permAuthenticated, http.HandlerFunc(account.ChangePassword)),
	)
}

// MakeKitStaticRouter serves the files of the GUI kit (styles, scripts,
// fonts, icons). Like the other static files, they need no authentication.
func MakeKitStaticRouter(router *mux.Router) {
	router.PathPrefix(constants.KitStaticPrefix).Handler(
		http.StripPrefix(
			constants.WebuiPrefix+constants.KitStaticPrefix,
			http.FileServerFS(gui.Static),
		),
	)
}

// MakeServicesRouter registers the "Service status" screen. As the historic
// page it replaces, it requires no particular permission, only an
// authenticated user; it serves GET only.
func MakeServicesRouter(router *mux.Router, logger *log.Logger) {
	router.Path("/services").Methods(http.MethodGet).Handler(
		withPermissions(permAuthenticated, services.Page(logger, internal.ListServices)),
	)
}

// serverRoutes registers the routes of a screen behind the rights on the
// servers: the pages in reading behind the read right, the forms (a GET page
// and its POST action) behind the write right, the deletions behind the
// delete right, the bare actions (start, stop) behind the write right. Every
// action is a POST (Post/Redirect/Get). The local clients follow the rights
// on the servers too: the product has no right of their own.
type serverRoutes struct{ router *mux.Router }

func (s serverRoutes) read(path string, handler http.HandlerFunc) {
	s.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermServersRead, handler))
}

func (s serverRoutes) write(path string, form, action http.HandlerFunc) {
	s.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermServersWrite, form))
	s.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermServersWrite, action))
}

func (s serverRoutes) del(path string, form, action http.HandlerFunc) {
	s.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermServersDelete, form))
	s.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermServersDelete, action))
}

func (s serverRoutes) action(path string, handler http.HandlerFunc) {
	s.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermServersWrite, handler))
}

// partnerRoutes registers the routes of the remote partners screen behind
// the rights on the partners: the pages in reading, the forms behind the
// write right, the deletions (of a partner, of an account, of a means)
// behind the delete right.
type partnerRoutes struct{ router *mux.Router }

func (p partnerRoutes) read(path string, handler http.HandlerFunc) {
	p.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermPartnersRead, handler))
}

func (p partnerRoutes) write(path string, form, action http.HandlerFunc) {
	p.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermPartnersWrite, form))
	p.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermPartnersWrite, action))
}

func (p partnerRoutes) del(path string, form, action http.HandlerFunc) {
	p.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermPartnersDelete, form))
	p.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermPartnersDelete, action))
}

// MakePartnersRouter registers the "Remote partners" screen, the sheet of a
// partner and the sheets of its remote accounts: the lists, the details and
// the tabs in reading, the
// creation, the edition, the deletions, the pages of the means.
func MakePartnersRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := partners.New(db, logger)
	routes := partnerRoutes{router}
	const partnerPath = "/partners/{id:[0-9]+}"

	routes.read("/partners", handlers.List)
	routes.write("/partners/new", handlers.NewForm, handlers.Create)
	routes.read(partnerPath, handlers.Detail)
	routes.write(partnerPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(partnerPath+"/delete", handlers.DeleteForm, handlers.Delete)

	const accountPath = partnerPath + "/accounts/{aid:[0-9]+}"

	// The means the gateway checks of the partner, then the means it
	// presents on one of its accounts: the same pages.
	for _, base := range []string{partnerPath + "/auth", accountPath + "/auth"} {
		routes.read(base, handlers.Auth)
		routes.write(base+"/new", handlers.NewCredForm, handlers.CreateCred)
		routes.read(base+"/{cid:[0-9]+}", handlers.CredDetail)
		routes.write(base+"/{cid:[0-9]+}/edit", handlers.EditCredForm, handlers.UpdateCred)
		routes.del(base+"/{cid:[0-9]+}/delete", handlers.DeleteCredForm, handlers.DeleteCred)
	}

	routes.read(partnerPath+"/accounts", handlers.Accounts)
	routes.write(partnerPath+"/accounts/new", handlers.NewAccountForm, handlers.CreateAccount)
	routes.read(accountPath, handlers.AccountDetail)
	routes.write(accountPath+"/edit", handlers.EditAccountForm, handlers.UpdateAccount)
	routes.del(accountPath+"/delete", handlers.DeleteAccountForm, handlers.DeleteAccount)
}

// MakeServersRouter registers the "Local servers" screen and the sheet of a
// server: the list, the details, the tabs and the pages in reading, the
// creation, the edition, the start and the stop, the deletions.
func MakeServersRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := servers.New(db, logger, servers.GatewayControl())
	routes := serverRoutes{router}
	const (
		serverPath  = "/servers/{id:[0-9]+}"
		accountPath = serverPath + "/accounts/{aid:[0-9]+}"
	)

	routes.read("/servers", handlers.List)
	routes.write("/servers/new", handlers.NewForm, handlers.Create)
	routes.read(serverPath, handlers.Detail)
	routes.write(serverPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(serverPath+"/delete", handlers.DeleteForm, handlers.Delete)
	routes.action(serverPath+"/start", handlers.Start)
	routes.action(serverPath+"/stop", handlers.Stop)

	// The means of the server, then of one of its accounts: the same pages.
	for _, base := range []string{serverPath + "/auth", accountPath + "/auth"} {
		routes.read(base, handlers.Auth)
		routes.write(base+"/new", handlers.NewCredForm, handlers.CreateCred)
		routes.read(base+"/{cid:[0-9]+}", handlers.CredDetail)
		routes.write(base+"/{cid:[0-9]+}/edit", handlers.EditCredForm, handlers.UpdateCred)
		routes.del(base+"/{cid:[0-9]+}/delete", handlers.DeleteCredForm, handlers.DeleteCred)
	}

	routes.read(serverPath+"/accounts", handlers.Accounts)
	routes.write(serverPath+"/accounts/new", handlers.NewAccountForm, handlers.CreateAccount)
	routes.read(accountPath, handlers.AccountDetail)
	routes.write(accountPath+"/edit", handlers.EditAccountForm, handlers.UpdateAccount)
	routes.del(accountPath+"/delete", handlers.DeleteAccountForm, handlers.DeleteAccount)
}

// MakeClientsRouter registers the "Local clients" screen: the list and the
// details in reading, the creation, the edition, the start and the stop,
// the deletion.
func MakeClientsRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := clients.New(db, logger, clients.GatewayControl())
	routes := serverRoutes{router}
	const clientPath = "/clients/{id:[0-9]+}"

	routes.read("/clients", handlers.List)
	routes.write("/clients/new", handlers.NewForm, handlers.Create)
	routes.read(clientPath, handlers.Detail)
	routes.write(clientPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(clientPath+"/delete", handlers.DeleteForm, handlers.Delete)
	routes.action(clientPath+"/start", handlers.Start)
	routes.action(clientPath+"/stop", handlers.Stop)
}

// MakeAuthoritiesRouter registers the "Authentication authorities" screen:
// the list and the sheet in reading, the creation, the edition, the deletion.
func MakeAuthoritiesRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := authorities.New(db, logger)
	routes := adminRoutes{router}
	const authorityPath = "/authorities/{id:[0-9]+}"

	routes.read("/authorities", handlers.List)
	routes.write("/authorities/new", handlers.NewForm, handlers.Create)
	routes.read(authorityPath, handlers.Detail)
	routes.write(authorityPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(authorityPath+"/delete", handlers.DeleteForm, handlers.Delete)
}

// MakeSNMPRouter registers the "SNMP" screen: the page of the monitors and
// of the agent, the sheet of a monitor, its creation, edition and deletion,
// the configuration of the agent and its deletion (the write right, as the
// REST API asks), the test notification.
func MakeSNMPRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := snmp.New(db, logger)
	routes := adminRoutes{router}
	const monitorPath = "/snmp/monitors/{id:[0-9]+}"

	routes.read("/snmp", handlers.Page)
	routes.write("/snmp/monitors/new", handlers.NewMonitorForm, handlers.CreateMonitor)
	routes.read(monitorPath, handlers.Monitor)
	routes.write(monitorPath+"/edit", handlers.EditMonitorForm, handlers.UpdateMonitor)
	routes.del(monitorPath+"/delete", handlers.DeleteMonitorForm, handlers.DeleteMonitor)
	routes.write("/snmp/agent/edit", handlers.AgentForm, handlers.SaveAgent)
	routes.write("/snmp/agent/delete", handlers.DisableAgentForm, handlers.DisableAgent)
	routes.act("/snmp/test", handlers.Test)
}

// MakeOverridesRouter registers the "Configuration overrides" screen: the
// list of the addresses replaced on the node, the pages that add, edit and
// delete a rule (named by its configured address, in a parameter).
func MakeOverridesRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := overrides.New(db, logger)
	routes := adminRoutes{router}

	routes.read("/overrides", handlers.List)
	routes.write("/overrides/new", handlers.NewForm, handlers.Create)
	routes.write("/overrides/edit", handlers.EditForm, handlers.Update)
	routes.del("/overrides/delete", handlers.DeleteForm, handlers.Delete)
}

// MakeRulesRouter registers the "Transfer rules" screen, behind the rights on
// the rules as the REST API: the list and the sheet in reading, the creation,
// the edition, the deletion; the Tasks tab in reading, the addition, the
// edition, the removal and the moves of a task behind the write right; the
// Access tab in reading, the addition and the removal of an access and the
// opening to all behind the write right. The address of the former tasks
// page leads to the Tasks tab.
func MakeRulesRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := rules.New(db, logger)
	routes := ruleRoutes{router}
	const (
		rulePath = "/rules/{id:[0-9]+}"
		taskPath = rulePath + "/tasks/{chain}/{rank:[0-9]+}"
	)

	routes.read("/rules", handlers.List)
	routes.write("/rules/new", handlers.NewForm, handlers.Create)
	routes.read(rulePath, handlers.Detail)
	routes.write(rulePath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(rulePath+"/delete", handlers.DeleteForm, handlers.Delete)
	routes.read("/tasks", rules.FormerTasksPage)
	routes.read(rulePath+"/tasks", handlers.TasksTab)
	routes.write(rulePath+"/tasks/{chain}/new", handlers.TaskNewForm, handlers.TaskCreate)
	routes.write(taskPath+"/edit", handlers.TaskEditForm, handlers.TaskUpdate)
	routes.write(taskPath+"/delete", handlers.TaskDeleteForm, handlers.TaskDelete)
	routes.act(taskPath+"/up", handlers.TaskUp)
	routes.act(taskPath+"/down", handlers.TaskDown)
	routes.read(rulePath+"/access", handlers.AccessTab)
	routes.write(rulePath+"/access/new", handlers.AccessNewForm, handlers.AccessCreate)
	routes.write(rulePath+"/access/open", handlers.AccessOpenForm, handlers.AccessOpen)
	routes.write(rulePath+"/access/{kind}/{target:[0-9]+}/delete", handlers.AccessDeleteForm, handlers.AccessDelete)
}

// MakeKeysRouter registers the "Cryptographic keys" screen: the list and the
// sheet in reading, the creation, the edition, the deletion.
func MakeKeysRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := keys.New(db, logger)
	routes := adminRoutes{router}
	const keyPath = "/keys/{id:[0-9]+}"

	routes.read("/keys", handlers.List)
	routes.write("/keys/new", handlers.NewForm, handlers.Create)
	routes.read(keyPath, handlers.Detail)
	routes.write(keyPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(keyPath+"/delete", handlers.DeleteForm, handlers.Delete)
}

// ruleRoutes registers the routes of a screen behind the rights on the
// rules: the pages in reading, the forms and the actions without a page
// behind the write right, the deletion behind the delete right. Every action
// is a POST.
type ruleRoutes struct{ router *mux.Router }

func (u ruleRoutes) read(path string, handler http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermRulesRead, handler))
}

func (u ruleRoutes) write(path string, form, action http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermRulesWrite, form))
	u.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermRulesWrite, action))
}

func (u ruleRoutes) del(path string, form, action http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermRulesDelete, form))
	u.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermRulesDelete, action))
}

func (u ruleRoutes) act(path string, action http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermRulesWrite, action))
}

// MakeSMTPCredentialsRouter registers the "SMTP credentials" screen, behind
// the rights on the rules as the REST API of the credentials: the list and
// the sheet in reading, the creation, the edition, the deletion.
func MakeSMTPCredentialsRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := smtp.New(db, logger)
	routes := ruleRoutes{router}
	const credPath = "/smtp/{id:[0-9]+}"

	routes.read("/smtp", handlers.List)
	routes.write("/smtp/new", handlers.NewForm, handlers.Create)
	routes.read(credPath, handlers.Detail)
	routes.write(credPath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(credPath+"/delete", handlers.DeleteForm, handlers.Delete)
}

// MakeEmailTemplatesRouter registers the "Email templates" screen, behind the
// rights on the rules as the REST API of the templates: the list and the
// sheet in reading, the preview of the template a form holds (a POST that
// writes nothing), the creation, the edition, the deletion.
func MakeEmailTemplatesRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := emailtemplates.New(db, logger)
	routes := ruleRoutes{router}
	const templatePath = "/email-templates/{id:[0-9]+}"

	routes.read("/email-templates", handlers.List)
	routes.write("/email-templates/new", handlers.NewForm, handlers.Create)
	router.Path("/email-templates/preview").Methods(http.MethodPost).
		Handler(withPermissions(model.PermRulesRead, http.HandlerFunc(handlers.Preview)))
	routes.read(templatePath, handlers.Detail)
	routes.write(templatePath+"/edit", handlers.EditForm, handlers.Update)
	routes.del(templatePath+"/delete", handlers.DeleteForm, handlers.Delete)
}

// transferRoutes registers the routes of the transfer monitoring screen
// behind the rights on the transfers: the pages in reading, every action and
// its page behind the write right. Every action is a POST.
type transferRoutes struct{ router *mux.Router }

func (u transferRoutes) read(path string, handler http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermTransfersRead, handler))
}

func (u transferRoutes) write(path string, form, action http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermTransfersWrite, form))
	u.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermTransfersWrite, action))
}

// act registers an action without a page of its own (a button of a page),
// behind the write right.
func (u transferRoutes) act(path string, action http.HandlerFunc) {
	u.router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermTransfersWrite, action))
}

// MakeTransfersRouter registers the "Transfer monitoring" screen: the list
// and the sheet in reading; the addition of a client transfer, the
// preregistration of a server transfer, the pause, the resumption, the
// cancellation and the retry behind the write right.
func MakeTransfersRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := transfers.New(db, logger)
	routes := transferRoutes{router}
	const transferPath = "/transfers/{id:[0-9]+}"

	routes.read("/transfers", handlers.List)
	routes.write("/transfers/new", handlers.NewForm, handlers.Create)
	routes.write("/transfers/preregister", handlers.PreregisterForm, handlers.Preregister)
	routes.read(transferPath, handlers.Detail)
	routes.act(transferPath+"/pause", handlers.Pause)
	routes.act(transferPath+"/resume", handlers.Resume)
	routes.write(transferPath+"/cancel", handlers.CancelForm, handlers.Cancel)
	routes.write(transferPath+"/retry", handlers.RetryForm, handlers.Retry)
}
