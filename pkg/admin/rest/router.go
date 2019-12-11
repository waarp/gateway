package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"github.com/gorilla/mux"
)

const (
	// APIPath is the root path for the Rest API endpoints
	APIPath = "/api"

	// LocalAgentsPath is the access path to the local servers entry point.
	LocalAgentsPath = "/servers"
	// RemoteAgentsPath is the access path to the partners entry point.
	RemoteAgentsPath = "/partners"
	// LocalAccountsPath is the access path to the local gateway accounts entry point.
	LocalAccountsPath = "/local_accounts"
	// RemoteAccountsPath is the access path to the distant partners accounts entry point.
	RemoteAccountsPath = "/remote_accounts"
	// CertificatesPath is the access path to the account certificates entry point.
	CertificatesPath = "/certificates"
)

func makeLocalAgentsHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	localAgentsHandler := apiHandler.PathPrefix(LocalAgentsPath).Subrouter()
	localAgentsHandler.HandleFunc("", listLocalAgents(logger, db)).
		Methods(http.MethodGet)
	localAgentsHandler.HandleFunc("", createLocalAgent(logger, db)).
		Methods(http.MethodPost)

	locAgHandler := localAgentsHandler.PathPrefix("/{local_agent:[0-9]+}").Subrouter()
	locAgHandler.HandleFunc("", getLocalAgent(logger, db)).
		Methods(http.MethodGet)
	locAgHandler.HandleFunc("", deleteLocalAgent(logger, db)).
		Methods(http.MethodDelete)
	locAgHandler.HandleFunc("", updateLocalAgent(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

func makeRemoteAgentsHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	remoteAgentsHandler := apiHandler.PathPrefix(RemoteAgentsPath).Subrouter()
	remoteAgentsHandler.HandleFunc("", listRemoteAgents(logger, db)).
		Methods(http.MethodGet)
	remoteAgentsHandler.HandleFunc("", createRemoteAgent(logger, db)).
		Methods(http.MethodPost)

	remAgHandler := remoteAgentsHandler.PathPrefix("/{remote_agent:[0-9]+}").Subrouter()
	remAgHandler.HandleFunc("", getRemoteAgent(logger, db)).
		Methods(http.MethodGet)
	remAgHandler.HandleFunc("", deleteRemoteAgent(logger, db)).
		Methods(http.MethodDelete)
	remAgHandler.HandleFunc("", updateRemoteAgent(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

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

func makeCertificatesHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	certificatesHandler := apiHandler.PathPrefix(CertificatesPath).Subrouter()
	certificatesHandler.HandleFunc("", listCertificates(logger, db)).
		Methods(http.MethodGet)
	certificatesHandler.HandleFunc("", createCertificate(logger, db)).
		Methods(http.MethodPost)

	certHandler := certificatesHandler.PathPrefix("/{certificate:[0-9]+}").Subrouter()
	certHandler.HandleFunc("", getCertificate(logger, db)).
		Methods(http.MethodGet)
	certHandler.HandleFunc("", deleteCertificate(logger, db)).
		Methods(http.MethodDelete)
	certHandler.HandleFunc("", updateCertificate(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
func MakeRESTHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	makeLocalAgentsHandler(logger, db, apiHandler)
	makeRemoteAgentsHandler(logger, db, apiHandler)
	makeLocalAccountsHandler(logger, db, apiHandler)
	makeRemoteAccountsHandler(logger, db, apiHandler)
	makeCertificatesHandler(logger, db, apiHandler)
}
