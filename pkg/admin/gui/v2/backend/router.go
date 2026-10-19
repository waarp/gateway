package backend

import (
	"net/http"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rules/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/servers"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/services"
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
	MakeServicesRouter(router, logger)
	MakeServersRouter(router, db, logger)
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

// MakeServersRouter registers the "Local servers" screen and the sheet of a
// server: the list, the details, the tabs and the pages in reading behind
// the read right, the creation, the edition, the start and the stop behind
// the write right, the deletions behind the delete right. Every action is a
// POST (Post/Redirect/Get).
func MakeServersRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	handlers := servers.New(db, logger, servers.GatewayControl())
	const (
		serverPath  = "/servers/{id:[0-9]+}"
		accountPath = serverPath + "/accounts/{aid:[0-9]+}"
	)

	read := func(path string, handler http.HandlerFunc) {
		router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermServersRead, handler))
	}
	write := func(path string, form, action http.HandlerFunc) {
		router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermServersWrite, form))
		router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermServersWrite, action))
	}
	del := func(path string, form, action http.HandlerFunc) {
		router.Path(path).Methods(http.MethodGet).Handler(withPermissions(model.PermServersDelete, form))
		router.Path(path).Methods(http.MethodPost).Handler(withPermissions(model.PermServersDelete, action))
	}

	read("/servers", handlers.List)
	write("/servers/new", handlers.NewForm, handlers.Create)
	read(serverPath, handlers.Detail)
	write(serverPath+"/edit", handlers.EditForm, handlers.Update)
	del(serverPath+"/delete", handlers.DeleteForm, handlers.Delete)
	router.Path(serverPath + "/start").Methods(http.MethodPost).Handler(
		withPermissions(model.PermServersWrite, http.HandlerFunc(handlers.Start)),
	)
	router.Path(serverPath + "/stop").Methods(http.MethodPost).Handler(
		withPermissions(model.PermServersWrite, http.HandlerFunc(handlers.Stop)),
	)

	// The means of the server, then of one of its accounts: the same pages.
	for _, base := range []string{serverPath + "/auth", accountPath + "/auth"} {
		read(base, handlers.Auth)
		write(base+"/new", handlers.NewCredForm, handlers.CreateCred)
		read(base+"/{cid:[0-9]+}", handlers.CredDetail)
		write(base+"/{cid:[0-9]+}/edit", handlers.EditCredForm, handlers.UpdateCred)
		del(base+"/{cid:[0-9]+}/delete", handlers.DeleteCredForm, handlers.DeleteCred)
	}

	read(serverPath+"/accounts", handlers.Accounts)
	write(serverPath+"/accounts/new", handlers.NewAccountForm, handlers.CreateAccount)
	read(accountPath, handlers.AccountDetail)
	write(accountPath+"/edit", handlers.EditAccountForm, handlers.UpdateAccount)
	del(accountPath+"/delete", handlers.DeleteAccountForm, handlers.DeleteAccount)
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
