package rest

import (
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:dupl //duplicate is for servers, best keep separate
func retrievePartner(r *http.Request, db *database.DB) (*model.RemoteAgent, error) {
	agentName, ok := mux.Vars(r)["partner"]
	if !ok {
		return nil, notFound("missing partner name")
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=? AND owner=?", agentName,
		conf.GlobalConfig.GatewayName).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("partner '%s' not found", agentName)
		}

		return nil, fmt.Errorf("failed to retrieve partner %q: %w", agentName, err)
	}

	return &partner, nil
}

//nolint:dupl //duplicate is for another type
func addPartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restPartner api.InPartner
		if err := readJSON(r, &restPartner); handleError(w, logger, err) {
			return
		}

		dbPartner, err := restPartnerToDB(&restPartner)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Insert(dbPartner).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbPartner.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

//nolint:dupl //duplicate is for completely different type (history), keep separate
func listPartners(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"proto+":  order{"protocol", true},
		"proto-":  order{"protocol", false},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbPartners model.RemoteAgents
		query, parseErr := parseSelectQuery(r, db, validSorting, &dbPartners)

		if handleError(w, logger, parseErr) {
			return
		}

		if err := parseProtoParam(r, query); handleError(w, logger, err) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restPartners, convErr := DBPartnersToREST(db, dbPartners)
		if handleError(w, logger, convErr) {
			return
		}

		response := map[string][]*api.OutPartner{"partners": restPartners}
		handleError(w, logger, writeJSON(w, response))
	}
}

func getPartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		restPartner, err := DBPartnerToREST(db, dbPartner)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, restPartner))
	}
}

func deletePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(dbPartner).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func updatePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldPartner, getErr := retrievePartner(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restPartner := &api.InPartner{
			Name:        api.AsNullable(oldPartner.Name),
			Protocol:    api.AsNullable(oldPartner.Protocol),
			Address:     api.AsNullable(oldPartner.Address.String()),
			ProtoConfig: oldPartner.ProtoConfig,
		}
		if err := readJSON(r, restPartner); handleError(w, logger, err) {
			return
		}

		dbPartner := &model.RemoteAgent{
			Name:        restPartner.Name.Value,
			Protocol:    restPartner.Protocol.Value,
			ProtoConfig: restPartner.ProtoConfig,
		}

		if err := dbPartner.Address.Set(restPartner.Address.Value); handleError(w, logger, err) {
			return
		}

		dbPartner.ID = oldPartner.ID

		if err := db.Update(dbPartner).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbPartner.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replacePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldPartner, getErr := retrievePartner(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restPartner api.InPartner
		if err := readJSON(r, &restPartner); handleError(w, logger, err) {
			return
		}

		dbPartner, convErr := restPartnerToDB(&restPartner)
		if handleError(w, logger, convErr) {
			return
		}

		dbPartner.ID = oldPartner.ID

		if err := db.Update(dbPartner).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbPartner.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func authorizePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, authorizeRule(w, r, db, dbPartner))
	}
}

func revokePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, revokeRule(w, r, db, dbPartner))
	}
}

func addPartnerCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, addCredential(w, r, db, dbPartner))
	}
}

func getPartnerCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, getCredential(w, r, db, dbPartner))
	}
}

func removePartnerCred(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, removeCredential(w, r, db, ag))
	}
}

// Deprecated: replaced by Credentials.
func getPartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, getCrypto(w, r, db, dbPartner))
	}
}

// Deprecated: replaced by Credentials.
func addPartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, createCrypto(w, r, db, dbPartner))
	}
}

// Deprecated: replaced by Credentials.
func listPartnerCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, listCryptos(w, r, db, dbPartner))
	}
}

// Deprecated: replaced by Credentials.
func deletePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, deleteCrypto(w, r, db, dbPartner))
	}
}

// Deprecated: replaced by Credentials.
func updatePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, updateCrypto(w, r, db, dbPartner))
	}
}

// Deprecated: replaced by Credentials.
func replacePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbPartner, err := retrievePartner(r, db)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, replaceCrypto(w, r, db, dbPartner))
	}
}
