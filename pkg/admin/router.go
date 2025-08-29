package admin

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/debug"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// MakeHandler returns the router for the REST & Admin http interface.
func MakeHandler(logger *log.Logger, db *database.DB) *mux.Router {
	adminHandler := mux.NewRouter()
	adminHandler.StrictSlash(true)
	adminHandler.Use(
		mux.CORSMethodMiddleware(adminHandler),
	)

	restRouter := adminHandler.PathPrefix(rest.Prefix).Subrouter()
	debugRouter := adminHandler.PathPrefix(debug.Prefix).Subrouter()
	guiRouter := adminHandler.PathPrefix(gui.Prefix).Subrouter()

	rest.MakeRESTHandler(logger, db, restRouter)
	debug.AddDebugHandler(debugRouter, logger, db)
	gui.AddGUIRouter(guiRouter, logger, db)

	adminHandler.HandleFunc("/", guiRedirect)

	return adminHandler
}

func guiRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, gui.Prefix + "/", http.StatusPermanentRedirect)
}
