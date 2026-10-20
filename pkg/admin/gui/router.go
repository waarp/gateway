// Package gui contains the HTTP handlers for the gateway's webUI.
package gui

import (
	"context"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
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
	handlePage(secureRouter, "/autocompletion/users", autocompletionFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/partners", autocompletionPartnersFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/credentialPartner", autocompletionCredentialsPartnersFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/remoteAccount", autocompletionRemoteAccountFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/credentialRemoteAccount",
		autocompletionCredentialsRemoteAccountsFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/servers", autocompletionServersFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/credentialServer", autocompletionCredentialsServersFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/localAccount", autocompletionLocalAccountFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/credentialLocalAccount",
		autocompletionCredentialsLocalAccountsFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/localClients", autocompletionLocalClientsFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/rules", autocompletionRulesFunc(db)).Methods("GET")
	handlePage(secureRouter, "/autocompletion/instanceCloud", autocompletionCloudFunc(db)).Methods("GET")
}

func RouterPages(secureRouter *mux.Router, db *database.DB, logger *log.Logger) {
	handlePage(secureRouter, "/home", homePage(logger)).Methods("GET")
	handlePage(secureRouter, "/user_management", userManagementPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/partner_management", partnerManagementPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/partner_authentication", partnerAuthenticationPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/remote_account_management", remoteAccountPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/remote_account_authentication",
		remoteAccountAuthenticationPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/server_management", serverManagementPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/server_authentication", serverAuthenticationPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/local_account_management", localAccountPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/local_account_authentication",
		localAccountAuthenticationPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/local_client_management", localClientManagementPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/transfer_rules_management", ruleManagementPage(logger, db)).Methods("GET", "POST")
	// secureRouter.HandleFunc("/tasks_transfer_rules", tasksTransferRulesPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/management_usage_rights_rules",
		managementUsageRightsRulesPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/transfer_monitoring", transferMonitoringPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/status_services", statusServicesPage(logger)).Methods("GET", "POST")
	handlePage(secureRouter, "/cloud_instance_management", cloudInstanceManagementPage(logger, db)).Methods("GET", "POST")
	handlePage(secureRouter, "/cryptographic_key_management",
		cryptographicKeyManagementPage(logger)).Methods("GET", "POST")
	handlePage(secureRouter, "/snmp_management", snmpManagementPage(logger)).Methods("GET", "POST")
	handlePage(secureRouter, "/managing_authentication_authorities",
		managingAuthenticationAuthoritiesPage(logger)).Methods("GET", "POST")
	handlePage(secureRouter, "/managing_configuration_overrides",
		managingConfigurationOverridesPage(logger)).Methods("GET", "POST")
	handlePage(secureRouter, "/email_templates_management", EmailTemplateManagementPage(logger)).Methods("GET", "POST")
	handlePage(secureRouter, "/smtp_credentials_management",
		SMTPCredentialManagementPage(logger)).Methods("GET", "POST")

	backend.MakeRouter(secureRouter, db, logger)
}

func AddGUIRouter(router *mux.Router, logger *log.Logger, db *database.DB) {
	router.StrictSlash(true)
	router.Use(LanguageMiddleware)
	// Add HTTP handlers to the router here.
	// Example:
	router.HandleFunc("/login", loginPage(logger, db)).Methods("GET", "POST")
	// The kit chrome signs out with a POST form; the historic pages link the
	// GET route.
	router.HandleFunc("/logout", logout()).Methods("GET", "POST")

	subFS, err := fs.Sub(webFS, "front-end")
	if err != nil {
		logger.Errorf("error accessing css file: %v", err)

		return
	}

	// The kit files live under the historic static prefix: their route must
	// come first, mux keeping the first prefix that matches.
	backend.MakeKitStaticRouter(router)
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
			invalidateToken(cookie.Value)
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

			tokenCookie, err := r.Cookie(tokenCookieName)
			if err != nil || tokenCookie.Value == "" {
				http.Redirect(w, r, loginURL.String(), http.StatusFound)

				return
			}

			user, err := validateSession(db, tokenCookie.Value)
			if err != nil {
				logger.Infof("Failed to validate session: %v", err)
				http.Redirect(w, r, loginURL.String(), http.StatusFound)

				return
			}

			// A refresh made by a page on its own (historic "partial", v2
			// "fragment") is not a user action: it does not extend the session.
			if query := r.URL.Query(); query.Get("partial") == "" && query.Get(constants.FragmentParam) == "" {
				refreshToken(r, w, tokenCookie.Value)
			}

			ctx := context.WithValue(r.Context(), ContextUserKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
