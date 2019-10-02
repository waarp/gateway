package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/go-xorm/builder"
)

func getLocalAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "local_agent")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		agent := &model.LocalAgent{ID: id}

		if err := restGet(db, agent); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, agent); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func listLocalAgents(logger *log.Logger, db *database.Db) http.HandlerFunc {
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
			for _, proto := range protocols {
				if !model.IsValidProtocol(proto) {
					handleErrors(w, logger, &badRequest{
						msg: fmt.Sprintf("'%s' is not a valid protocol", proto)})
				}
			}
			conditions = append(conditions, builder.In("protocol", protocols))
		}

		filters := &database.Filters{
			Limit:      limit,
			Offset:     offset,
			Order:      order,
			Conditions: builder.And(conditions...),
		}

		results := []model.LocalAgent{}
		if err := db.Select(&results, filters); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.LocalAgent{"localAgents": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createLocalAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//db.Dump()
		//defer db.Dump()
		agent := &model.LocalAgent{}

		if err := restCreate(db, r, agent); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := strconv.FormatUint(agent.ID, 10)
		w.Header().Set("Location", APIPath+LocalAgentsPath+"/"+newID)
		w.WriteHeader(http.StatusCreated)

	}
}

func deleteLocalAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "local_agent")
		if err != nil {
			handleErrors(w, logger, err)
			return
		}
		agent := &model.LocalAgent{ID: id}

		if err := restDelete(db, agent); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateLocalAgent(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "local_agent")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		agent := &model.LocalAgent{}

		if err := restUpdate(db, r, agent, id); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", APIPath+LocalAgentsPath+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
