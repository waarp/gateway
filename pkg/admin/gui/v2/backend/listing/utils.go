package listing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func encodeList[S ~[]E, E any](w http.ResponseWriter, logger *log.Logger,
	s S, f func(E) string,
) {
	list := make([]string, len(s))
	for i, e := range s {
		list[i] = f(e)
	}

	slices.Sort(list)

	encoder := json.NewEncoder(w)
	if err := encoder.Encode(list); err != nil {
		common.SendError(w, logger, common.NewErrorWith(
			http.StatusInternalServerError, "failed to encode list", err))
	}
}

func listClients(db database.ReadAccess) (model.Clients, error) {
	var clients model.Clients
	if err := db.Select(&clients).Owner().Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list clients", err)
	}

	return clients, nil
}

func listServers(db database.ReadAccess) (model.LocalAgents, error) {
	var servers model.LocalAgents
	if err := db.Select(&servers).Owner().Run(); err != nil {
		return nil, common.NewErrorWith(http.StatusInternalServerError,
			"failed to list servers", err)
	}

	return servers, nil
}

func listPartners(db database.ReadAccess, protocol string) (model.RemoteAgents, error) {
	var partners model.RemoteAgents
	if err := db.Select(&partners).Owner().Where("protocol=?", protocol).Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list partners", err)
	}

	return partners, nil
}

func listRules(db database.ReadAccess, r *http.Request) (model.Rules, error) {
	var isSend bool

	switch direction := r.URL.Query().Get("direction"); direction {
	case "send":
		isSend = true
	case "receive":
		isSend = false
	case "":
		return nil, common.NewError(http.StatusBadRequest, "missing direction URL parameter")
	default:
		return nil, common.NewError(http.StatusBadRequest, fmt.Sprintf("unknown direction %q", direction))
	}

	var rules model.Rules
	if err := db.Select(&rules).Where("is_send=?", isSend).Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list rules", err)
	}

	return rules, nil
}

func getClient(db database.ReadAccess, r *http.Request) (*model.Client, error) {
	clientName := r.URL.Query().Get("client")
	if clientName == "" {
		return nil, common.NewError(http.StatusBadRequest, "missing client URL parameter")
	}

	var client model.Client
	if err := db.Get(&client, "name=?", clientName).Owner().Run(); database.IsNotFound(err) {
		return nil, common.NewError(http.StatusNotFound,
			fmt.Sprintf("client %q does not exist", clientName))
	} else if err != nil {
		return nil, common.NewErrorWith(http.StatusInternalServerError,
			"failed to retrieve client", err)
	}

	return &client, nil
}

func getServer(db database.ReadAccess, r *http.Request) (*model.LocalAgent, error) {
	serverName := r.URL.Query().Get("server")
	if serverName == "" {
		return nil, common.NewError(http.StatusBadRequest, "missing server URL parameter")
	}

	var server model.LocalAgent
	if err := db.Get(&server, "name=?", serverName).Owner().Run(); database.IsNotFound(err) {
		return nil, common.NewError(http.StatusNotFound,
			fmt.Sprintf("server %q does not exist", serverName))
	} else if err != nil {
		return nil, common.NewErrorWith(http.StatusInternalServerError,
			"failed to retrieve server", err)
	}

	return &server, nil
}

func getPartner(db database.ReadAccess, r *http.Request) (*model.RemoteAgent, error) {
	partnerName := r.URL.Query().Get("partner")
	if partnerName == "" {
		return nil, common.NewError(http.StatusBadRequest, "missing partner URL parameter")
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", partnerName).Owner().Run(); database.IsNotFound(err) {
		return nil, common.NewError(http.StatusNotFound,
			fmt.Sprintf("partner %q does not exist", partnerName))
	} else if err != nil {
		return nil, common.NewErrorWith(http.StatusInternalServerError,
			"failed to retrieve partner", err)
	}

	return &partner, nil
}

func listLocalAccounts(db database.ReadAccess, server *model.LocalAgent,
) (model.LocalAccounts, error) {
	var accounts model.LocalAccounts
	if err := db.Select(&accounts).Where("local_agent_id=?", server.ID).Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list local accounts", err)
	}

	return accounts, nil
}

func listRemoteAccounts(db database.ReadAccess, partner *model.RemoteAgent,
) (model.RemoteAccounts, error) {
	var accounts model.RemoteAccounts
	if err := db.Select(&accounts).Where("remote_agent_id=?", partner.ID).Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list remote accounts", err)
	}

	return accounts, nil
}

func getKeyTypes(r *http.Request) ([]any, error) {
	query := r.URL.Query()
	operation := query.Get("operation")
	method := query.Get("method")

	if operation == "" {
		return nil, common.NewError(http.StatusBadRequest, `missing "operation" URL parameter`)
	}

	if method == "" {
		return nil, common.NewError(http.StatusBadRequest, `missing "method" URL parameter`)
	}

	switch operation {
	case "encrypt":
		if info, ok := tasks.EncryptMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypes), nil
		}
	case "decrypt":
		if info, ok := tasks.DecryptMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypes), nil
		}
	case "sign":
		if info, ok := tasks.SignMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypes), nil
		}
	case "verify":
		if info, ok := tasks.VerifyMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypes), nil
		}
	case "encrypt-sign":
		if info, ok := tasks.EncryptSignMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypesEncrypt), nil
		}
	case "sign-encrypt":
		if info, ok := tasks.EncryptSignMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypesSign), nil
		}
	case "decrypt-verify":
		if info, ok := tasks.DecryptVerifyMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypesDecrypt), nil
		}
	case "verify-decrypt":
		if info, ok := tasks.DecryptVerifyMethods.Get(method); ok {
			return utils.AsAny(info.KeyTypesVerify), nil
		}

	default:
		return nil, common.NewError(http.StatusBadRequest,
			fmt.Sprintf(`unknown crypto operation %q`, operation))
	}

	return nil, common.NewError(http.StatusBadRequest,
		fmt.Sprintf(`unknown %s crypto method %q`, operation, method))
}

func listCryptoKeys(db database.ReadAccess, types []any) (model.CryptoKeys, error) {
	var keys model.CryptoKeys
	if err := db.Select(&keys).Owner().In("type", types...).Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list crypto keys", err)
	}

	return keys, nil
}

func listEmailTemplates(db database.ReadAccess) (model.EmailTemplates, error) {
	var templates model.EmailTemplates
	if err := db.Select(&templates).Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list email templates", err)
	}

	return templates, nil
}

func listSMTPCredentials(db database.ReadAccess) (model.SMTPCredentials, error) {
	var creds model.SMTPCredentials
	if err := db.Select(&creds).Owner().Run(); err != nil {
		return nil, common.NewErrorWith(
			http.StatusInternalServerError, "failed to list smtp credentials", err)
	}

	return creds, nil
}
