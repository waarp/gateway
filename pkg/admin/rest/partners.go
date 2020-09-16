package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest/api"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/gorilla/mux"
)

func getRemAg(r *http.Request, db *database.DB) (*model.RemoteAgent, error) {
	agentName, ok := mux.Vars(r)["partner"]
	if !ok {
		return nil, notFound("missing partner name")
	}
	agent := &model.RemoteAgent{Name: agentName}
	if err := db.Get(agent); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("partner '%s' not found", agentName)
		}
		return nil, err
	}
	return agent, nil
}

func addPartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var part api.InPartner
		if err := readJSON(r, &part); handleError(w, logger, err) {
			return
		}

		agent := partToDB(&part, 0)
		if err := db.Create(agent); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, agent.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func listPartners(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := map[string]string{
		"default": "name ASC",
		"proto+":  "protocol ASC",
		"proto-":  "protocol DESC",
		"name+":   "name ASC",
		"name-":   "name DESC",
	}
	typ := (&model.RemoteAgent{}).TableName()

	return func(w http.ResponseWriter, r *http.Request) {
		filters, err := parseListFilters(r, validSorting)
		if handleError(w, logger, err) {
			return
		}
		if err := parseProtoParam(r, filters); handleError(w, logger, err) {
			return
		}

		var results []model.RemoteAgent
		if err := db.Select(&results, filters); handleError(w, logger, err) {
			return
		}

		ids := make([]uint64, len(results))
		for i, res := range results {
			ids[i] = res.ID
		}
		rules, err := getAuthorizedRuleList(db, typ, ids)
		if handleError(w, logger, err) {
			return
		}

		resp := map[string][]api.OutPartner{"partners": FromRemoteAgents(results, rules)}
		err = writeJSON(w, resp)
		handleError(w, logger, err)
	}
}

func getPartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := getRemAg(r, db)
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
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err := db.Delete(ag); handleError(w, logger, err) {
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updatePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		part := newInPartner(old)
		if err := readJSON(r, part); handleError(w, logger, err) {
			return
		}

		if err := db.Update(partToDB(part, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(part.Name)))
		w.WriteHeader(http.StatusCreated)
	}
}

func replacePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		old, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		var part api.InPartner
		if err := readJSON(r, &part); handleError(w, logger, err) {
			return
		}

		if err := db.Update(partToDB(&part, old.ID)); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, str(part.Name)))
		w.WriteHeader(http.StatusCreated)
	}
}

func authorizePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = authorizeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func revokePartner(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = revokeRule(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func getPartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = getCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func addPartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = createCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func listPartnerCerts(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = listCertificates(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func deletePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = deleteCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func updatePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = updateCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}

func replacePartnerCert(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ag, err := getRemAg(r, db)
		if handleError(w, logger, err) {
			return
		}

		err = replaceCertificate(w, r, db, ag.TableName(), ag.ID)
		handleError(w, logger, err)
	}
}
