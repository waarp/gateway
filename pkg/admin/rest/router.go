package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/service"
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

	overrideSettingsAddressesPath = "/api/override/addresses"
	overrideSettingsAddressPath   = "/api/override/addresses/{address}"
)

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
//nolint:funlen
func MakeRESTHandler(logger *log.Logger, db *database.DB, router *mux.Router,
	coreServices map[string]service.Service, protoServices map[string]service.ProtoService) {

	router.StrictSlash(true)

	router.Path(statusPath).Methods(http.MethodGet).Handler(getStatus(logger, coreServices, protoServices))
	f := makeHandlerFactory(logger, db, router)

	// Users
	f.mkHandler(usersPath, listUsers, model.PermUsersRead, http.MethodGet)
	f.mkHandler(usersPath, addUser, model.PermUsersWrite, http.MethodPost)
	f.mkHandler(userPath, getUser, model.PermUsersRead, http.MethodGet)
	f.mkHandler(userPath, updateUser, model.PermUsersWrite, http.MethodPatch)
	f.mkHandler(userPath, replaceUser, model.PermUsersWrite, http.MethodPut)
	f.mkHandler(userPath, deleteUser, model.PermUsersDelete, http.MethodDelete)

	// Rules
	f.mkHandler(rulesPath, listRules, model.PermRulesRead, http.MethodGet)
	f.mkHandler(rulesPath, addRule, model.PermRulesWrite, http.MethodPost)
	f.mkHandler(rulePath, getRule, model.PermRulesRead, http.MethodGet)
	f.mkHandler(rulePath, updateRule, model.PermRulesWrite, http.MethodPatch)
	f.mkHandler(rulePath, replaceRule, model.PermRulesWrite, http.MethodPut)
	f.mkHandler(rulePath, deleteRule, model.PermRulesDelete, http.MethodDelete)
	f.mkHandler(ruleAllowPath, allowAllRule, model.PermRulesWrite, http.MethodPut)

	// Transfers
	f.mkHandler(transfersPath, listTransfers, model.PermTransfersRead, http.MethodGet)
	f.mkHandler(transfersPath, addTransfer, model.PermTransfersWrite, http.MethodPost)
	f.mkHandler(transferPath, getTransfer, model.PermTransfersRead, http.MethodGet)
	f.mkHandler(transPausePath, pauseTransfer(protoServices), model.PermTransfersWrite, http.MethodPut)
	f.mkHandler(transResumePath, resumeTransfer, model.PermTransfersWrite, http.MethodPut)
	f.mkHandler(transCancelPath, cancelTransfer(protoServices), model.PermTransfersWrite, http.MethodPut)

	// History
	f.mkHandler(historiesPath, listHistory, model.PermTransfersRead, http.MethodGet)
	f.mkHandler(historyPath, getHistory, model.PermTransfersRead, http.MethodGet)
	f.mkHandler(histRetryPath, retryTransfer, model.PermTransfersWrite, http.MethodPut)

	// Servers
	f.mkHandler(serversPath, listServers, model.PermServersRead, http.MethodGet)
	f.mkHandler(serversPath, addServer, model.PermServersWrite, http.MethodPost)
	f.mkHandler(serverPath, getServer, model.PermServersRead, http.MethodGet)
	f.mkHandler(serverPath, deleteServer, model.PermServersDelete, http.MethodDelete)
	f.mkHandler(serverPath, updateServer, model.PermServersWrite, http.MethodPatch)
	f.mkHandler(serverPath, replaceServer, model.PermServersWrite, http.MethodPut)
	f.mkHandler(serverCertsPath, listServerCerts, model.PermServersRead, http.MethodGet)
	f.mkHandler(serverCertsPath, addServerCert, model.PermServersWrite, http.MethodPost)
	f.mkHandler(serverCertPath, getServerCert, model.PermServersRead, http.MethodGet)
	f.mkHandler(serverCertPath, deleteServerCert, model.PermServersWrite, http.MethodDelete)
	f.mkHandler(serverCertPath, updateServerCert, model.PermServersWrite, http.MethodPatch)
	f.mkHandler(serverCertPath, replaceServerCert, model.PermServersWrite, http.MethodPut)
	f.mkHandler(serverAuthPath, authorizeServer, model.PermRulesWrite, http.MethodPut)
	f.mkHandler(serverRevPath, revokeServer, model.PermRulesWrite, http.MethodPut)

	// Local accounts
	f.mkHandler(locAccountsPath, listLocalAccounts, model.PermServersRead, http.MethodGet)
	f.mkHandler(locAccountsPath, addLocalAccount, model.PermServersWrite, http.MethodPost)
	f.mkHandler(locAccountPath, getLocalAccount, model.PermServersRead, http.MethodGet)
	f.mkHandler(locAccountPath, deleteLocalAccount, model.PermServersDelete, http.MethodDelete)
	f.mkHandler(locAccountPath, updateLocalAccount, model.PermServersWrite, http.MethodPatch)
	f.mkHandler(locAccountPath, replaceLocalAccount, model.PermServersWrite, http.MethodPut)
	f.mkHandler(locAccCertsPath, listLocAccountCerts, model.PermServersRead, http.MethodGet)
	f.mkHandler(locAccCertsPath, addLocAccountCert, model.PermServersWrite, http.MethodPost)
	f.mkHandler(locAccCertPath, getLocAccountCert, model.PermServersRead, http.MethodGet)
	f.mkHandler(locAccCertPath, deleteLocAccountCert, model.PermServersWrite, http.MethodDelete)
	f.mkHandler(locAccCertPath, updateLocAccountCert, model.PermServersWrite, http.MethodPatch)
	f.mkHandler(locAccCertPath, replaceLocAccountCert, model.PermServersWrite, http.MethodPut)
	f.mkHandler(locAccAuthPath, authorizeLocalAccount, model.PermRulesWrite, http.MethodPut)
	f.mkHandler(locAccRevPath, revokeLocalAccount, model.PermRulesWrite, http.MethodPut)

	// Partners
	f.mkHandler(partnersPath, listPartners, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(partnersPath, addPartner, model.PermPartnersWrite, http.MethodPost)
	f.mkHandler(partnerPath, getPartner, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(partnerPath, deletePartner, model.PermPartnersDelete, http.MethodDelete)
	f.mkHandler(partnerPath, updatePartner, model.PermPartnersWrite, http.MethodPatch)
	f.mkHandler(partnerPath, replacePartner, model.PermPartnersWrite, http.MethodPut)
	f.mkHandler(partnerCertsPath, listPartnerCerts, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(partnerCertsPath, addPartnerCert, model.PermPartnersWrite, http.MethodPost)
	f.mkHandler(partnerCertPath, getPartnerCert, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(partnerCertPath, deletePartnerCert, model.PermPartnersWrite, http.MethodDelete)
	f.mkHandler(partnerCertPath, updatePartnerCert, model.PermPartnersWrite, http.MethodPatch)
	f.mkHandler(partnerCertPath, replacePartnerCert, model.PermPartnersWrite, http.MethodPut)
	f.mkHandler(partnerAuthPath, authorizePartner, model.PermRulesWrite, http.MethodPut)
	f.mkHandler(partnerRevPath, revokePartner, model.PermRulesWrite, http.MethodPut)

	// Remote accounts
	f.mkHandler(remAccountsPath, listRemoteAccounts, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(remAccountsPath, addRemoteAccount, model.PermPartnersWrite, http.MethodPost)
	f.mkHandler(remAccountPath, getRemoteAccount, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(remAccountPath, deleteRemoteAccount, model.PermPartnersDelete, http.MethodDelete)
	f.mkHandler(remAccountPath, updateRemoteAccount, model.PermPartnersWrite, http.MethodPatch)
	f.mkHandler(remAccountPath, replaceRemoteAccount, model.PermPartnersWrite, http.MethodPut)
	f.mkHandler(remAccCertsPath, listRemAccountCerts, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(remAccCertsPath, addRemAccountCert, model.PermPartnersWrite, http.MethodPost)
	f.mkHandler(remAccCertPath, getRemAccountCert, model.PermPartnersRead, http.MethodGet)
	f.mkHandler(remAccCertPath, deleteRemAccountCert, model.PermPartnersWrite, http.MethodDelete)
	f.mkHandler(remAccCertPath, updateRemAccountCert, model.PermPartnersWrite, http.MethodPatch)
	f.mkHandler(remAccCertPath, replaceRemAccountCert, model.PermPartnersWrite, http.MethodPut)
	f.mkHandler(remAccAuthPath, authorizeRemoteAccount, model.PermRulesWrite, http.MethodPut)
	f.mkHandler(remAccRevPath, revokeRemoteAccount, model.PermRulesWrite, http.MethodPut)

	// Settings override
	f.mkHandlerNoDB(overrideSettingsAddressesPath, listAddressOverrides, model.PermAdminRead, http.MethodGet)
	f.mkHandlerNoDB(overrideSettingsAddressesPath, addAddressOverride, model.PermAdminWrite,
		http.MethodPost, http.MethodPut, http.MethodPatch)
	f.mkHandlerNoDB(overrideSettingsAddressPath, getAddressOverride, model.PermAdminRead, http.MethodGet)
	f.mkHandlerNoDB(overrideSettingsAddressPath, deleteAddressOverride, model.PermAdminDelete, http.MethodDelete)
}
