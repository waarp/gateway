package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
)

const (
	// APIPath is the root path for the Rest API endpoints
	APIPath = "/api"
)

// MakeHandler returns the router for the REST & Admin http interface
func MakeHandler(logger *log.Logger, db *database.DB, services map[string]service.Service) http.Handler {

	// REST handler
	adminHandler := mux.NewRouter()
	adminHandler.Use(mux.CORSMethodMiddleware(adminHandler), Authentication(logger, db))

	rest.MakeRESTHandler(logger, db, adminHandler, services)

	return adminHandler
}
