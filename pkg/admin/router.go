package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
)

func makeRulesHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	rulesHandler := apiHandler.PathPrefix(RulesPath).Subrouter()
	rulesHandler.HandleFunc("", listRules(logger, db)).Methods(http.MethodGet)
	rulesHandler.HandleFunc("", createRule(logger, db)).Methods(http.MethodPost)

	ruleHandler := rulesHandler.PathPrefix("/{rule:[0-9]+}").Subrouter()
	ruleHandler.HandleFunc("", getRule(logger, db)).Methods(http.MethodGet)
	ruleHandler.HandleFunc("", deleteRule(logger, db)).Methods(http.MethodDelete)

	permHandler := ruleHandler.PathPrefix(RulePermissionPath).Subrouter()
	permHandler.HandleFunc("", createAccess(logger, db)).Methods(http.MethodPost)
	permHandler.HandleFunc("", listAccess(logger, db)).Methods(http.MethodGet)
	permHandler.HandleFunc("", deleteAccess(logger, db)).Methods(http.MethodDelete)

	taskHandler := ruleHandler.PathPrefix(RuleTasksPath).Subrouter()
	taskHandler.HandleFunc("", listTasks(logger, db)).Methods(http.MethodGet)
	taskHandler.HandleFunc("", updateTasks(logger, db)).Methods(http.MethodPut)
}

// MakeHandler returns the router for the REST & Admin http interface
func MakeHandler(logger *log.Logger, db *database.Db, services map[string]service.Service) http.Handler {

	// REST handler
	handler := mux.NewRouter()
	handler.Use(mux.CORSMethodMiddleware(handler), Authentication(logger, db))
	apiHandler := handler.PathPrefix(APIPath).Subrouter()
	apiHandler.HandleFunc(StatusPath, getStatus(logger, services)).
		Methods(http.MethodGet)

	makeRulesHandler(logger, db, apiHandler)

	rest.MakeRESTHandler(logger, db, apiHandler)

	return handler
}
