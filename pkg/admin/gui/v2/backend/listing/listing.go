package listing

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func ListClients(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		clients, cErr := listClients(db)
		if common.SendError(w, logger, cErr) {
			return
		}

		encodeList(w, logger, clients, func(c *model.Client) string {
			return c.Name
		})
	}
}

func ListServers(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		servers, sErr := listServers(db)
		if common.SendError(w, logger, sErr) {
			return
		}

		encodeList(w, logger, servers, func(s *model.LocalAgent) string {
			return s.Name
		})
	}
}

func ListPartners(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		client, cErr := getClient(db, r)
		if common.SendError(w, logger, cErr) {
			return
		}

		partners, dbErr := listPartners(db, client.Protocol)
		if common.SendError(w, logger, dbErr) {
			return
		}

		encodeList(w, logger, partners, func(p *model.RemoteAgent) string {
			return p.Name
		})
	}
}

func ListLocalAccounts(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		server, pErr := getServer(db, r)
		if common.SendError(w, logger, pErr) {
			return
		}

		accounts, aErr := listLocalAccounts(db, server)
		if common.SendError(w, logger, aErr) {
			return
		}

		encodeList(w, logger, accounts, func(acc *model.LocalAccount) string {
			return acc.Login
		})
	}
}

func ListRemoteAccounts(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partner, pErr := getPartner(db, r)
		if common.SendError(w, logger, pErr) {
			return
		}

		accounts, aErr := listRemoteAccounts(db, partner)
		if common.SendError(w, logger, aErr) {
			return
		}

		encodeList(w, logger, accounts, func(acc *model.RemoteAccount) string {
			return acc.Login
		})
	}
}

func ListRules(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rules, rErr := listRules(db, r)
		if common.SendError(w, logger, rErr) {
			return
		}

		encodeList(w, logger, rules, func(r *model.Rule) string {
			return r.Name
		})
	}
}

func ListCryptoKeys(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		types, tErr := getKeyTypes(r)
		if common.SendError(w, logger, tErr) {
			return
		}

		keys, dbErr := listCryptoKeys(db, types)
		if common.SendError(w, logger, dbErr) {
			return
		}

		encodeList(w, logger, keys, func(key *model.CryptoKey) string {
			return key.Name
		})
	}
}

func ListEmailTemplates(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		templates, dbErr := listEmailTemplates(db)
		if common.SendError(w, logger, dbErr) {
			return
		}

		encodeList(w, logger, templates, func(t *model.EmailTemplate) string {
			return t.Name
		})
	}
}

func ListSMTPCredentials(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		creds, dbErr := listSMTPCredentials(db)
		if common.SendError(w, logger, dbErr) {
			return
		}

		encodeList(w, logger, creds, func(c *model.SMTPCredential) string {
			return c.EmailAddress
		})
	}
}
