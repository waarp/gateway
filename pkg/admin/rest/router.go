package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/gorilla/mux"
)

func makeLocalAccountsHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	localAccountsHandler := apiHandler.PathPrefix(LocalAccountsPath).Subrouter()
	localAccountsHandler.HandleFunc("", listLocalAccounts(logger, db)).
		Methods(http.MethodGet)
	localAccountsHandler.HandleFunc("", createLocalAccount(logger, db)).
		Methods(http.MethodPost)

	locAcHandler := localAccountsHandler.PathPrefix("/{local_account:[0-9]+}").Subrouter()
	locAcHandler.HandleFunc("", getLocalAccount(logger, db)).
		Methods(http.MethodGet)
	locAcHandler.HandleFunc("", deleteLocalAccount(logger, db)).
		Methods(http.MethodDelete)
	locAcHandler.HandleFunc("", updateLocalAccount(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

func makeRemoteAccountsHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	remoteAccountsHandler := apiHandler.PathPrefix(RemoteAccountsPath).Subrouter()
	remoteAccountsHandler.HandleFunc("", listRemoteAccounts(logger, db)).
		Methods(http.MethodGet)
	remoteAccountsHandler.HandleFunc("", createRemoteAccount(logger, db)).
		Methods(http.MethodPost)

	remAcHandler := remoteAccountsHandler.PathPrefix("/{remote_account:[0-9]+}").Subrouter()
	remAcHandler.HandleFunc("", getRemoteAccount(logger, db)).
		Methods(http.MethodGet)
	remAcHandler.HandleFunc("", deleteRemoteAccount(logger, db)).
		Methods(http.MethodDelete)
	remAcHandler.HandleFunc("", updateRemoteAccount(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
func MakeRESTHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	makeLocalAccountsHandler(logger, db, apiHandler)
	makeRemoteAccountsHandler(logger, db, apiHandler)
}
