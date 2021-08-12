package rest

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
	"github.com/gorilla/mux"
)

// The definitions of all the REST entry points paths.
const (
	statusPath = "/api/status"

	usersPath = "/api/users"
	userPath  = "/api/users/{user}"

	rulesPath     = "/api/rules"
	rulePath      = "/api/rules/{rule}/{direction:send|receive}"
	ruleAllowPath = "/api/rules/{rule}/{direction:send|receive}/allow_all"

	transfersPath   = "/api/transfers"
	transferPath    = "/api/transfers/{transfer}"
	transPausePath  = "/api/transfers/{transfer}/pause"
	transResumePath = "/api/transfers/{transfer}/resume"
	transCancelPath = "/api/transfers/{transfer}/cancel"

	historiesPath = "/api/history"
	historyPath   = "/api/history/{history}"
	histRetryPath = "/api/history/{history}/retry"

	serversPath     = "/api/servers"
	serverPath      = "/api/servers/{server}"
	serverCertsPath = "/api/servers/{server}/certificates"
	serverCertPath  = "/api/servers/{server}/certificates/{certificate}"
	serverAuthPath  = "/api/servers/{server}/authorize/{rule}/{direction:send|receive}"
	serverRevPath   = "/api/servers/{server}/revoke/{rule}/{direction:send|receive}"

	locAccountsPath = "/api/servers/{server}/accounts"
	locAccountPath  = "/api/servers/{server}/accounts/{local_account}"
	locAccCertsPath = "/api/servers/{server}/accounts/{local_account}/certificates"
	locAccCertPath  = "/api/servers/{server}/accounts/{local_account}/certificates/{certificate}"
	locAccAuthPath  = "/api/servers/{server}/accounts/{local_account}/authorize/{rule}/{direction:send|receive}"
	locAccRevPath   = "/api/servers/{server}/accounts/{local_account}/revoke/{rule}/{direction:send|receive}"

	partnersPath     = "/api/partners"
	partnerPath      = "/api/partners/{partner}"
	partnerCertsPath = "/api/partners/{partner}/certificates"
	partnerCertPath  = "/api/partners/{partner}/certificates/{certificate}"
	partnerAuthPath  = "/api/partners/{partner}/authorize/{rule}/{direction:send|receive}"
	partnerRevPath   = "/api/partners/{partner}/revoke/{rule}/{direction:send|receive}"

	remAccountsPath = "/api/partners/{partner}/accounts"
	remAccountPath  = "/api/partners/{partner}/accounts/{remote_account}"
	remAccCertsPath = "/api/partners/{partner}/accounts/{remote_account}/certificates"
	remAccCertPath  = "/api/partners/{partner}/accounts/{remote_account}/certificates/{certificate}"
	remAccAuthPath  = "/api/partners/{partner}/accounts/{remote_account}/authorize/{rule}/{direction:send|receive}"
	remAccRevPath   = "/api/partners/{partner}/accounts/{remote_account}/revoke/{rule}/{direction:send|receive}"
)

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
//nolint:funlen
func MakeRESTHandler(logger *log.Logger, db *database.DB, router *mux.Router,
	services map[string]service.Service) {

	router.StrictSlash(true)

	router.Path(statusPath).Methods(http.MethodGet).Handler(getStatus(logger, services))
	mkHandler := makeHandlerFactory(logger, db, router)

	// Users
	mkHandler(usersPath, listUsers, model.PermUsersRead, http.MethodGet)
	mkHandler(usersPath, addUser, model.PermUsersWrite, http.MethodPost)
	mkHandler(userPath, getUser, model.PermUsersRead, http.MethodGet)
	mkHandler(userPath, updateUser, model.PermUsersWrite, http.MethodPatch)
	mkHandler(userPath, replaceUser, model.PermUsersWrite, http.MethodPut)
	mkHandler(userPath, deleteUser, model.PermUsersDelete, http.MethodDelete)

	// Rules
	mkHandler(rulesPath, listRules, model.PermRulesRead, http.MethodGet)
	mkHandler(rulesPath, addRule, model.PermRulesWrite, http.MethodPost)
	mkHandler(rulePath, getRule, model.PermRulesRead, http.MethodGet)
	mkHandler(rulePath, updateRule, model.PermRulesWrite, http.MethodPatch)
	mkHandler(rulePath, replaceRule, model.PermRulesWrite, http.MethodPut)
	mkHandler(rulePath, deleteRule, model.PermRulesDelete, http.MethodDelete)
	mkHandler(ruleAllowPath, allowAllRule, model.PermRulesWrite, http.MethodPut)

	// Transfers
	mkHandler(transfersPath, listTransfers, model.PermTransfersRead, http.MethodGet)
	mkHandler(transfersPath, addTransfer, model.PermTransfersWrite, http.MethodPost)
	mkHandler(transferPath, getTransfer, model.PermTransfersRead, http.MethodGet)
	mkHandler(transPausePath, pauseTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(transResumePath, resumeTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(transCancelPath, cancelTransfer, model.PermTransfersWrite, http.MethodPut)

	// History
	mkHandler(historiesPath, listHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(historyPath, getHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(histRetryPath, retryTransfer, model.PermTransfersWrite, http.MethodPut)

	// Servers
	mkHandler(serversPath, listServers, model.PermServersRead, http.MethodGet)
	mkHandler(serversPath, addServer, model.PermServersWrite, http.MethodPost)
	mkHandler(serverPath, getServer, model.PermServersRead, http.MethodGet)
	mkHandler(serverPath, deleteServer, model.PermServersDelete, http.MethodDelete)
	mkHandler(serverPath, updateServer, model.PermServersWrite, http.MethodPatch)
	mkHandler(serverPath, replaceServer, model.PermServersWrite, http.MethodPut)
	mkHandler(serverCertsPath, listServerCerts, model.PermServersRead, http.MethodGet)
	mkHandler(serverCertsPath, addServerCert, model.PermServersWrite, http.MethodPost)
	mkHandler(serverCertPath, getServerCert, model.PermServersRead, http.MethodGet)
	mkHandler(serverCertPath, deleteServerCert, model.PermServersWrite, http.MethodDelete)
	mkHandler(serverCertPath, updateServerCert, model.PermServersWrite, http.MethodPatch)
	mkHandler(serverCertPath, replaceServerCert, model.PermServersWrite, http.MethodPut)
	mkHandler(serverAuthPath, authorizeServer, model.PermRulesWrite, http.MethodPut)
	mkHandler(serverRevPath, revokeServer, model.PermRulesWrite, http.MethodPut)

	// Local accounts
	mkHandler(locAccountsPath, listLocalAccounts, model.PermServersRead, http.MethodGet)
	mkHandler(locAccountsPath, addLocalAccount, model.PermServersWrite, http.MethodPost)
	mkHandler(locAccountPath, getLocalAccount, model.PermServersRead, http.MethodGet)
	mkHandler(locAccountPath, deleteLocalAccount, model.PermServersDelete, http.MethodDelete)
	mkHandler(locAccountPath, updateLocalAccount, model.PermServersWrite, http.MethodPatch)
	mkHandler(locAccountPath, replaceLocalAccount, model.PermServersWrite, http.MethodPut)
	mkHandler(locAccCertsPath, listLocAccountCerts, model.PermServersRead, http.MethodGet)
	mkHandler(locAccCertsPath, addLocAccountCert, model.PermServersWrite, http.MethodPost)
	mkHandler(locAccCertPath, getLocAccountCert, model.PermServersRead, http.MethodGet)
	mkHandler(locAccCertPath, deleteLocAccountCert, model.PermServersWrite, http.MethodDelete)
	mkHandler(locAccCertPath, updateLocAccountCert, model.PermServersWrite, http.MethodPatch)
	mkHandler(locAccCertPath, replaceLocAccountCert, model.PermServersWrite, http.MethodPut)
	mkHandler(locAccAuthPath, authorizeLocalAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(locAccRevPath, revokeLocalAccount, model.PermRulesWrite, http.MethodPut)

	// Partners
	mkHandler(partnersPath, listPartners, model.PermPartnersRead, http.MethodGet)
	mkHandler(partnersPath, addPartner, model.PermPartnersWrite, http.MethodPost)
	mkHandler(partnerPath, getPartner, model.PermPartnersRead, http.MethodGet)
	mkHandler(partnerPath, deletePartner, model.PermPartnersDelete, http.MethodDelete)
	mkHandler(partnerPath, updatePartner, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(partnerPath, replacePartner, model.PermPartnersWrite, http.MethodPut)
	mkHandler(partnerCertsPath, listPartnerCerts, model.PermPartnersRead, http.MethodGet)
	mkHandler(partnerCertsPath, addPartnerCert, model.PermPartnersWrite, http.MethodPost)
	mkHandler(partnerCertPath, getPartnerCert, model.PermPartnersRead, http.MethodGet)
	mkHandler(partnerCertPath, deletePartnerCert, model.PermPartnersWrite, http.MethodDelete)
	mkHandler(partnerCertPath, updatePartnerCert, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(partnerCertPath, replacePartnerCert, model.PermPartnersWrite, http.MethodPut)
	mkHandler(partnerAuthPath, authorizePartner, model.PermRulesWrite, http.MethodPut)
	mkHandler(partnerRevPath, revokePartner, model.PermRulesWrite, http.MethodPut)

	// Remote accounts
	mkHandler(remAccountsPath, listRemoteAccounts, model.PermPartnersRead, http.MethodGet)
	mkHandler(remAccountsPath, addRemoteAccount, model.PermPartnersWrite, http.MethodPost)
	mkHandler(remAccountPath, getRemoteAccount, model.PermPartnersRead, http.MethodGet)
	mkHandler(remAccountPath, deleteRemoteAccount, model.PermPartnersDelete, http.MethodDelete)
	mkHandler(remAccountPath, updateRemoteAccount, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(remAccountPath, replaceRemoteAccount, model.PermPartnersWrite, http.MethodPut)
	mkHandler(remAccCertsPath, listRemAccountCerts, model.PermPartnersRead, http.MethodGet)
	mkHandler(remAccCertsPath, addRemAccountCert, model.PermPartnersWrite, http.MethodPost)
	mkHandler(remAccCertPath, getRemAccountCert, model.PermPartnersRead, http.MethodGet)
	mkHandler(remAccCertPath, deleteRemAccountCert, model.PermPartnersWrite, http.MethodDelete)
	mkHandler(remAccCertPath, updateRemAccountCert, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(remAccCertPath, replaceRemAccountCert, model.PermPartnersWrite, http.MethodPut)
	mkHandler(remAccAuthPath, authorizeRemoteAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(remAccRevPath, revokeRemoteAccount, model.PermRulesWrite, http.MethodPut)
}
