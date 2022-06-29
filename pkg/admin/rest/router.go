package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/service"
)

// The definitions of all the REST entry points paths.
const (
	StatusPath = "/api/status"

	UsersPath = "/api/users"
	UserPath  = "/api/users/{user}"

	RulesPath     = "/api/rules"
	RulePath      = "/api/rules/{rule}/{direction:send|receive}"
	RuleAllowPath = "/api/rules/{rule}/{direction:send|receive}/allow_all"

	TransfersPath   = "/api/transfers"
	TransferPath    = "/api/transfers/{transfer}"
	TransPausePath  = "/api/transfers/{transfer}/pause"
	TransResumePath = "/api/transfers/{transfer}/resume"
	TransCancelPath = "/api/transfers/{transfer}/cancel"

	HistoriesPath = "/api/history"
	HistoryPath   = "/api/history/{history}"
	HistRetryPath = "/api/history/{history}/retry"

	ServersPath     = "/api/servers"
	ServerPath      = "/api/servers/{server}"
	ServerCertsPath = "/api/servers/{server}/certificates"
	ServerCertPath  = "/api/servers/{server}/certificates/{certificate}"
	ServerAuthPath  = "/api/servers/{server}/authorize/{rule}/{direction:send|receive}"
	ServerRevPath   = "/api/servers/{server}/revoke/{rule}/{direction:send|receive}"

	LocAccountsPath = "/api/servers/{server}/accounts"
	LocAccountPath  = "/api/servers/{server}/accounts/{local_account}"
	LocAccCertsPath = "/api/servers/{server}/accounts/{local_account}/certificates"
	LocAccCertPath  = "/api/servers/{server}/accounts/{local_account}/certificates/{certificate}"
	LocAccAuthPath  = "/api/servers/{server}/accounts/{local_account}/authorize/{rule}/{direction:send|receive}"
	LocAccRevPath   = "/api/servers/{server}/accounts/{local_account}/revoke/{rule}/{direction:send|receive}"

	PartnersPath     = "/api/partners"
	PartnerPath      = "/api/partners/{partner}"
	PartnerCertsPath = "/api/partners/{partner}/certificates"
	PartnerCertPath  = "/api/partners/{partner}/certificates/{certificate}"
	PartnerAuthPath  = "/api/partners/{partner}/authorize/{rule}/{direction:send|receive}"
	PartnerRevPath   = "/api/partners/{partner}/revoke/{rule}/{direction:send|receive}"

	RemAccountsPath = "/api/partners/{partner}/accounts"
	RemAccountPath  = "/api/partners/{partner}/accounts/{remote_account}"
	RemAccCertsPath = "/api/partners/{partner}/accounts/{remote_account}/certificates"
	RemAccCertPath  = "/api/partners/{partner}/accounts/{remote_account}/certificates/{certificate}"
	RemAccAuthPath  = "/api/partners/{partner}/accounts/{remote_account}/authorize/{rule}/{direction:send|receive}"
	RemAccRevPath   = "/api/partners/{partner}/accounts/{remote_account}/revoke/{rule}/{direction:send|receive}"

	OverrideSettingsAddressesPath = "/api/override/addresses"
	OverrideSettingsAddressPath   = "/api/override/addresses/{address}"
)

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
// All routes can be retrieved and modified from the router using their name.
// Routes are named by concatenating their method (in all caps), followed by a
// space, followed by their full path.
//nolint:funlen // hard to shorten
func MakeRESTHandler(logger *log.Logger, db *database.DB, router *mux.Router,
	coreServices map[string]service.Service, protoServices map[string]service.ProtoService,
) {
	router.StrictSlash(true)

	router.Name("GET " + StatusPath).Path(StatusPath).Methods(http.MethodGet).
		Handler(getStatus(logger, coreServices, protoServices))

	mkHandler := makeHandlerFactory(logger, db, router)

	// Users
	mkHandler(UsersPath, listUsers, model.PermUsersRead, http.MethodGet)
	mkHandler(UsersPath, addUser, model.PermUsersWrite, http.MethodPost)
	mkHandler(UserPath, getUser, model.PermUsersRead, http.MethodGet)
	mkHandler(UserPath, updateUser, model.PermUsersWrite, http.MethodPatch)
	mkHandler(UserPath, replaceUser, model.PermUsersWrite, http.MethodPut)
	mkHandler(UserPath, deleteUser, model.PermUsersDelete, http.MethodDelete)

	// Rules
	mkHandler(RulesPath, listRules, model.PermRulesRead, http.MethodGet)
	mkHandler(RulesPath, addRule, model.PermRulesWrite, http.MethodPost)
	mkHandler(RulePath, getRule, model.PermRulesRead, http.MethodGet)
	mkHandler(RulePath, updateRule, model.PermRulesWrite, http.MethodPatch)
	mkHandler(RulePath, replaceRule, model.PermRulesWrite, http.MethodPut)
	mkHandler(RulePath, deleteRule, model.PermRulesDelete, http.MethodDelete)
	mkHandler(RuleAllowPath, allowAllRule, model.PermRulesWrite, http.MethodPut)

	// Transfers
	mkHandler(TransfersPath, listTransfers, model.PermTransfersRead, http.MethodGet)
	mkHandler(TransfersPath, addTransfer, model.PermTransfersWrite, http.MethodPost)
	mkHandler(TransferPath, getTransfer, model.PermTransfersRead, http.MethodGet)
	mkHandler(TransPausePath, pauseTransfer(protoServices), model.PermTransfersWrite, http.MethodPut)
	mkHandler(TransResumePath, resumeTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(TransCancelPath, cancelTransfer(protoServices), model.PermTransfersWrite, http.MethodPut)

	// History
	mkHandler(HistoriesPath, listHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(HistoryPath, getHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(HistRetryPath, retryTransfer, model.PermTransfersWrite, http.MethodPut)

	// Servers
	mkHandler(ServersPath, listServers, model.PermServersRead, http.MethodGet)
	mkHandler(ServersPath, addServer, model.PermServersWrite, http.MethodPost)
	mkHandler(ServerPath, getServer, model.PermServersRead, http.MethodGet)
	mkHandler(ServerPath, deleteServer, model.PermServersDelete, http.MethodDelete)
	mkHandler(ServerPath, updateServer, model.PermServersWrite, http.MethodPatch)
	mkHandler(ServerPath, replaceServer, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerCertsPath, listServerCerts, model.PermServersRead, http.MethodGet)
	mkHandler(ServerCertsPath, addServerCert, model.PermServersWrite, http.MethodPost)
	mkHandler(ServerCertPath, getServerCert, model.PermServersRead, http.MethodGet)
	mkHandler(ServerCertPath, deleteServerCert, model.PermServersWrite, http.MethodDelete)
	mkHandler(ServerCertPath, updateServerCert, model.PermServersWrite, http.MethodPatch)
	mkHandler(ServerCertPath, replaceServerCert, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerAuthPath, authorizeServer, model.PermRulesWrite, http.MethodPut)
	mkHandler(ServerRevPath, revokeServer, model.PermRulesWrite, http.MethodPut)

	// Local accounts
	mkHandler(LocAccountsPath, listLocalAccounts, model.PermServersRead, http.MethodGet)
	mkHandler(LocAccountsPath, addLocalAccount, model.PermServersWrite, http.MethodPost)
	mkHandler(LocAccountPath, getLocalAccount, model.PermServersRead, http.MethodGet)
	mkHandler(LocAccountPath, deleteLocalAccount, model.PermServersDelete, http.MethodDelete)
	mkHandler(LocAccountPath, updateLocalAccount, model.PermServersWrite, http.MethodPatch)
	mkHandler(LocAccountPath, replaceLocalAccount, model.PermServersWrite, http.MethodPut)
	mkHandler(LocAccCertsPath, listLocAccountCerts, model.PermServersRead, http.MethodGet)
	mkHandler(LocAccCertsPath, addLocAccountCert, model.PermServersWrite, http.MethodPost)
	mkHandler(LocAccCertPath, getLocAccountCert, model.PermServersRead, http.MethodGet)
	mkHandler(LocAccCertPath, deleteLocAccountCert, model.PermServersWrite, http.MethodDelete)
	mkHandler(LocAccCertPath, updateLocAccountCert, model.PermServersWrite, http.MethodPatch)
	mkHandler(LocAccCertPath, replaceLocAccountCert, model.PermServersWrite, http.MethodPut)
	mkHandler(LocAccAuthPath, authorizeLocalAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(LocAccRevPath, revokeLocalAccount, model.PermRulesWrite, http.MethodPut)

	// Partners
	mkHandler(PartnersPath, listPartners, model.PermPartnersRead, http.MethodGet)
	mkHandler(PartnersPath, addPartner, model.PermPartnersWrite, http.MethodPost)
	mkHandler(PartnerPath, getPartner, model.PermPartnersRead, http.MethodGet)
	mkHandler(PartnerPath, deletePartner, model.PermPartnersDelete, http.MethodDelete)
	mkHandler(PartnerPath, updatePartner, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(PartnerPath, replacePartner, model.PermPartnersWrite, http.MethodPut)
	mkHandler(PartnerCertsPath, listPartnerCerts, model.PermPartnersRead, http.MethodGet)
	mkHandler(PartnerCertsPath, addPartnerCert, model.PermPartnersWrite, http.MethodPost)
	mkHandler(PartnerCertPath, getPartnerCert, model.PermPartnersRead, http.MethodGet)
	mkHandler(PartnerCertPath, deletePartnerCert, model.PermPartnersWrite, http.MethodDelete)
	mkHandler(PartnerCertPath, updatePartnerCert, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(PartnerCertPath, replacePartnerCert, model.PermPartnersWrite, http.MethodPut)
	mkHandler(PartnerAuthPath, authorizePartner, model.PermRulesWrite, http.MethodPut)
	mkHandler(PartnerRevPath, revokePartner, model.PermRulesWrite, http.MethodPut)

	// Remote accounts
	mkHandler(RemAccountsPath, listRemoteAccounts, model.PermPartnersRead, http.MethodGet)
	mkHandler(RemAccountsPath, addRemoteAccount, model.PermPartnersWrite, http.MethodPost)
	mkHandler(RemAccountPath, getRemoteAccount, model.PermPartnersRead, http.MethodGet)
	mkHandler(RemAccountPath, deleteRemoteAccount, model.PermPartnersDelete, http.MethodDelete)
	mkHandler(RemAccountPath, updateRemoteAccount, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(RemAccountPath, replaceRemoteAccount, model.PermPartnersWrite, http.MethodPut)
	mkHandler(RemAccCertsPath, listRemAccountCerts, model.PermPartnersRead, http.MethodGet)
	mkHandler(RemAccCertsPath, addRemAccountCert, model.PermPartnersWrite, http.MethodPost)
	mkHandler(RemAccCertPath, getRemAccountCert, model.PermPartnersRead, http.MethodGet)
	mkHandler(RemAccCertPath, deleteRemAccountCert, model.PermPartnersWrite, http.MethodDelete)
	mkHandler(RemAccCertPath, updateRemAccountCert, model.PermPartnersWrite, http.MethodPatch)
	mkHandler(RemAccCertPath, replaceRemAccountCert, model.PermPartnersWrite, http.MethodPut)
	mkHandler(RemAccAuthPath, authorizeRemoteAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(RemAccRevPath, revokeRemoteAccount, model.PermRulesWrite, http.MethodPut)

	// Settings override
	mkHandler.noDB(OverrideSettingsAddressesPath, listAddressOverrides, model.PermAdminRead, http.MethodGet)
	mkHandler.noDB(OverrideSettingsAddressesPath, addAddressOverride, model.PermAdminWrite,
		http.MethodPost, http.MethodPut, http.MethodPatch)
	mkHandler.noDB(OverrideSettingsAddressPath, deleteAddressOverride, model.PermAdminDelete, http.MethodDelete)
}
