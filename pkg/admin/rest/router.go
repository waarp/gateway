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
	LocalAccountsPath = "/local_accounts"

	// RemoteAccountsPath is the access path to the distant partners accounts entry point.
	RemoteAccountsPath = "/remote_accounts"

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

	// RuleTasksPath is the access path to the transfer rule tasks entry point.
	RuleTasksPath = "/tasks"
)

func makeUsersHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	usersHandler := apiHandler.PathPrefix(UsersPath).Subrouter()
	usersHandler.HandleFunc("", listUsers(logger, db)).
		Methods(http.MethodGet)
	usersHandler.HandleFunc("", createUser(logger, db)).
		Methods(http.MethodPost)

	userHandler := usersHandler.PathPrefix("/{user:[0-9]+}").Subrouter()
	userHandler.HandleFunc("", getUser(logger, db)).
		Methods(http.MethodGet)
	userHandler.HandleFunc("", deleteUser(logger, db)).
		Methods(http.MethodDelete)
	userHandler.HandleFunc("", updateUser(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

func makeLocalAgentsHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	localAgentsHandler := apiHandler.PathPrefix(LocalAgentsPath).Subrouter()
	localAgentsHandler.HandleFunc("", listLocalAgents(logger, db)).
		Methods(http.MethodGet)
	localAgentsHandler.HandleFunc("", createLocalAgent(logger, db)).
		Methods(http.MethodPost)

	locAgHandler := localAgentsHandler.PathPrefix("/{local_agent:.+}").Subrouter()
	locAgHandler.HandleFunc("", getLocalAgent(logger, db)).
		Methods(http.MethodGet)
	locAgHandler.HandleFunc("", deleteLocalAgent(logger, db)).
		Methods(http.MethodDelete)
	locAgHandler.HandleFunc("", updateLocalAgent(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	makeLocalAccountsHandler(logger, db, locAgHandler)
}

func makeRemoteAgentsHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	remoteAgentsHandler := apiHandler.PathPrefix(RemoteAgentsPath).Subrouter()
	remoteAgentsHandler.HandleFunc("", listRemoteAgents(logger, db)).
		Methods(http.MethodGet)
	remoteAgentsHandler.HandleFunc("", createRemoteAgent(logger, db)).
		Methods(http.MethodPost)

	remAgHandler := remoteAgentsHandler.PathPrefix("/{remote_agent:.+}").Subrouter()
	remAgHandler.HandleFunc("", getRemoteAgent(logger, db)).
		Methods(http.MethodGet)
	remAgHandler.HandleFunc("", deleteRemoteAgent(logger, db)).
		Methods(http.MethodDelete)
	remAgHandler.HandleFunc("", updateRemoteAgent(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)

	makeRemoteAccountsHandler(logger, db, remAgHandler)
}

func makeLocalAccountsHandler(logger *log.Logger, db *database.DB, agentHandler *mux.Router) {
	localAccountsHandler := agentHandler.PathPrefix(LocalAccountsPath).Subrouter()
	localAccountsHandler.HandleFunc("", listLocalAccounts(logger, db)).
		Methods(http.MethodGet)
	localAccountsHandler.HandleFunc("", createLocalAccount(logger, db)).
		Methods(http.MethodPost)

	locAcHandler := localAccountsHandler.PathPrefix("/{local_account:.+}").Subrouter()
	locAcHandler.HandleFunc("", getLocalAccount(logger, db)).
		Methods(http.MethodGet)
	locAcHandler.HandleFunc("", deleteLocalAccount(logger, db)).
		Methods(http.MethodDelete)
	locAcHandler.HandleFunc("", updateLocalAccount(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

func makeRemoteAccountsHandler(logger *log.Logger, db *database.DB, agentHandler *mux.Router) {
	remoteAccountsHandler := agentHandler.PathPrefix(RemoteAccountsPath).Subrouter()
	remoteAccountsHandler.HandleFunc("", listRemoteAccounts(logger, db)).
		Methods(http.MethodGet)
	remoteAccountsHandler.HandleFunc("", createRemoteAccount(logger, db)).
		Methods(http.MethodPost)

	remAcHandler := remoteAccountsHandler.PathPrefix("/{remote_account:.+}").Subrouter()
	remAcHandler.HandleFunc("", getRemoteAccount(logger, db)).
		Methods(http.MethodGet)
	remAcHandler.HandleFunc("", deleteRemoteAccount(logger, db)).
		Methods(http.MethodDelete)
	remAcHandler.HandleFunc("", updateRemoteAccount(logger, db)).
		Methods(http.MethodPatch, http.MethodPut)
}

func makeCertificatesHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
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
	histHandler.HandleFunc("/restart", restartTransfer(logger, db)).
		Methods(http.MethodPut)
}

func makeRulesHandler(logger *log.Logger, db *database.DB, apiHandler *mux.Router) {
	rulesHandler := apiHandler.PathPrefix(RulesPath).Subrouter()
	rulesHandler.HandleFunc("", listRules(logger, db)).Methods(http.MethodGet)
	rulesHandler.HandleFunc("", createRule(logger, db)).Methods(http.MethodPost)

	ruleHandler := rulesHandler.PathPrefix("/{rule:[0-9]+}").Subrouter()
	ruleHandler.HandleFunc("", getRule(logger, db)).Methods(http.MethodGet)
	ruleHandler.HandleFunc("", updateRule(logger, db)).Methods(http.MethodPatch, http.MethodPut)
	ruleHandler.HandleFunc("", deleteRule(logger, db)).Methods(http.MethodDelete)

	permHandler := ruleHandler.PathPrefix(RulePermissionPath).Subrouter()
	permHandler.HandleFunc("", createAccess(logger, db)).Methods(http.MethodPost)
	permHandler.HandleFunc("", listAccess(logger, db)).Methods(http.MethodGet)
	permHandler.HandleFunc("", deleteAccess(logger, db)).Methods(http.MethodDelete)

	taskHandler := ruleHandler.PathPrefix(RuleTasksPath).Subrouter()
	taskHandler.HandleFunc("", listTasks(logger, db)).Methods(http.MethodGet)
	taskHandler.HandleFunc("", updateTasks(logger, db)).Methods(http.MethodPut)
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
	makeCertificatesHandler(logger, db, apiHandler)
	makeTransfersHandler(logger, db, apiHandler)
	makeHistoryHandler(logger, db, apiHandler)
	makeRulesHandler(logger, db, apiHandler)
}
