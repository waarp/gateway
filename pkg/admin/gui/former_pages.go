package gui

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// The pages of the historic web interface are retired: their addresses lead
// for good (301) to the screens of the reworked interface that replace them,
// so that the bookmarks of the operators keep working. An address that named
// an object (a server, a partner, one of their accounts, a rule) leads to the
// same object on the new screen; to the list when it names none, when the
// object no longer exists, or when the user may not read it, the list then
// answering by itself.

// formerTarget gives the path, under the prefix, of the screen that replaces
// a former page, for the request made to the former address.
type formerTarget func(db database.ReadAccess, r *http.Request) string

// formerOwner is a kind of object a former address named by its identifier
// in a query parameter: its list on the new screen, the right to read it,
// whether an identifier still names one, and whether an account identifier
// names one of its accounts.
type formerOwner struct {
	list    string
	param   string
	perm    model.PermsMask
	found   func(db database.ReadAccess, id int64) bool
	account func(db database.ReadAccess, owner, id int64) bool
}

//nolint:gochecknoglobals // read-only kinds of the former addresses
var (
	formerServer = formerOwner{
		list: "servers", param: "serverID", perm: model.PermServersRead,
		found: func(db database.ReadAccess, id int64) bool {
			_, err := internal.GetServerByID(db, id)

			return err == nil
		},
		account: func(db database.ReadAccess, owner, id int64) bool {
			var account model.LocalAccount

			return db.Get(&account, "id=? AND local_agent_id=?", id, owner).Run() == nil
		},
	}
	formerPartner = formerOwner{
		list: "partners", param: "partnerID", perm: model.PermPartnersRead,
		found: func(db database.ReadAccess, id int64) bool {
			_, err := internal.GetPartnerByID(db, id)

			return err == nil
		},
		account: func(db database.ReadAccess, owner, id int64) bool {
			var account model.RemoteAccount

			return db.Get(&account, "id=? AND remote_agent_id=?", id, owner).Run() == nil
		},
	}
	formerRule = formerOwner{
		list: "rules", param: "ruleID", perm: model.PermRulesRead,
		found: func(db database.ReadAccess, id int64) bool {
			_, err := internal.GetRuleByID(db, id)

			return err == nil
		},
	}
)

// formerPages maps each address of the historic web interface to the screen
// that replaces it. The former tasks page of a rule leads to its tab from
// the rules screen (rules.FormerTasksPage).
//
//nolint:gochecknoglobals // read-only table of the former addresses
var formerPages = map[string]formerTarget{
	"/status_services":                     formerList("services"),
	"/server_management":                   formerList("servers"),
	"/local_client_management":             formerList("clients"),
	"/partner_management":                  formerList("partners"),
	"/transfer_rules_management":           formerList("rules"),
	"/transfer_monitoring":                 formerList("transfers"),
	"/user_management":                     formerList("users"),
	"/cloud_instance_management":           formerList("clouds"),
	"/cryptographic_key_management":        formerList("keys"),
	"/snmp_management":                     formerList("snmp"),
	"/managing_authentication_authorities": formerList("authorities"),
	"/managing_configuration_overrides":    formerList("overrides"),
	"/email_templates_management":          formerList("email-templates"),
	"/smtp_credentials_management":         formerList("smtp"),

	"/server_authentication":         formerServer.tab("auth"),
	"/local_account_management":      formerServer.tab("accounts"),
	"/local_account_authentication":  formerServer.accountTab("auth"),
	"/partner_authentication":        formerPartner.tab("auth"),
	"/remote_account_management":     formerPartner.tab("accounts"),
	"/remote_account_authentication": formerPartner.accountTab("auth"),
	"/management_usage_rights_rules": formerRule.tab("access"),
}

// formerList leads to the list of a new screen.
func formerList(list string) formerTarget {
	return func(database.ReadAccess, *http.Request) string { return list }
}

// tab leads to a tab of the object the former address named.
func (o formerOwner) tab(tab string) formerTarget {
	return func(db database.ReadAccess, r *http.Request) string {
		id, ok := o.named(db, r)
		if !ok {
			return o.list
		}

		return o.list + "/" + strconv.FormatInt(id, 10) + "/" + tab
	}
}

// accountTab leads to a tab of the account the former address named, among
// the accounts of the object it named too; to the accounts of that object
// when the account is not one of them any more.
func (o formerOwner) accountTab(tab string) formerTarget {
	return func(db database.ReadAccess, r *http.Request) string {
		id, ok := o.named(db, r)
		if !ok {
			return o.list
		}

		accounts := o.list + "/" + strconv.FormatInt(id, 10) + "/accounts"

		account, err := strconv.ParseInt(r.URL.Query().Get("accountID"), 10, 64)
		if err != nil || account <= 0 || !o.account(db, id, account) {
			return accounts
		}

		return accounts + "/" + strconv.FormatInt(account, 10) + "/" + tab
	}
}

// named reads the identifier of the object the former address named, when
// it still names one the user may read.
func (o formerOwner) named(db database.ReadAccess, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.URL.Query().Get(o.param), 10, 64)
	if err != nil || id <= 0 || !common.GetUser(r).Permissions.HasPermission(o.perm) {
		return 0, false
	}

	return id, o.found(db, id)
}

// routeFormerPages registers the former addresses of the historic web
// interface, whatever the method: a form of a page left open still posts to
// them.
func routeFormerPages(router *mux.Router, db database.ReadAccess) {
	for path, target := range formerPages {
		router.Handle(path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, pagePath(target(db, r)), http.StatusMovedPermanently)
		}))
	}
}
