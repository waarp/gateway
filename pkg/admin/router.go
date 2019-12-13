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
func MakeHandler(logger *log.Logger, db *database.Db, services map[string]service.Service) http.Handler {

	// REST handler
	adminHandler := mux.NewRouter()
	adminHandler.Use(mux.CORSMethodMiddleware(adminHandler), Authentication(logger, db))

	rest.MakeRESTHandler(logger, db, adminHandler, services)

	return adminHandler
}
