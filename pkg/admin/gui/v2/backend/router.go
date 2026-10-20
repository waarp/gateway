package backend

import (
	"net/http"

	"code.waarp.fr/lib/gui"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rules/tasks"
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
