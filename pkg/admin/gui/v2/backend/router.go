package backend

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rules/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/listing"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/webfs"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func MakeRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	router.PathPrefix(constants.StaticPrefix).Handler(
		http.StripPrefix(
			constants.WebuiPrefix+constants.StaticPrefix,
			http.FileServerFS(webfs.WebFS),
		),
	)

	MakeListingRouter(router, db, logger)
	MakeTasksRouter(router, db, logger)
}

func MakeListingRouter(router *mux.Router, db *database.DB, logger *log.Logger) {
	router.Path("/listing/rules").Methods(http.MethodGet).Handler(
		withPermissions(model.PermRulesRead, listing.ListRules(db, logger)),
	)

	router.Path("/listing/clients").Methods(http.MethodGet).Handler(
		withPermissions(model.PermServersRead, listing.ListClients(db, logger)),
	)

	router.Path("/listing/servers").Methods(http.MethodGet).Handler(
		withPermissions(model.PermPartnersRead, listing.ListServers(db, logger)),
	)

	router.Path("/listing/partners").Methods(http.MethodGet).Handler(
		withPermissions(model.PermPartnersRead, listing.ListPartners(db, logger)),
	)

	router.Path("/listing/local_accounts").Methods(http.MethodGet).Handler(
		withPermissions(model.PermPartnersRead, listing.ListLocalAccounts(db, logger)),
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
