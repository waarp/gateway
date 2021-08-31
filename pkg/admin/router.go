package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
)

// MakeHandler returns the router for the REST & Admin http interface
func MakeHandler(logger *log.Logger, db *database.DB, coreServices map[string]service.Service,
	protoServices map[string]service.ProtoService) http.Handler {

	// REST handler
	adminHandler := mux.NewRouter()
	adminHandler.Use(mux.CORSMethodMiddleware(adminHandler), authentication(logger, db))

	rest.MakeRESTHandler(logger, db, adminHandler, coreServices, protoServices)

	return adminHandler
}
