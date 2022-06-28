package admin

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// MakeHandler returns the router for the REST & Admin http interface.
func MakeHandler(logger *log.Logger, db *database.DB, coreServices map[string]service.Service,
	protoServices map[string]service.ProtoService,
) http.Handler {
	// REST handler
	adminHandler := mux.NewRouter()
	adminHandler.Use(mux.CORSMethodMiddleware(adminHandler), authentication(logger, db))

	rest.MakeRESTHandler(logger, db, adminHandler, coreServices, protoServices)

	return adminHandler
}
