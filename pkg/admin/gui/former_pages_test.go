package gui

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

// formerCase is a request to a former address and where it must lead.
type formerCase struct {
	target string
	perms  model.PermsMask
	want   string
}

// TestFormerPages checks that each address of the historic web interface
// leads for good to the screen that replaces it: to the object it named when
// it still exists and the user may read it, otherwise to the list.
func TestFormerPages(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)

	server := &model.LocalAgent{Name: "srv-sftp", Protocol: "sftp", Address: types.Addr("0.0.0.0", 2222)}
	other := &model.LocalAgent{Name: "srv-ftp", Protocol: "sftp", Address: types.Addr("0.0.0.0", 2223)}
	partner := &model.RemoteAgent{Name: "banque-demo", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10022)}
	elsewhere := &model.RemoteAgent{Name: "assurance-demo", Protocol: "sftp", Address: types.Addr("127.0.0.1", 10023)}
	rule := &model.Rule{Name: "releves", IsSend: true, Path: "/releves"}

	for _, obj := range []database.InsertBean{server, other, partner, elsewhere, rule} {
		require.NoError(t, db.Insert(obj).Run())
	}

	local := &model.LocalAccount{LocalAgentID: server.ID, Login: "filiale-sud"}
	otherLocal := &model.LocalAccount{LocalAgentID: other.ID, Login: "filiale-nord"}
	remote := &model.RemoteAccount{RemoteAgentID: partner.ID, Login: "prod"}
	otherRemote := &model.RemoteAccount{RemoteAgentID: elsewhere.ID, Login: "recette"}

	for _, obj := range []database.InsertBean{local, otherLocal, remote, otherRemote} {
		require.NoError(t, db.Insert(obj).Run())
	}

	router := mux.NewRouter()
	routeFormerPages(router.PathPrefix(Prefix).Subrouter(), db)

	const (
		all     = model.PermAll
		missing = 999999
	)

	cases := map[string][]formerCase{
		"/status_services":                     {{"", all, "/services"}},
		"/server_management":                   {{"", all, "/servers"}},
		"/local_client_management":             {{"", all, "/clients"}},
		"/partner_management":                  {{"", all, "/partners"}},
		"/transfer_rules_management":           {{"", all, "/rules"}},
		"/transfer_monitoring":                 {{"?status=ERROR", all, "/transfers"}},
		"/user_management":                     {{"", all, "/users"}},
		"/cloud_instance_management":           {{"", all, "/clouds"}},
		"/cryptographic_key_management":        {{"", all, "/keys"}},
		"/snmp_management":                     {{"", all, "/snmp"}},
		"/managing_authentication_authorities": {{"", all, "/authorities"}},
		"/managing_configuration_overrides":    {{"", all, "/overrides"}},
		"/email_templates_management":          {{"", all, "/email-templates"}},
		"/smtp_credentials_management":         {{"", all, "/smtp"}},

		"/server_authentication": {
			{fmt.Sprintf("?serverID=%d", server.ID), all, fmt.Sprintf("/servers/%d/auth", server.ID)},
			{"", all, "/servers"},
			{fmt.Sprintf("?serverID=%d", missing), all, "/servers"},
			{"?serverID=srv-sftp", all, "/servers"},
			{fmt.Sprintf("?serverID=%d", server.ID), model.PermPartnersRead, "/servers"},
		},
		"/local_account_management": {
			{fmt.Sprintf("?serverID=%d", server.ID), all, fmt.Sprintf("/servers/%d/accounts", server.ID)},
			{fmt.Sprintf("?serverID=%d", missing), all, "/servers"},
		},
		"/local_account_authentication": {
			{
				fmt.Sprintf("?serverID=%d&accountID=%d", server.ID, local.ID), all,
				fmt.Sprintf("/servers/%d/accounts/%d/auth", server.ID, local.ID),
			},
			{
				fmt.Sprintf("?serverID=%d&accountID=%d", server.ID, otherLocal.ID), all,
				fmt.Sprintf("/servers/%d/accounts", server.ID),
			},
			{fmt.Sprintf("?serverID=%d", server.ID), all, fmt.Sprintf("/servers/%d/accounts", server.ID)},
			{fmt.Sprintf("?serverID=%d&accountID=%d", missing, local.ID), all, "/servers"},
		},
		"/partner_authentication": {
			{fmt.Sprintf("?partnerID=%d", partner.ID), all, fmt.Sprintf("/partners/%d/auth", partner.ID)},
			{fmt.Sprintf("?partnerID=%d", missing), all, "/partners"},
			{fmt.Sprintf("?partnerID=%d", partner.ID), model.PermServersRead, "/partners"},
		},
		"/remote_account_management": {
			{fmt.Sprintf("?partnerID=%d", partner.ID), all, fmt.Sprintf("/partners/%d/accounts", partner.ID)},
			{"", all, "/partners"},
		},
		"/remote_account_authentication": {
			{
				fmt.Sprintf("?partnerID=%d&accountID=%d", partner.ID, remote.ID), all,
				fmt.Sprintf("/partners/%d/accounts/%d/auth", partner.ID, remote.ID),
			},
			{
				fmt.Sprintf("?partnerID=%d&accountID=%d", partner.ID, otherRemote.ID), all,
				fmt.Sprintf("/partners/%d/accounts", partner.ID),
			},
			{fmt.Sprintf("?partnerID=%d&accountID=%d", missing, remote.ID), all, "/partners"},
		},
		"/management_usage_rights_rules": {
			{fmt.Sprintf("?ruleID=%d", rule.ID), all, fmt.Sprintf("/rules/%d/access", rule.ID)},
			{fmt.Sprintf("?ruleID=%d", missing), all, "/rules"},
			{fmt.Sprintf("?ruleID=%d", rule.ID), model.PermServersRead, "/rules"},
		},
	}

	for path := range formerPages {
		assert.Containsf(t, cases, path, "the former address %s has no test", path)
	}

	for path, list := range cases {
		require.Containsf(t, formerPages, path, "%s is not a former address", path)

		for _, c := range list {
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				t.Run(method+" "+path+c.target, func(t *testing.T) {
					t.Parallel()

					user := &model.User{Username: "former", Permissions: c.perms}
					ctx := context.WithValue(context.Background(), ContextUserKey, user)
					req := httptest.NewRequestWithContext(ctx, method, Prefix+path+c.target, nil)
					rec := httptest.NewRecorder()
					router.ServeHTTP(rec, req)

					assert.Equal(t, http.StatusMovedPermanently, rec.Code)
					assert.Equal(t, Prefix+c.want, rec.Header().Get("Location"))
				})
			}
		}
	}
}
