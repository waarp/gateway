package gui

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const errorsTranslationFile = "errors.yaml"

// pagePermissions lists the permissions required to use a WebUI page, aligned
// on those of the REST API: "read" for GET requests, "write" for POST requests,
// and "remove" for POST requests carrying one of the "removeFields" form
// values. A zero mask only requires the user to be authenticated.
type pagePermissions struct {
	read, write, remove model.PermsMask
	removeFields        []string
}

func permsFor(read, write, remove model.PermsMask, removeFields ...string) pagePermissions {
	return pagePermissions{read: read, write: write, remove: remove, removeFields: removeFields}
}

//nolint:gochecknoglobals //read-only table of the WebUI pages permissions
var webuiPermissions = map[string]pagePermissions{
	"/status_services": {},

	"/user_management": permsFor(model.PermUsersRead, model.PermUsersWrite,
		model.PermUsersDelete, "deleteUser"),

	"/partner_management": permsFor(model.PermPartnersRead, model.PermPartnersWrite,
		model.PermPartnersDelete, "deletePartner"),
	"/partner_authentication": permsFor(model.PermPartnersRead, model.PermPartnersWrite,
		model.PermPartnersWrite),
	"/remote_account_management": permsFor(model.PermPartnersRead, model.PermPartnersWrite,
		model.PermPartnersDelete, "deleteRemoteAccount"),
	"/remote_account_authentication": permsFor(model.PermPartnersRead, model.PermPartnersWrite,
		model.PermPartnersWrite),

	"/server_management": permsFor(model.PermServersRead, model.PermServersWrite,
		model.PermServersDelete, "deleteServer"),
	"/server_authentication": permsFor(model.PermServersRead, model.PermServersWrite,
		model.PermServersWrite),
	"/local_account_management": permsFor(model.PermServersRead, model.PermServersWrite,
		model.PermServersDelete, "deleteLocalAccount"),
	"/local_account_authentication": permsFor(model.PermServersRead, model.PermServersWrite,
		model.PermServersWrite),
	"/local_client_management": permsFor(model.PermServersRead, model.PermServersWrite,
		model.PermServersDelete, "deleteLocalClient"),

	"/transfer_rules_management": permsFor(model.PermRulesRead, model.PermRulesWrite,
		model.PermRulesDelete, "deleteRule"),
	"/management_usage_rights_rules": permsFor(model.PermRulesRead, model.PermRulesWrite,
		model.PermRulesWrite),
	"/email_templates_management": permsFor(model.PermRulesRead, model.PermRulesWrite,
		model.PermRulesDelete),
	"/smtp_credentials_management": permsFor(model.PermRulesRead, model.PermRulesWrite,
		model.PermRulesDelete),

	"/transfer_monitoring": permsFor(model.PermTransfersRead, model.PermTransfersWrite,
		model.PermTransfersWrite),

	"/cloud_instance_management": permsFor(model.PermAdminRead, model.PermAdminWrite,
		model.PermAdminDelete, "deleteCloudInstance"),
	"/cryptographic_key_management": permsFor(model.PermAdminRead, model.PermAdminWrite,
		model.PermAdminDelete),
	"/snmp_management": permsFor(model.PermAdminRead, model.PermAdminWrite,
		model.PermAdminDelete),
	"/managing_configuration_overrides": permsFor(model.PermAdminRead, model.PermAdminWrite,
		model.PermAdminDelete),

	"/autocompletion/users":                   permsFor(model.PermUsersRead, 0, 0),
	"/autocompletion/partners":                permsFor(model.PermPartnersRead, 0, 0),
	"/autocompletion/credentialPartner":       permsFor(model.PermPartnersRead, 0, 0),
	"/autocompletion/remoteAccount":           permsFor(model.PermPartnersRead, 0, 0),
	"/autocompletion/credentialRemoteAccount": permsFor(model.PermPartnersRead, 0, 0),
	"/autocompletion/servers":                 permsFor(model.PermServersRead, 0, 0),
	"/autocompletion/credentialServer":        permsFor(model.PermServersRead, 0, 0),
	"/autocompletion/localAccount":            permsFor(model.PermServersRead, 0, 0),
	"/autocompletion/credentialLocalAccount":  permsFor(model.PermServersRead, 0, 0),
	"/autocompletion/localClients":            permsFor(model.PermServersRead, 0, 0),
	"/autocompletion/rules":                   permsFor(model.PermRulesRead, 0, 0),
	"/autocompletion/instanceCloud":           permsFor(model.PermAdminRead, 0, 0),
}

// required returns the permission required by the given request.
func (p pagePermissions) required(r *http.Request) model.PermsMask {
	if r.Method != http.MethodPost {
		return p.read
	}

	for _, field := range p.removeFields {
		if r.FormValue(field) != "" {
			return p.remove
		}
	}

	return p.write
}

// withPagePermissions refuses the request with 403 Forbidden when the user
// lacks the permission it requires, before the page handler is called.
func withPagePermissions(perms pagePermissions, next http.HandlerFunc) http.HandlerFunc {
	errorsLocalization := locale.ParseLocalizationFile(errorsTranslationFile)

	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)

		if !user.Permissions.HasPermission(perms.required(r)) {
			text := locale.MakeLocalText(locale.GetLanguage(r), errorsLocalization)
			http.Error(w, text["unauthorized"], http.StatusForbidden)

			return
		}

		next(w, r)
	}
}

// handlePage registers the given WebUI page handler, behind the permissions
// declared for it in webuiPermissions. Registering a page without declared
// permissions is a programming error.
func handlePage(router *mux.Router, path string, handler http.HandlerFunc) *mux.Route {
	perms, ok := webuiPermissions[path]
	if !ok {
		panic(fmt.Sprintf("no permissions declared for the WebUI page %q", path))
	}

	return router.HandleFunc(path, withPagePermissions(perms, handler))
}
