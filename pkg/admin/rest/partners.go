package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getPart(r *http.Request, db *database.DB) (*model.RemoteAgent, error) {
	agentName, ok := mux.Vars(r)["partner"]
	if !ok {
		return nil, notFound("missing partner name")
	}

	var partner model.RemoteAgent
	if err := db.Get(&partner, "name=?", agentName).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("partner '%s' not found", agentName)
		}

		return nil, err
	}

	return &partner, nil
}

//nolint:dupl // duplicated code is about a different type
func addPartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var part api.InPartner
		if err := readJSON(r, &part); handleError(w, logger, err) {
			return
		}

		partner := partToDB(&part, 0)
		if err := db.Insert(partner).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, partner.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func listPartners(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"proto+":  order{"protocol", true},
		"proto-":  order{"protocol", false},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
	}
	typ := (&model.RemoteAgent{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		var partners model.RemoteAgents
		query, err := parseSelectQuery(r, db, validSorting, &partners)

		if handleError(w, logger, err) {
			return
		}

		if err2 := parseProtoParam(r, query); handleError(w, logger, err2) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		ids := make([]uint64, len(partners))
		for i := range partners {
			ids[i] = partners[i].ID
		}

		rules, err := getAuthorizedRuleList(db, typ, ids)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutPartner{"partners": FromRemoteAgents(partners, rules)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func getPartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		rules, err := getAuthorizedRules(db, result.TableName(), result.ID)
		if handleError(w, logger, err) {
			return
		}

		err = writeJSON(w, FromRemoteAgent(result, rules))
		handleError(w, logger, err)
	}
}

func deletePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		partner, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(partner).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

//nolint:dupl // duplicated code is about a different type
func updatePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		jPart := newInPartner(old)
		if err := readJSON(r, jPart); handleError(w, logger, err) {
			return
		}

		partner := partToDB(jPart, old.ID)
		if err := db.Update(partner).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, partner.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replacePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		var jPart api.InPartner
		if err := readJSON(r, &jPart); handleError(w, logger, err) {
			return
		}

		partner := partToDB(&jPart, old.ID)
		if err := db.Update(partner).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, partner.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func authorizePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = authorizeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func revokePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = revokeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func getPartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = getCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func addPartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = createCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func listPartnerCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = listCryptos(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func deletePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = deleteCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func updatePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func replacePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getPart(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCrypto(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}
