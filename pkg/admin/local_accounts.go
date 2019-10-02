package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-xorm/builder"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func createLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account := model.LocalAccount{}

		if err := restCreate(db, r, &account); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := strconv.FormatUint(account.ID, 10)
		w.Header().Set("Location", APIPath+LocalAccountsPath+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

func listAccounts(db *database.Db, r *http.Request, validSorting []string,
	agentCol string, results interface{}) error {

	limit := 20
	offset := 0
	order := "login"
	if err := parseLimitOffsetOrder(r, &limit, &offset, &order, validSorting); err != nil {
		return err
	}

	agents := r.Form["agent"]
	conditions := make([]builder.Cond, 0)
	if len(agents) > 0 {
		ids := make([]uint64, len(agents))
		for i, agent := range agents {
			id, err := strconv.ParseUint(agent, 10, 64)
			if err != nil {
				return fmt.Errorf("'%s' is not a valid agent ID", agent)
			}
			ids[i] = id
		}

		conditions = append(conditions, builder.In(agentCol, ids))
	}

	filters := &database.Filters{
		Limit:      limit,
		Offset:     offset,
		Order:      order,
		Conditions: builder.And(conditions...),
	}

	if err := db.Select(results, filters); err != nil {
		return err
	}

	return nil
}

func listLocalAccounts(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// WARNING: the entries of `validSorting` MUST be ordered alphabetically
		validSorting := []string{"local_agent_id", "login"}

		results := []model.LocalAccount{}
		if err := listAccounts(db, r, validSorting, "local_agent_id", &results); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.LocalAccount{"localAccounts": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func getLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "local_account")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := model.LocalAccount{ID: id}

		if err := restGet(db, &account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		if err := writeJSON(w, &account); err != nil {
			handleErrors(w, logger, err)
			return
		}
	}
}

func updateLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "local_account")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := model.LocalAccount{}

		if err := restUpdate(db, r, &account, id); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", APIPath+LocalAccountsPath+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteLocalAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "local_account")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := model.LocalAccount{ID: id}

		if err := restDelete(db, &account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
