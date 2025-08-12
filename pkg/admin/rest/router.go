package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const Prefix = "/api"

// MakeRESTHandler appends all the REST API handlers to the given HTTP router.
// All routes can be retrieved and modified from the router using their name.
// Routes are named by concatenating their method (in all caps), followed by a
// space, followed by their full path.
func MakeRESTHandler(logger *log.Logger, db *database.DB, router *mux.Router) {
	router.StrictSlash(true)
	router.Use(
		AuthenticationMiddleware(logger, db),
		LoggingMiddleware(logger),
		ServerInfoMiddleware(),
	)

	router.Name("GET /status").Path("/status").Methods(http.MethodGet).
		Handler(getStatus(logger))
	router.Name("GET /about").Path("/about").Methods(http.MethodGet).
		Handler(makeAbout(logger))

	mkHandler := makeHandlerFactory(logger, db, router)

	makeUserHandlers(mkHandler)
	makeRuleHandlers(mkHandler)
	makeTransferHandlers(mkHandler)
	makeServerHandlers(mkHandler)
	makeLocalAccountHandlers(mkHandler)
	makePartnerHandlers(mkHandler)
	makeRemoteAccountHandlers(mkHandler)
	makeClientHandlers(mkHandler)
	makeOverrideHandlers(mkHandler)
	makeCloudHandlers(mkHandler)
	makeAuthoritiesHandlers(mkHandler)
	makeSNMPHandlers(mkHandler)
	makeKeysHandlers(mkHandler)
}

func makeUserHandlers(mkHandler HandlerFactory) {
	const (
		usersPath = "/users"
		userPath  = "/users/{user}"
	)

	mkHandler(usersPath, listUsers, model.PermUsersRead, http.MethodGet)
	mkHandler(usersPath, addUser, model.PermUsersWrite, http.MethodPost)
	mkHandler(userPath, getUser, model.PermUsersRead, http.MethodGet)
	mkHandler(userPath, updateUser, model.PermUsersWrite, http.MethodPatch)
	mkHandler(userPath, replaceUser, model.PermUsersWrite, http.MethodPut)
	mkHandler(userPath, deleteUser, model.PermUsersDelete, http.MethodDelete)
}

func makeRuleHandlers(mkHandler HandlerFactory) {
	const (
		rulesPath     = "/rules"
		rulePath      = "/rules/{rule}/{direction:send|receive}"
		ruleAllowPath = "/rules/{rule}/{direction:send|receive}/allow_all"
	)

	mkHandler(rulesPath, listRules, model.PermRulesRead, http.MethodGet)
	mkHandler(rulesPath, addRule, model.PermRulesWrite, http.MethodPost)
	mkHandler(rulePath, getRule, model.PermRulesRead, http.MethodGet)
	mkHandler(rulePath, updateRule, model.PermRulesWrite, http.MethodPatch)
	mkHandler(rulePath, replaceRule, model.PermRulesWrite, http.MethodPut)
	mkHandler(rulePath, deleteRule, model.PermRulesDelete, http.MethodDelete)
	mkHandler(ruleAllowPath, allowAllRule, model.PermRulesWrite, http.MethodPut)
}

