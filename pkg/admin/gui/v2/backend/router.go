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
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/home"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/me"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/partners"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rules/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/servers"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/snmp"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/users"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/listing"
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

	MakeListingRouter(router, db, logger)
	MakeTasksRouter(router, db, logger)
	MakeHomeRouter(router, db, logger)
	MakeServicesRouter(router, logger)
	MakeServersRouter(router, db, logger)
	MakeClientsRouter(router, db, logger)
	MakePartnersRouter(router, db, logger)
	MakeUsersRouter(router, db, logger)
	MakeCloudsRouter(router, db, logger)
	MakeAuthoritiesRouter(router, db, logger)
	MakeSNMPRouter(router, db, logger)
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

func MakeListingRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	router.Path("/listing/rules").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, listing.ListRules(db, logger)),
	)

	router.Path("/listing/clients").Methods(http.MethodGet).Handler(
		withPermissions(model.PermServersRead, listing.ListClients(db, logger)),
	)

	router.Path("/listing/servers").Methods(http.MethodGet).Handler(
		withPermissions(model.PermServersRead, listing.ListServers(db, logger)),
	)

	router.Path("/listing/partners").Methods(http.MethodGet).Handler(
		withPermissions(model.PermPartnersRead, listing.ListPartners(db, logger)),
	)

	router.Path("/listing/local_accounts").Methods(http.MethodGet).Handler(
		withPermissions(model.PermServersRead, listing.ListLocalAccounts(db, logger)),
	)

	router.Path("/listing/remote_accounts").Methods(http.MethodGet).Handler(
		withPermissions(model.PermPartnersRead, listing.ListRemoteAccounts(db, logger)),
	)

	router.Path("/listing/keys").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, listing.ListCryptoKeys(db, logger)),
	)

	router.Path("/listing/email_templates").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, listing.ListEmailTemplates(db, logger)),
	)

	router.Path("/listing/smtp_credentials").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, listing.ListSMTPCredentials(db, logger)),
	)
}

func MakeTasksRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	router.Path("/tasks").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, tasks.GetTasksPage(db, logger)),
	)
	router.Path("/tasks").Methods(http.MethodPost).Handler(
		withPermissions(model.PermRulesWrite, tasks.PostTask(db, logger)),
	)
	router.Path("/tasks").Methods(http.MethodPut).Handler(
		withPermissions(model.PermRulesWrite, tasks.ReorderTasks(db, logger)),
	)
	router.Path("/tasks").Methods(http.MethodDelete).Handler(
		withPermissions(model.PermRulesDelete, tasks.DeleteTask(db, logger)),
	)

	router.Path("/tasks/modal").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, tasks.GetNewTaskModal(db, logger)),
	)

	router.Path("/tasks/forms").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, tasks.GetNewTaskForm(db, logger)),
	)
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
