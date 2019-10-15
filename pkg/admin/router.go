package admin

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
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

func makeTransfersHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	transfersHandler := apiHandler.PathPrefix(TransfersPath).Subrouter()
	transfersHandler.HandleFunc("", listTransfers(logger, db)).
		Methods(http.MethodGet)
	transfersHandler.HandleFunc("", addTransfer(logger, db)).
		Methods(http.MethodPost)
	transferHandler := transfersHandler.PathPrefix("/{transfer:[0-9]+}").Subrouter()
	transferHandler.HandleFunc("", getTransfer(logger, db)).
		Methods(http.MethodGet)
}

func makeHistoryHandler(logger *log.Logger, db *database.Db, apiHandler *mux.Router) {
	historyHandler := apiHandler.PathPrefix(HistoryPath).Subrouter()
	historyHandler.HandleFunc("", listHistory(logger, db)).
		Methods(http.MethodGet)
	histHandler := historyHandler.PathPrefix("/{history:[0-9]+}").Subrouter()
	histHandler.HandleFunc("", getHistory(logger, db)).
		Methods(http.MethodGet)
}

// MakeHandler returns the router for the REST & Admin http interface
func MakeHandler(logger *log.Logger, db *database.Db, services map[string]service.Service) http.Handler {

	// REST handler
	handler := mux.NewRouter()
	handler.Use(mux.CORSMethodMiddleware(handler), Authentication(logger, db))
	apiHandler := handler.PathPrefix(APIPath).Subrouter()
	apiHandler.HandleFunc(StatusPath, getStatus(logger, services)).
		Methods(http.MethodGet)

	makeLocalAgentsHandler(logger, db, apiHandler)
	makeRemoteAgentsHandler(logger, db, apiHandler)
	makeLocalAccountsHandler(logger, db, apiHandler)
	makeRemoteAccountsHandler(logger, db, apiHandler)
	makeCertificatesHandler(logger, db, apiHandler)
	makeTransfersHandler(logger, db, apiHandler)
	makeHistoryHandler(logger, db, apiHandler)

	return handler
}
