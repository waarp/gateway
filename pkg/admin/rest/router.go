package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
	"github.com/gorilla/mux"
)

const (
	// APIPath is the root path for the Rest API endpoints
	APIPath = "/api"

	// StatusPath is the access path to the status entry point.
	StatusPath = "/status"

	// UsersPath is the access path to the status entry point.
	UsersPath = "/users"

	// LocalAgentsPath is the access path to the local servers entry point.
	LocalAgentsPath = "/servers"

	// RemoteAgentsPath is the access path to the partners entry point.
	RemoteAgentsPath = "/partners"

	// LocalAccountsPath is the access path to the local gateway accounts entry point.
	LocalAccountsPath = "/accounts"

	// RemoteAccountsPath is the access path to the distant partners accounts entry point.
	RemoteAccountsPath = "/accounts"

	// CertificatesPath is the access path to the account certificates entry point.
	CertificatesPath = "/certificates"

	// TransfersPath is the access path to the transfers entry point.
	TransfersPath = "/transfers"

	// HistoryPath is the access path to the transfers history entry point.
	HistoryPath = "/history"

	// RulesPath is the access path to the transfers rules entry point.
	RulesPath = "/rules"

	// RulePermissionPath is the access path to the transfer rule permissions
	// entry point.
	RulePermissionPath = "/access"
)

func makeUsersHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	usersHandler := apiHandler.PathPrefix(UsersPath).Subrouter()
	usersHandler.HandleFunc("", listUsers(logger, db)).
		Methods(http.MethodGet)
	usersHandler.HandleFunc("", createUser(logger, db)).
		Methods(http.MethodPost)

	userHandler := usersHandler.PathPrefix("/{user:[^\\/]+}").Subrouter()
	userHandler.HandleFunc("", getUser(logger, db)).
		Methods(http.MethodGet)
	userHandler.HandleFunc("", deleteUser(logger, db)).
		Methods(http.MethodDelete)
	userHandler.HandleFunc("", updateUser(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

//nolint:dupl
func makeLocalAgentsHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	localAgentsHandler := apiHandler.PathPrefix(LocalAgentsPath).Subrouter()
	localAgentsHandler.HandleFunc("", listLocalAgents(logger, db)).
		Methods(http.MethodGet)
	localAgentsHandler.HandleFunc("", createLocalAgent(logger, db)).
		Methods(http.MethodPost)

	locAgHandler := localAgentsHandler.PathPrefix("/{local_agent:[^\\/]+}").Subrouter()
	locAgHandler.HandleFunc("", getLocalAgent(logger, db)).
		Methods(http.MethodGet)
	locAgHandler.HandleFunc("", deleteLocalAgent(logger, db)).
		Methods(http.MethodDelete)
	locAgHandler.HandleFunc("", updateLocalAgent(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	locAgHandler.HandleFunc("/authorize/{rule:[^\\/]+}", authorizeLocalAgent(logger, db)).
		Methods(http.MethodPut)
	locAgHandler.HandleFunc("/revoke/{rule:[^\\/]+}", revokeLocalAgent(logger, db)).
		Methods(http.MethodPut)

	certificatesHandler := locAgHandler.PathPrefix(CertificatesPath).Subrouter()
	certificatesHandler.HandleFunc("", listLocAgentCerts(logger, db)).
		Methods(http.MethodGet)
	certificatesHandler.HandleFunc("", createLocAgentCert(logger, db)).
		Methods(http.MethodPost)

	certHandler := certificatesHandler.PathPrefix("/{certificate:[^\\/]+}").Subrouter()
	certHandler.HandleFunc("", getLocAgentCert(logger, db)).
		Methods(http.MethodGet)
	certHandler.HandleFunc("", deleteLocAgentCert(logger, db)).
		Methods(http.MethodDelete)
	certHandler.HandleFunc("", updateLocAgentCert(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	makeLocalAccountsHandler(logger, db, locAgHandler)
}

//nolint:dupl
func makeRemoteAgentsHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	remoteAgentsHandler := apiHandler.PathPrefix(RemoteAgentsPath).Subrouter()
	remoteAgentsHandler.HandleFunc("", listRemoteAgents(logger, db)).
		Methods(http.MethodGet)
	remoteAgentsHandler.HandleFunc("", createRemoteAgent(logger, db)).
		Methods(http.MethodPost)

	remAgHandler := remoteAgentsHandler.PathPrefix("/{remote_agent:[^\\/]+}").Subrouter()
	remAgHandler.HandleFunc("", getRemoteAgent(logger, db)).
		Methods(http.MethodGet)
	remAgHandler.HandleFunc("", deleteRemoteAgent(logger, db)).
		Methods(http.MethodDelete)
	remAgHandler.HandleFunc("", updateRemoteAgent(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	remAgHandler.HandleFunc("/authorize/{rule:[^\\/]+}", authorizeRemoteAgent(logger, db)).
		Methods(http.MethodPut)
	remAgHandler.HandleFunc("/revoke/{rule:[^\\/]+}", revokeRemoteAgent(logger, db)).
		Methods(http.MethodPut)

	certificatesHandler := remAgHandler.PathPrefix(CertificatesPath).Subrouter()
	certificatesHandler.HandleFunc("", listRemAgentCerts(logger, db)).
		Methods(http.MethodGet)
	certificatesHandler.HandleFunc("", createRemAgentCert(logger, db)).
		Methods(http.MethodPost)

	certHandler := certificatesHandler.PathPrefix("/{certificate:[^\\/]+}").Subrouter()
	certHandler.HandleFunc("", getRemAgentCert(logger, db)).
		Methods(http.MethodGet)
	certHandler.HandleFunc("", deleteRemAgentCert(logger, db)).
		Methods(http.MethodDelete)
	certHandler.HandleFunc("", updateRemAgentCert(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	makeRemoteAccountsHandler(logger, db, remAgHandler)
}

//nolint:dupl
func makeLocalAccountsHandler(logger *log.Logger, db *database.DB, agentHandler *mux.Router) {
	localAccountsHandler := agentHandler.PathPrefix(LocalAccountsPath).Subrouter()
	localAccountsHandler.HandleFunc("", listLocalAccounts(logger, db)).
		Methods(http.MethodGet)
	localAccountsHandler.HandleFunc("", createLocalAccount(logger, db)).
		Methods(http.MethodPost)

	locAcHandler := localAccountsHandler.PathPrefix("/{local_account:[^\\/]+}").Subrouter()
	locAcHandler.HandleFunc("", getLocalAccount(logger, db)).
		Methods(http.MethodGet)
	locAcHandler.HandleFunc("", deleteLocalAccount(logger, db)).
		Methods(http.MethodDelete)
	locAcHandler.HandleFunc("", updateLocalAccount(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	locAcHandler.HandleFunc("/authorize/{rule:[^\\/]+}", authorizeLocalAccount(logger, db)).
		Methods(http.MethodPut)
	locAcHandler.HandleFunc("/revoke/{rule:[^\\/]+}", revokeLocalAccount(logger, db)).
		Methods(http.MethodPut)

	certificatesHandler := locAcHandler.PathPrefix(CertificatesPath).Subrouter()
	certificatesHandler.HandleFunc("", listLocAccountCerts(logger, db)).
		Methods(http.MethodGet)
	certificatesHandler.HandleFunc("", createLocAccountCert(logger, db)).
		Methods(http.MethodPost)

	certHandler := certificatesHandler.PathPrefix("/{certificate:[^\\/]+}").Subrouter()
	certHandler.HandleFunc("", getLocAccountCert(logger, db)).
		Methods(http.MethodGet)
	certHandler.HandleFunc("", deleteLocAccountCert(logger, db)).
		Methods(http.MethodDelete)
	certHandler.HandleFunc("", updateLocAccountCert(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

//nolint:dupl
func makeRemoteAccountsHandler(logger *log.Logger, db *database.DB, agentHandler *mux.Router) {
	remoteAccountsHandler := agentHandler.PathPrefix(RemoteAccountsPath).Subrouter()
	remoteAccountsHandler.HandleFunc("", listRemoteAccounts(logger, db)).
		Methods(http.MethodGet)
	remoteAccountsHandler.HandleFunc("", createRemoteAccount(logger, db)).
		Methods(http.MethodPost)

	remAcHandler := remoteAccountsHandler.PathPrefix("/{remote_account:[^\\/]+}").Subrouter()
	remAcHandler.HandleFunc("", getRemoteAccount(logger, db)).
		Methods(http.MethodGet)
	remAcHandler.HandleFunc("", deleteRemoteAccount(logger, db)).
		Methods(http.MethodDelete)
	remAcHandler.HandleFunc("", updateRemoteAccount(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	remAcHandler.HandleFunc("/authorize/{rule:[^\\/]+}", authorizeRemoteAccount(logger, db)).
		Methods(http.MethodPut)
	remAcHandler.HandleFunc("/revoke/{rule:[^\\/]+}", revokeRemoteAccount(logger, db)).
		Methods(http.MethodPut)

	certificatesHandler := remAcHandler.PathPrefix(CertificatesPath).Subrouter()
	certificatesHandler.HandleFunc("", listRemAccountCerts(logger, db)).
		Methods(http.MethodGet)
	certificatesHandler.HandleFunc("", createRemAccountCert(logger, db)).
		Methods(http.MethodPost)

	certHandler := certificatesHandler.PathPrefix("/{certificate:[^\\/]+}").Subrouter()
	certHandler.HandleFunc("", getRemAccountCert(logger, db)).
		Methods(http.MethodGet)
	certHandler.HandleFunc("", deleteRemAccountCert(logger, db)).
		Methods(http.MethodDelete)
	certHandler.HandleFunc("", updateRemAccountCert(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

func makeTransfersHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	transfersHandler := apiHandler.PathPrefix(TransfersPath).Subrouter()
	transfersHandler.HandleFunc("", listTransfers(logger, db)).
		Methods(http.MethodGet)
	transfersHandler.HandleFunc("", createTransfer(logger, db)).
		Methods(http.MethodPost)
	transferHandler := transfersHandler.PathPrefix("/{transfer:[0-9]+}").Subrouter()
	transferHandler.HandleFunc("", getTransfer(logger, db)).
		Methods(http.MethodGet)
	transferHandler.HandleFunc("/pause", pauseTransfer(logger, db)).
		Methods(http.MethodPut)
	transferHandler.HandleFunc("/cancel", cancelTransfer(logger, db)).
		Methods(http.MethodPut)
	transferHandler.HandleFunc("/resume", resumeTransfer(logger, db)).
		Methods(http.MethodPut)
}

func makeHistoryHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	historyHandler := apiHandler.PathPrefix(HistoryPath).Subrouter()
	historyHandler.HandleFunc("", listHistory(logger, db)).
		Methods(http.MethodGet)
	histHandler := historyHandler.PathPrefix("/{history:[0-9]+}").Subrouter()
	histHandler.HandleFunc("", getHistory(logger, db)).
		Methods(http.MethodGet)
	histHandler.HandleFunc("/retry", retryTransfer(logger, db)).
		Methods(http.MethodPut)
}

func makeRulesHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	rulesHandler := apiHandler.PathPrefix(RulesPath).Subrouter()
	rulesHandler.HandleFunc("", listRules(logger, db)).Methods(http.MethodGet)
	rulesHandler.HandleFunc("", createRule(logger, db)).Methods(http.MethodPost)

	ruleHandler := rulesHandler.PathPrefix("/{rule:[^\\/]+}").Subrouter()
	ruleHandler.HandleFunc("", getRule(logger, db)).Methods(http.MethodGet)
	ruleHandler.HandleFunc("", updateRule(logger, db)).Methods(http.MethodPatch, http.MethodPut)
	ruleHandler.HandleFunc("", deleteRule(logger, db)).Methods(http.MethodDelete)
	ruleHandler.HandleFunc("/allow_all", allowAllRule(logger, db)).Methods(http.MethodPut)
}

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
func MakeRESTHandler(logger *log.Logger, db *database.DB, adminHandler *mux.Router,
	services map[string]service.Service) {

	apiHandler := adminHandler.PathPrefix(APIPath).Subrouter()

	apiHandler.HandleFunc(StatusPath, getStatus(logger, services)).
		Methods(http.MethodGet)

	makeUsersHandler(logger, db, apiHandler)
	makeLocalAgentsHandler(logger, db, apiHandler)
	makeRemoteAgentsHandler(logger, db, apiHandler)
	makeTransfersHandler(logger, db, apiHandler)
	makeHistoryHandler(logger, db, apiHandler)
	makeRulesHandler(logger, db, apiHandler)
}
