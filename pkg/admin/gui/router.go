// Package gui contains the HTTP handlers for the gateway's webUI.
package gui

import (
	"context"
	"io/fs"
	"net/http"
	"net/url"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const Prefix = constants.WebuiPrefix

//nolint:gochecknoglobals // global
const (
	ContextUserKey     = constants.ContextUserKey
	ContextLanguageKey = constants.ContextLanguageKey
)

func LanguageMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userLanguage := changeLanguage(w, r)
		ctx := context.WithValue(r.Context(), ContextLanguageKey, userLanguage)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RouterAutocompletions(secureRouter *mux.Router, db *database.DB) {
	secureRouter.HandleFunc("/autocompletion/users", autocompletionFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/partners", autocompletionPartnersFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/credentialPartner", autocompletionCredentialsPartnersFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/remoteAccount", autocompletionRemoteAccountFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/credentialRemoteAccount",
		autocompletionCredentialsRemoteAccountsFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/servers", autocompletionServersFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/credentialServer", autocompletionCredentialsServersFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/localAccount", autocompletionLocalAccountFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/credentialLocalAccount",
		autocompletionCredentialsLocalAccountsFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/localClients", autocompletionLocalClientsFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/rules", autocompletionRulesFunc(db)).Methods("GET")
	secureRouter.HandleFunc("/autocompletion/instanceCloud", autocompletionCloudFunc(db)).Methods("GET")
}

func RouterPages(secureRouter *mux.Router, db *database.DB, logger *log.Logger) {
	secureRouter.HandleFunc("/home", homePage(logger)).Methods("GET")
	secureRouter.HandleFunc("/user_management", userManagementPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/partner_management", partnerManagementPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/partner_authentication", partnerAuthenticationPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/remote_account_management", remoteAccountPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/remote_account_authentication",
		remoteAccountAuthenticationPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/server_management", serverManagementPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/server_authentication", serverAuthenticationPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/local_account_management", localAccountPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/local_account_authentication",
		localAccountAuthenticationPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/local_client_management", localClientManagementPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/transfer_rules_management", ruleManagementPage(logger, db)).Methods("GET", "POST")
	// secureRouter.HandleFunc("/tasks_transfer_rules", tasksTransferRulesPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/management_usage_rights_rules",
		managementUsageRightsRulesPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/transfer_monitoring", transferMonitoringPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/status_services", statusServicesPage(logger)).Methods("GET", "POST")
	secureRouter.HandleFunc("/cloud_instance_management", cloudInstanceManagementPage(logger, db)).Methods("GET", "POST")
	secureRouter.HandleFunc("/cryptographic_key_management",
		cryptographicKeyManagementPage(logger)).Methods("GET", "POST")
	secureRouter.HandleFunc("/snmp_management", snmpManagementPage(logger)).Methods("GET", "POST")
	secureRouter.HandleFunc("/managing_authentication_authorities",
		managingAuthenticationAuthoritiesPage(logger)).Methods("GET", "POST")
	secureRouter.HandleFunc("/managing_configuration_overrides",
		managingConfigurationOverridesPage(logger)).Methods("GET", "POST")
	secureRouter.HandleFunc("/email_templates_management", EmailTemplateManagementPage(logger)).Methods("GET", "POST")
	secureRouter.HandleFunc("/smtp_credentials_management",
		SMTPCredentialManagementPage(logger)).Methods("GET", "POST")

	backend.MakeRouter(secureRouter, db, logger)
}

func AddGUIRouter(router *mux.Router, logger *log.Logger, db *database.DB) {
	router.StrictSlash(true)
	router.Use(LanguageMiddleware)
	// Add HTTP handlers to the router here.
	// Example:
	router.HandleFunc("/login", loginPage(logger, db)).Methods("GET", "POST")
	router.HandleFunc("/logout", logout()).Methods("GET")

	subFS, err := fs.Sub(webFS, "front-end")
	if err != nil {
		logger.Errorf("error accessing css file: %v", err)

		return
	}

	router.PathPrefix("/static/").Handler(http.StripPrefix(Prefix+"/static/", http.FileServerFS(subFS)))

	secureRouter := router.PathPrefix("/").Subrouter()
	secureRouter.Use(AuthenticationMiddleware(logger, db))
	secureRouter.StrictSlash(true)

	RouterAutocompletions(secureRouter, db)

	secureRouter.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "home", http.StatusFound)
	})

	RouterPages(secureRouter, db, logger)
}

func logout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(tokenCookieName); err == nil {
			if _, err = parseToken(cookie.Value); err == nil {
				invalidateToken(cookie.Value)
			}
		}

		http.SetCookie(w, &http.Cookie{
			Name:     tokenCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
		})

		http.Redirect(w, r, "login", http.StatusFound)
	}
}

func AuthenticationMiddleware(logger *log.Logger, db *database.DB) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			loginURL := url.URL{Path: "login"}
			if curPage := r.URL.RequestURI(); curPage != Prefix+"/" {
				params := loginURL.Query()
				params.Set("redirect", curPage)
				loginURL.RawQuery = params.Encode()
			}

			token, err := r.Cookie(tokenCookieName)
			if err != nil || token.Value == "" {
				http.Redirect(w, r, loginURL.String(), http.StatusFound)

				return
			}

			user, err := ValidateSession(db, token.Value)
			if err != nil {
				logger.Infof("Failed to validate session: %v", err)
				http.Redirect(w, r, loginURL.String(), http.StatusFound)

				return
			}

			if r.URL.Query().Get("partial") == "" {
				RefreshExpirationToken(logger, w, r, user.ID)
			}

			ctx := context.WithValue(r.Context(), ContextUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
