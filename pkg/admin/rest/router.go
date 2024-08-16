package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The definitions of all the REST entry points paths.
//
//nolint:lll,gosec //paths are long; credentials are not hardcoded
const (
	AboutPath  = "/api/about"
	StatusPath = "/api/status" // Deprecated: replaced by AboutPath

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
	TransRetryPath  = "/api/transfers/{transfer}/retry"

	HistoriesPath = "/api/history"                 // Deprecated: merged with transfers
	HistoryPath   = "/api/history/{history}"       // Deprecated: merged with transfers
	HistRetryPath = "/api/history/{history}/retry" // Deprecated: merged with transfers

	ServersPath         = "/api/servers"
	ServerPath          = "/api/servers/{server}"
	ServerPathEnable    = "/api/servers/{server}/enable"
	ServerPathDisable   = "/api/servers/{server}/disable"
	ServerStartPath     = "/api/servers/{server}/start"
	ServerStopPath      = "/api/servers/{server}/stop"
	ServerRestartPath   = "/api/servers/{server}/restart"
	ServerCertsPath     = "/api/servers/{server}/certificates"               // Deprecated: replaced by credentials
	ServerCertPath      = "/api/servers/{server}/certificates/{certificate}" // Deprecated: replaced by credentials
	ServerAuthorizePath = "/api/servers/{server}/authorize/{rule}/{direction:send|receive}"
	ServerRevokePath    = "/api/servers/{server}/revoke/{rule}/{direction:send|receive}"
	ServerCredsPath     = "/api/servers/{server}/credentials"
	ServerCredPath      = "/api/servers/{server}/credentials/{credential}"

	LocAccountsPath     = "/api/servers/{server}/accounts"
	LocAccountPath      = "/api/servers/{server}/accounts/{local_account}"
	LocAccCertsPath     = "/api/servers/{server}/accounts/{local_account}/certificates"               // Deprecated: replaced by credentials
	LocAccCertPath      = "/api/servers/{server}/accounts/{local_account}/certificates/{certificate}" // Deprecated: replaced by credentials
	LocAccAuthorizePath = "/api/servers/{server}/accounts/{local_account}/authorize/{rule}/{direction:send|receive}"
	LocAccRevokePath    = "/api/servers/{server}/accounts/{local_account}/revoke/{rule}/{direction:send|receive}"
	LocAccCredsPath     = "/api/servers/{server}/accounts/{local_account}/credentials"
	LocAccCredPath      = "/api/servers/{server}/accounts/{local_account}/credentials/{credential}"

	PartnersPath         = "/api/partners"
	PartnerPath          = "/api/partners/{partner}"
	PartnerCertsPath     = "/api/partners/{partner}/certificates"               // Deprecated: replaced by credentials
	PartnerCertPath      = "/api/partners/{partner}/certificates/{certificate}" // Deprecated: replaced by credentials
	PartnerAuthorizePath = "/api/partners/{partner}/authorize/{rule}/{direction:send|receive}"
	PartnerRevokePath    = "/api/partners/{partner}/revoke/{rule}/{direction:send|receive}"
	PartnerCredsPath     = "/api/partners/{partner}/credentials"
	PartnerCredPath      = "/api/partners/{partner}/credentials/{credential}"

	RemAccountsPath     = "/api/partners/{partner}/accounts"
	RemAccountPath      = "/api/partners/{partner}/accounts/{remote_account}"
	RemAccCertsPath     = "/api/partners/{partner}/accounts/{remote_account}/certificates"               // Deprecated: replaced by credentials
	RemAccCertPath      = "/api/partners/{partner}/accounts/{remote_account}/certificates/{certificate}" // Deprecated: replaced by credentials
	RemAccAuthorizePath = "/api/partners/{partner}/accounts/{remote_account}/authorize/{rule}/{direction:send|receive}"
	RemAccRevokePath    = "/api/partners/{partner}/accounts/{remote_account}/revoke/{rule}/{direction:send|receive}"
	RemAccCredsPath     = "/api/partners/{partner}/accounts/{remote_account}/credentials"
	RemAccCredPath      = "/api/partners/{partner}/accounts/{remote_account}/credentials/{credential}"

	ClientsPath       = "/api/clients"
	ClientPath        = "/api/clients/{client}"
	ClientStartPath   = "/api/clients/{client}/start"
	ClientStopPath    = "/api/clients/{client}/stop"
	ClientRestartPath = "/api/clients/{client}/restart"

	OverrideSettingsAddressesPath = "/api/override/addresses"
	OverrideSettingsAddressPath   = "/api/override/addresses/{address}"

	CloudInstancesPath = "/api/clouds"
	CloudInstancePath  = "/api/clouds/{cloud}"

	AuthAuthoritiesPath = "/api/authorities"
	AuthAuthorityPath   = "/api/authorities/{authority}"

	SNMPPath         = "/api/snmp"
	SNMPServerPath   = "/api/snmp/server"
	SNMPMonitorsPath = "/api/snmp/monitors"
	SNMPMonitorPath  = "/api/snmp/monitors/{snmp_monitor}"
)

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
// All routes can be retrieved and modified from the router using their name.
// Routes are named by concatenating their method (in all caps), followed by a
// space, followed by their full path.
//
//nolint:funlen // hard to shorten
func MakeRESTHandler(logger *log.Logger, db *database.DB, router *mux.Router,
) {
	router.StrictSlash(true)

	router.Name("GET " + StatusPath).Path(StatusPath).Methods(http.MethodGet).
		Handler(getStatus(logger))
	router.Name("GET " + AboutPath).Path(AboutPath).Methods(http.MethodGet).
		Handler(makeAbout(logger))

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
	mkHandler(TransfersPath, cancelTransfers, model.PermTransfersWrite, http.MethodDelete)
	mkHandler(TransferPath, getTransfer, model.PermTransfersRead, http.MethodGet)
	mkHandler(TransPausePath, pauseTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(TransResumePath, resumeTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(TransCancelPath, cancelTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(TransRetryPath, retryTransfer, model.PermTransfersWrite, http.MethodPut)

	// History
	mkHandler(HistoriesPath, listHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(HistoryPath, getHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(HistRetryPath, retryHistory, model.PermTransfersWrite, http.MethodPut)

	// Servers
	mkHandler(ServersPath, listServers, model.PermServersRead, http.MethodGet)
	mkHandler(ServersPath, addServer, model.PermServersWrite, http.MethodPost)
	mkHandler(ServerPath, getServer, model.PermServersRead, http.MethodGet)
	mkHandler(ServerPath, deleteServer, model.PermServersDelete, http.MethodDelete)
	mkHandler(ServerPath, updateServer, model.PermServersWrite, http.MethodPatch)
	mkHandler(ServerPath, replaceServer, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerPathEnable, enableServer, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerPathDisable, disableServer, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerCertsPath, listServerCerts, model.PermServersRead, http.MethodGet)
	mkHandler(ServerCertsPath, addServerCert, model.PermServersWrite, http.MethodPost)
	mkHandler(ServerCertPath, getServerCert, model.PermServersRead, http.MethodGet)
	mkHandler(ServerCertPath, deleteServerCert, model.PermServersWrite, http.MethodDelete)
	mkHandler(ServerCertPath, updateServerCert, model.PermServersWrite, http.MethodPatch)
	mkHandler(ServerCertPath, replaceServerCert, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerAuthorizePath, authorizeServer, model.PermRulesWrite, http.MethodPut)
	mkHandler(ServerRevokePath, revokeServer, model.PermRulesWrite, http.MethodPut)
	mkHandler(ServerStartPath, startServer, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerStopPath, stopServer, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerRestartPath, restartServer, model.PermServersWrite, http.MethodPut)
	mkHandler(ServerCredsPath, addServerCred, model.PermServersWrite, http.MethodPost)
	mkHandler(ServerCredPath, getServerCred, model.PermServersRead, http.MethodGet)
	mkHandler(ServerCredPath, removeServerCred, model.PermServersWrite, http.MethodDelete)

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
	mkHandler(LocAccAuthorizePath, authorizeLocalAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(LocAccRevokePath, revokeLocalAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(LocAccCredsPath, addLocAccCred, model.PermServersWrite, http.MethodPost)
	mkHandler(LocAccCredPath, getLocAccCred, model.PermServersRead, http.MethodGet)
	mkHandler(LocAccCredPath, removeLocAccCred, model.PermServersWrite, http.MethodDelete)

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
	mkHandler(PartnerAuthorizePath, authorizePartner, model.PermRulesWrite, http.MethodPut)
	mkHandler(PartnerRevokePath, revokePartner, model.PermRulesWrite, http.MethodPut)
	mkHandler(PartnerCredsPath, addPartnerCred, model.PermPartnersWrite, http.MethodPost)
	mkHandler(PartnerCredPath, getPartnerCred, model.PermPartnersRead, http.MethodGet)
	mkHandler(PartnerCredPath, removePartnerCred, model.PermPartnersWrite, http.MethodDelete)

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
	mkHandler(RemAccAuthorizePath, authorizeRemoteAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(RemAccRevokePath, revokeRemoteAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(RemAccCredsPath, addRemAccCred, model.PermPartnersWrite, http.MethodPost)
	mkHandler(RemAccCredPath, getRemAccCred, model.PermPartnersRead, http.MethodGet)
	mkHandler(RemAccCredPath, removeRemAccCred, model.PermPartnersWrite, http.MethodDelete)

	// Clients
	mkHandler(ClientsPath, createClient, model.PermServersWrite, http.MethodPost)
	mkHandler(ClientsPath, listClients, model.PermServersRead, http.MethodGet)
	mkHandler(ClientPath, getClient, model.PermServersRead, http.MethodGet)
	mkHandler(ClientPath, updateClient, model.PermServersWrite, http.MethodPatch)
	mkHandler(ClientPath, replaceClient, model.PermServersWrite, http.MethodPut)
	mkHandler(ClientPath, deleteClient, model.PermServersWrite, http.MethodDelete)
	mkHandler(ClientStartPath, startClient, model.PermServersWrite, http.MethodPut)
	mkHandler(ClientStopPath, stopClient, model.PermServersWrite, http.MethodPut)
	mkHandler(ClientRestartPath, restartClient, model.PermServersWrite, http.MethodPut)

	// Settings override
	mkHandler.noDB(OverrideSettingsAddressesPath, listAddressOverrides, model.PermAdminRead, http.MethodGet)
	mkHandler.noDB(OverrideSettingsAddressesPath, addAddressOverride, model.PermAdminWrite,
		http.MethodPost, http.MethodPut, http.MethodPatch)
	mkHandler.noDB(OverrideSettingsAddressPath, deleteAddressOverride, model.PermAdminDelete, http.MethodDelete)

	// Cloud instances
	mkHandler(CloudInstancesPath, listClouds, model.PermAdminRead, http.MethodGet)
	mkHandler(CloudInstancesPath, addCloud, model.PermAdminWrite, http.MethodPost)
	mkHandler(CloudInstancePath, getCloud, model.PermAdminRead, http.MethodGet)
	mkHandler(CloudInstancePath, updateCloud, model.PermAdminWrite, http.MethodPatch)
	mkHandler(CloudInstancePath, replaceCloud, model.PermAdminWrite, http.MethodPut)
	mkHandler(CloudInstancePath, deleteCloud, model.PermAdminDelete, http.MethodDelete)

	// Authentication authority
	mkHandler(AuthAuthoritiesPath, addAuthAuthority, model.PermAdminWrite, http.MethodPost)
	mkHandler(AuthAuthoritiesPath, listAuthAuthorities, model.PermAdminRead, http.MethodGet)
	mkHandler(AuthAuthorityPath, getAuthAuthority, model.PermAdminRead, http.MethodGet)
	mkHandler(AuthAuthorityPath, updateAuthAuthority, model.PermAdminWrite, http.MethodPatch)
	mkHandler(AuthAuthorityPath, replaceAuthAuthority, model.PermAdminWrite, http.MethodPut)
	mkHandler(AuthAuthorityPath, deleteAuthAuthority, model.PermAdminWrite, http.MethodDelete)

	// SNMP monitors
	mkHandler(SNMPMonitorsPath, addSnmpMonitor, model.PermAdminWrite, http.MethodPost)
	mkHandler(SNMPMonitorsPath, listSnmpMonitors, model.PermAdminRead, http.MethodGet)
	mkHandler(SNMPMonitorPath, getSnmpMonitor, model.PermAdminRead, http.MethodGet)
	mkHandler(SNMPMonitorPath, updateSnmpMonitor, model.PermAdminWrite, http.MethodPatch)
	mkHandler(SNMPMonitorPath, deleteSnmpMonitor, model.PermAdminDelete, http.MethodDelete)

	// SNMP server
	mkHandler(SNMPServerPath, getSnmpService, model.PermAdminRead, http.MethodGet)
	mkHandler(SNMPServerPath, setSnmpService, model.PermAdminWrite, http.MethodPut)
	mkHandler(SNMPServerPath, deleteSnmpService, model.PermAdminWrite, http.MethodDelete)
}
