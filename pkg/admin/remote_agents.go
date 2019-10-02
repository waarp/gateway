package admin

import (
	"net/http"
	"strconv"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func getRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "remote_agent")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		agent := model.RemoteAgent{ID: id}

		if err := restGet(db, &agent); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, &agent); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func listRemoteAgents(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 20
		offset := 0
		order := "name"
		// WARNING: the entries of `validSorting` MUST be ordered alphabetically
		validSorting := []string{"name", "protocol"}

		if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
			handleErrors(w, logger, err)
			return
		}

		protocols := r.Form["protocol"]

		conditions := make([]builder.Cond, 0)
		if len(protocols) > 0 {
			conditions = append(conditions, builder.In("protocol", protocols))
		}

		filters := database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.And(conditions...),
		}

		results := []model.RemoteAgent{}
		if err := db.Select(&results, &filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.RemoteAgent{"remoteAgents": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := model.RemoteAgent{}

		if err := restCreate(db, r, &agent); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := strconv.FormatUint(agent.ID, 10)
		w.Header().Set("Location", APIPath+RemoteAgentsPath+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "remote_agent")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		agent := model.RemoteAgent{ID: id}

		if err := restDelete(db, &agent); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateRemoteAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "remote_agent")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		agent := model.RemoteAgent{}

		if err := restUpdate(db, r, &agent, id); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", APIPath+RemoteAgentsPath+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