func makeTransferHandlers(mkHandler HandlerFactory) {
	// Transfers
	const (
		transfersPath   = "/transfers"
		transferPath    = "/transfers/{transfer}"
		transPausePath  = "/transfers/{transfer}/pause"
		transResumePath = "/transfers/{transfer}/resume"
		transCancelPath = "/transfers/{transfer}/cancel"
		transRetryPath  = "/transfers/{transfer}/retry"
	)

	mkHandler(transfersPath, listTransfers, model.PermTransfersRead, http.MethodGet)
	mkHandler(transfersPath, addTransfer, model.PermTransfersWrite, http.MethodPost)
	mkHandler(transfersPath, preregisterServerTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(transfersPath, cancelTransfers, model.PermTransfersWrite, http.MethodDelete)
	mkHandler(transferPath, getTransfer, model.PermTransfersRead, http.MethodGet)
	mkHandler(transPausePath, pauseTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(transResumePath, resumeTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(transCancelPath, cancelTransfer, model.PermTransfersWrite, http.MethodPut)
	mkHandler(transRetryPath, retryTransfer, model.PermTransfersWrite, http.MethodPut)

	// History (deprecated)
	const (
		historiesPath = "/history"
		historyPath   = "/history/{history}"
		histRetryPath = "/history/{history}/retry"
	)

	mkHandler(historiesPath, listHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(historyPath, getHistory, model.PermTransfersRead, http.MethodGet)
	mkHandler(histRetryPath, retryHistory, model.PermTransfersWrite, http.MethodPut)
}

//nolint:dupl // best keep servers & partners separate
func makeServerHandlers(mkHandler HandlerFactory) {
	const (
		serversPath         = "/servers"
		serverPath          = "/servers/{server}"
		serverPathEnable    = "/servers/{server}/enable"
		serverPathDisable   = "/servers/{server}/disable"
		serverStartPath     = "/servers/{server}/start"
		serverStopPath      = "/servers/{server}/stop"
		serverRestartPath   = "/servers/{server}/restart"
		serverCertsPath     = "/servers/{server}/certificates"
		serverCertPath      = "/servers/{server}/certificates/{certificate}"
		serverAuthorizePath = "/servers/{server}/authorize/{rule}/{direction:send|receive}"
		serverRevokePath    = "/servers/{server}/revoke/{rule}/{direction:send|receive}"
		serverCredsPath     = "/servers/{server}/credentials"
		serverCredPath      = "/servers/{server}/credentials/{credential}"
	)

	mkHandler(serversPath, listServers, model.PermServersRead, http.MethodGet)
	mkHandler(serversPath, addServer, model.PermServersWrite, http.MethodPost)
	mkHandler(serverPath, getServer, model.PermServersRead, http.MethodGet)
	mkHandler(serverPath, deleteServer, model.PermServersDelete, http.MethodDelete)
	mkHandler(serverPath, updateServer, model.PermServersWrite, http.MethodPatch)
	mkHandler(serverPath, replaceServer, model.PermServersWrite, http.MethodPut)
	mkHandler(serverPathEnable, enableServer, model.PermServersWrite, http.MethodPut)
	mkHandler(serverPathDisable, disableServer, model.PermServersWrite, http.MethodPut)
	mkHandler(serverCertsPath, listServerCerts, model.PermServersRead, http.MethodGet)
	mkHandler(serverCertsPath, addServerCert, model.PermServersWrite, http.MethodPost)
	mkHandler(serverCertPath, getServerCert, model.PermServersRead, http.MethodGet)
	mkHandler(serverCertPath, deleteServerCert, model.PermServersWrite, http.MethodDelete)
	mkHandler(serverCertPath, updateServerCert, model.PermServersWrite, http.MethodPatch)
	mkHandler(serverCertPath, replaceServerCert, model.PermServersWrite, http.MethodPut)
	mkHandler(serverAuthorizePath, authorizeServer, model.PermRulesWrite, http.MethodPut)
	mkHandler(serverRevokePath, revokeServer, model.PermRulesWrite, http.MethodPut)
	mkHandler(serverStartPath, startServer, model.PermServersWrite, http.MethodPut)
	mkHandler(serverStopPath, stopServer, model.PermServersWrite, http.MethodPut)
	mkHandler(serverRestartPath, restartServer, model.PermServersWrite, http.MethodPut)
	mkHandler(serverCredsPath, addServerCred, model.PermServersWrite, http.MethodPost)
	mkHandler(serverCredPath, getServerCred, model.PermServersRead, http.MethodGet)
	mkHandler(serverCredPath, removeServerCred, model.PermServersWrite, http.MethodDelete)
}

//nolint:dupl // best keep local & remote accounts separate
func makeLocalAccountHandlers(mkHandler HandlerFactory) {
	const (
		locAccountsPath     = "/servers/{server}/accounts"
		locAccountPath      = "/servers/{server}/accounts/{local_account}"
		locAccCertsPath     = "/servers/{server}/accounts/{local_account}/certificates"
		locAccCertPath      = "/servers/{server}/accounts/{local_account}/certificates/{certificate}"
		locAccAuthorizePath = "/servers/{server}/accounts/{local_account}/authorize/{rule}/{direction:send|receive}"
		locAccRevokePath    = "/servers/{server}/accounts/{local_account}/revoke/{rule}/{direction:send|receive}"
		locAccCredsPath     = "/servers/{server}/accounts/{local_account}/credentials"
		locAccCredPath      = "/servers/{server}/accounts/{local_account}/credentials/{credential}"
	)

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
	mkHandler(locAccAuthorizePath, authorizeLocalAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(locAccRevokePath, revokeLocalAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(locAccCredsPath, addLocAccCred, model.PermServersWrite, http.MethodPost)
	mkHandler(locAccCredPath, getLocAccCred, model.PermServersRead, http.MethodGet)
	mkHandler(locAccCredPath, removeLocAccCred, model.PermServersWrite, http.MethodDelete)
}

//nolint:dupl // best keep servers & partners separate
func makePartnerHandlers(mkHandler HandlerFactory) {
	const (
		partnersPath         = "/partners"
		partnerPath          = "/partners/{partner}"
		partnerCertsPath     = "/partners/{partner}/certificates"
		partnerCertPath      = "/partners/{partner}/certificates/{certificate}"
		partnerAuthorizePath = "/partners/{partner}/authorize/{rule}/{direction:send|receive}"
		partnerRevokePath    = "/partners/{partner}/revoke/{rule}/{direction:send|receive}"
		partnerCredsPath     = "/partners/{partner}/credentials"
		partnerCredPath      = "/partners/{partner}/credentials/{credential}"
	)

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
	mkHandler(partnerAuthorizePath, authorizePartner, model.PermRulesWrite, http.MethodPut)
	mkHandler(partnerRevokePath, revokePartner, model.PermRulesWrite, http.MethodPut)
	mkHandler(partnerCredsPath, addPartnerCred, model.PermPartnersWrite, http.MethodPost)
	mkHandler(partnerCredPath, getPartnerCred, model.PermPartnersRead, http.MethodGet)
	mkHandler(partnerCredPath, removePartnerCred, model.PermPartnersWrite, http.MethodDelete)
}

//nolint:dupl // best keep local & remote accounts separate
func makeRemoteAccountHandlers(mkHandler HandlerFactory) {
	const (
		remAccountsPath     = "/partners/{partner}/accounts"
		remAccountPath      = "/partners/{partner}/accounts/{remote_account}"
		remAccCertsPath     = "/partners/{partner}/accounts/{remote_account}/certificates"
		remAccCertPath      = "/partners/{partner}/accounts/{remote_account}/certificates/{certificate}"
		remAccAuthorizePath = "/partners/{partner}/accounts/{remote_account}/authorize/{rule}/{direction:send|receive}"
		remAccRevokePath    = "/partners/{partner}/accounts/{remote_account}/revoke/{rule}/{direction:send|receive}"
		remAccCredsPath     = "/partners/{partner}/accounts/{remote_account}/credentials"
		remAccCredPath      = "/partners/{partner}/accounts/{remote_account}/credentials/{credential}"
	)

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
	mkHandler(remAccAuthorizePath, authorizeRemoteAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(remAccRevokePath, revokeRemoteAccount, model.PermRulesWrite, http.MethodPut)
	mkHandler(remAccCredsPath, addRemAccCred, model.PermPartnersWrite, http.MethodPost)
	mkHandler(remAccCredPath, getRemAccCred, model.PermPartnersRead, http.MethodGet)
	mkHandler(remAccCredPath, removeRemAccCred, model.PermPartnersWrite, http.MethodDelete)
}

func makeClientHandlers(mkHandler HandlerFactory) {
	const (
		clientsPath       = "/clients"
		clientPath        = "/clients/{client}"
		clientStartPath   = "/clients/{client}/start"
		clientStopPath    = "/clients/{client}/stop"
		clientRestartPath = "/clients/{client}/restart"
	)

	mkHandler(clientsPath, createClient, model.PermServersWrite, http.MethodPost)
	mkHandler(clientsPath, listClients, model.PermServersRead, http.MethodGet)
	mkHandler(clientPath, getClient, model.PermServersRead, http.MethodGet)
	mkHandler(clientPath, updateClient, model.PermServersWrite, http.MethodPatch)
	mkHandler(clientPath, replaceClient, model.PermServersWrite, http.MethodPut)
	mkHandler(clientPath, deleteClient, model.PermServersWrite, http.MethodDelete)
	mkHandler(clientStartPath, startClient, model.PermServersWrite, http.MethodPut)
	mkHandler(clientStopPath, stopClient, model.PermServersWrite, http.MethodPut)
	mkHandler(clientRestartPath, restartClient, model.PermServersWrite, http.MethodPut)
}

func makeOverrideHandlers(mkHandler HandlerFactory) {
	const (
		overrideSettingsAddressesPath = "/override/addresses"
		overrideSettingsAddressPath   = "/override/addresses/{address}"
	)

	mkHandler.noDB(overrideSettingsAddressesPath, listAddressOverrides, model.PermAdminRead, http.MethodGet)
	mkHandler.noDB(overrideSettingsAddressesPath, addAddressOverride, model.PermAdminWrite,
		http.MethodPost, http.MethodPut, http.MethodPatch)
	mkHandler.noDB(overrideSettingsAddressPath, deleteAddressOverride, model.PermAdminDelete, http.MethodDelete)
}

func makeCloudHandlers(mkHandler HandlerFactory) {
	const (
		cloudInstancesPath = "/clouds"
		cloudInstancePath  = "/clouds/{cloud}"
	)

	mkHandler(cloudInstancesPath, listClouds, model.PermAdminRead, http.MethodGet)
	mkHandler(cloudInstancesPath, addCloud, model.PermAdminWrite, http.MethodPost)
	mkHandler(cloudInstancePath, getCloud, model.PermAdminRead, http.MethodGet)
	mkHandler(cloudInstancePath, updateCloud, model.PermAdminWrite, http.MethodPatch)
	mkHandler(cloudInstancePath, replaceCloud, model.PermAdminWrite, http.MethodPut)
	mkHandler(cloudInstancePath, deleteCloud, model.PermAdminDelete, http.MethodDelete)
}

func makeAuthoritiesHandlers(mkHandler HandlerFactory) {
	const (
		authAuthoritiesPath = "/authorities"
		authAuthorityPath   = "/authorities/{authority}"
	)

	mkHandler(authAuthoritiesPath, addAuthAuthority, model.PermAdminWrite, http.MethodPost)
	mkHandler(authAuthoritiesPath, listAuthAuthorities, model.PermAdminRead, http.MethodGet)
	mkHandler(authAuthorityPath, getAuthAuthority, model.PermAdminRead, http.MethodGet)
	mkHandler(authAuthorityPath, updateAuthAuthority, model.PermAdminWrite, http.MethodPatch)
	mkHandler(authAuthorityPath, replaceAuthAuthority, model.PermAdminWrite, http.MethodPut)
	mkHandler(authAuthorityPath, deleteAuthAuthority, model.PermAdminWrite, http.MethodDelete)
}

func makeSNMPHandlers(mkHandler HandlerFactory) {
	const (
		snmpServerPath   = "/snmp/server"
		snmpMonitorsPath = "/snmp/monitors"
		snmpMonitorPath  = "/snmp/monitors/{snmp_monitor}"
	)

	// SNMP monitors
	mkHandler(snmpMonitorsPath, addSnmpMonitor, model.PermAdminWrite, http.MethodPost)
	mkHandler(snmpMonitorsPath, listSnmpMonitors, model.PermAdminRead, http.MethodGet)
	mkHandler(snmpMonitorPath, getSnmpMonitor, model.PermAdminRead, http.MethodGet)
	mkHandler(snmpMonitorPath, updateSnmpMonitor, model.PermAdminWrite, http.MethodPatch)
	mkHandler(snmpMonitorPath, deleteSnmpMonitor, model.PermAdminDelete, http.MethodDelete)

	// SNMP server
	mkHandler(snmpServerPath, getSnmpService, model.PermAdminRead, http.MethodGet)
	mkHandler(snmpServerPath, setSnmpService, model.PermAdminWrite, http.MethodPut)
	mkHandler(snmpServerPath, deleteSnmpService, model.PermAdminWrite, http.MethodDelete)
}

func makeKeysHandlers(mkHandler HandlerFactory) {
	const (
		cryptoKeysPath = "/keys"
		cryptoKeyPath  = "/keys/{crypto_key}"
	)

	mkHandler(cryptoKeysPath, addCryptoKey, model.PermAdminWrite, http.MethodPost)
	mkHandler(cryptoKeysPath, listCryptoKeys, model.PermAdminRead, http.MethodGet)
	mkHandler(cryptoKeyPath, getCryptoKey, model.PermAdminRead, http.MethodGet)
	mkHandler(cryptoKeyPath, updateCryptoKey, model.PermAdminWrite, http.MethodPatch)
	mkHandler(cryptoKeyPath, deleteCryptoKey, model.PermAdminDelete, http.MethodDelete)
}

func makeEmailHanddlers(mkHandler HandlerFactory) {
	const (
		emailTemplatesPath  = "/email/templates"
		emailTemplatePath   = "/email/templates/{email_template}"
		smtpCredentialsPath = "/email/credentials"
		smtpCredentialPath  = "/email/credentials/{smtp_credential}"
	)

	// Email templates
	mkHandler(emailTemplatesPath, addEmailTemplate, model.PermRulesWrite, http.MethodPost)
	mkHandler(emailTemplatesPath, listEmailTemplates, model.PermRulesRead, http.MethodGet)
	mkHandler(emailTemplatePath, getEmailTemplate, model.PermRulesRead, http.MethodGet)
	mkHandler(emailTemplatePath, updateEmailTemplate, model.PermRulesWrite, http.MethodPatch)
	mkHandler(emailTemplatePath, deleteEmailTemplate, model.PermRulesDelete, http.MethodDelete)

	// SMTP credentials
	mkHandler(smtpCredentialsPath, addSMTPCredential, model.PermRulesWrite, http.MethodPost)
	mkHandler(smtpCredentialsPath, listSMPTCredentials, model.PermRulesRead, http.MethodGet)
	mkHandler(smtpCredentialPath, getSMTPCredential, model.PermRulesRead, http.MethodGet)
	mkHandler(smtpCredentialPath, updateSMTPCredential, model.PermRulesWrite, http.MethodPatch)
	mkHandler(smtpCredentialPath, deleteSMTPCredential, model.PermRulesDelete, http.MethodDelete)

}