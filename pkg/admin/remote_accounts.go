package admin

import (
	"net/http"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/database"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

func getRemoteAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "remote_account")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := model.RemoteAccount{ID: id}

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

func listRemoteAccounts(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// WARNING: the entries of `validSorting` MUST be ordered alphabetically
		validSorting := []string{"login", "remote_agent_id"}

		results := []model.RemoteAccount{}
		if err := listAccounts(db, r, validSorting, "remote_agent_id", &results); err != nil {
			handleErrors(w, logger, err)
			return
		}

		resp := map[string][]model.RemoteAccount{"remoteAccounts": results}
		if err := writeJSON(w, resp); err != nil {
			handleErrors(w, logger, err)
		}
	}
}

func createRemoteAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		account := model.RemoteAccount{}

		if err := restCreate(db, r, &account); err != nil {
			handleErrors(w, logger, err)
			return
		}

		newID := strconv.FormatUint(account.ID, 10)
		w.Header().Set("Location", APIPath+RemoteAccountsPath+"/"+newID)
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteRemoteAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "remote_account")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := model.RemoteAccount{ID: id}

		if err := restDelete(db, &account); err != nil {
			handleErrors(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func updateRemoteAccount(logger *log.Logger, db *database.Db) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := parseID(r, "remote_account")
		if err != nil {
			handleErrors(w, logger, &notFound{})
			return
		}
		account := model.RemoteAccount{}

		if err := restUpdate(db, r, &account, id); err != nil {
			handleErrors(w, logger, err)
			return
		}

		strID := strconv.FormatUint(id, 10)
		w.Header().Set("Location", APIPath+RemoteAccountsPath+"/"+strID)
		w.WriteHeader(http.StatusCreated)
	}
}
