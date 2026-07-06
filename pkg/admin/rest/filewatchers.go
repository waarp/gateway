package rest

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/filewatcher"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func getDBFilewatcher(r *http.Request, db *database.DB) (*model.FileWatcher, error) {
	flow, ok := mux.Vars(r)["filewatcher"]
	if !ok {
		return nil, notFound("missing filewatcher flow name")
	}

	var dbFw model.FileWatcher
	if err := db.Get(&dbFw, "flow=?", flow).Eager().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("filewatcher %q not found", flow)
		}

		return nil, fmt.Errorf("failed to retrieve filewatcher %q: %w", flow, err)
	}

	if !filewatcher.Filewatchers.Exists(&dbFw) {
		fw := filewatcher.NewFilewatcher(db, &dbFw)
		filewatcher.Filewatchers.Add(&dbFw, fw)
	}

	return &dbFw, nil
}

func filewatcherDBtoREST(fw *model.FileWatcher, db *database.DB) (*api.OutFilewatcher, error) {
	var partner model.RemoteAgent
	if err := db.Get(&partner, "id=?", fw.RemoteAccount.RemoteAgentID).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve partner for filewatcher %d: %w", fw.ID, err)
	}

	return &api.OutFilewatcher{
		Flow:             fw.Flow,
		Disabled:         fw.Disabled,
		Interval:         api.Duration(fw.Interval),
		Pattern:          fw.Pattern,
		NoDuplicateCheck: fw.NoDuplicateCheck,
		Partner:          partner.Name,
		Account:          fw.RemoteAccount.Login,
		Client:           fw.Client.Name,
		Rule:             fw.Rule.Name,
	}, nil
}

func filewatchersDBtoREST(fws model.FileWatchers, db *database.DB) ([]*api.OutFilewatcher, error) {
	restFWs := make([]*api.OutFilewatcher, len(fws))

	for i, fw := range fws {
		restFW, err := filewatcherDBtoREST(fw, db)
		if err != nil {
			return nil, err
		}

		restFWs[i] = restFW
	}

	return restFWs, nil
}

func filewatcherRESTtoDB(restFW *api.InFilewatcher, db *database.DB) (*model.FileWatcher, error) {
	dbFW := &model.FileWatcher{}

	setIfValid(&dbFW.Flow, restFW.Flow)
	setIfValid(&dbFW.Disabled, restFW.Disabled)
	setIfValid(&dbFW.NoDuplicateCheck, restFW.NoDuplicateCheck)
	setIfValid(&dbFW.Pattern, restFW.Pattern)

	if restFW.Interval.Valid {
		dbFW.Interval = time.Duration(restFW.Interval.Value)
	}

	if restFW.Partner.Valid || restFW.Account.Valid {
		if !restFW.Partner.Valid || !restFW.Account.Valid {
			return nil, badRequest("partner and account must be provided together")
		}

		var partner model.RemoteAgent
		if err := db.Get(&partner, "name=?", restFW.Partner.Value).Run(); err != nil {
			if database.IsNotFound(err) {
				return nil, badRequestf("no partner %q found", restFW.Partner.Value)
			}

			return nil, fmt.Errorf("failed to retrieve partner %q: %w", restFW.Partner.Value, err)
		}

		var account model.RemoteAccount
		if err := db.Get(&account, "login=? AND remote_agent_id=?",
			restFW.Account.Value, partner.ID).Run(); err != nil {
			if database.IsNotFound(err) {
				return nil, badRequestf("no account %q found for partner %q",
					restFW.Account.Value, restFW.Partner.Value)
			}

			return nil, fmt.Errorf("failed to retrieve account %q: %w", restFW.Account.Value, err)
		}

		dbFW.RemoteAccount = account
	}

	if restFW.Client.Valid {
		var client model.Client
		if err := db.Get(&client, "name=?", restFW.Client.Value).Run(); err != nil {
			if database.IsNotFound(err) {
				return nil, badRequestf("no client %q found", restFW.Client.Value)
			}

			return nil, fmt.Errorf("failed to retrieve client %q: %w", restFW.Client.Value, err)
		}

		dbFW.Client = client
	}

	if restFW.Rule.Valid {
		var rule model.Rule
		if err := db.Get(&rule, "name=?", restFW.Rule.Value).And("is_send=?", false).Run(); err != nil {
			if database.IsNotFound(err) {
				return nil, badRequestf("no receive rule %q found", restFW.Rule.Value)
			}

			return nil, fmt.Errorf("failed to retrieve rule %q: %w", restFW.Rule.Value, err)
		}

		dbFW.Rule = rule
	}

	return dbFW, nil
}

func listFilewatchers(logger *log.Logger, db *database.DB) http.HandlerFunc {
	//nolint:goconst //best keep separate
	validSorting := orders{
		"default": order{"id", true},
		"flow+":   order{"flow", true},
		"flow-":   order{"flow", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbFWs model.FileWatchers

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbFWs)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restFWs, convErr := filewatchersDBtoREST(dbFWs, db)
		if handleError(w, logger, convErr) {
			return
		}

		resp := map[string][]*api.OutFilewatcher{"filewatchers": restFWs}
		handleError(w, logger, writeJSON(w, resp))
	}
}

func getFilewatcher(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbFW, getErr := getDBFilewatcher(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restFW, convErr := filewatcherDBtoREST(dbFW, db)
		if handleError(w, logger, convErr) {
			return
		}

		handleError(w, logger, writeJSON(w, restFW))
	}
}

func createFilewatcher(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restFW api.InFilewatcher
		if err := readJSON(r, &restFW); handleError(w, logger, err) {
			return
		}

		dbFW, convErr := filewatcherRESTtoDB(&restFW, db)
		if handleError(w, logger, convErr) {
			return
		}

		if err := db.Insert(dbFW).Run(); handleError(w, logger, err) {
			return
		}

		fw := filewatcher.NewFilewatcher(db, dbFW)
		filewatcher.Filewatchers.Add(dbFW, fw)

		w.Header().Set("Location", location(r.URL, dbFW.Flow))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateFilewatcher(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldFW, getErr := getDBFilewatcher(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		stopped, stopErr := filewatcher.Filewatchers.Stop(r.Context(), oldFW)
		if handleError(w, logger, stopErr) {
			return
		}

		var partner model.RemoteAgent
		if err := db.Get(&partner, "id=?", oldFW.RemoteAccount.RemoteAgentID).Run(); handleError(w, logger, err) {
			return
		}

		restFW := &api.InFilewatcher{
			Disabled:         asNullableBool(oldFW.Disabled),
			Flow:             asNullable(oldFW.Flow),
			Interval:         asNullable(api.Duration(oldFW.Interval)),
			Pattern:          asNullable(oldFW.Pattern),
			NoDuplicateCheck: asNullableBool(oldFW.NoDuplicateCheck),
			Partner:          asNullable(partner.Name),
			Account:          asNullable(oldFW.RemoteAccount.Login),
			Client:           asNullable(oldFW.Client.Name),
			Rule:             asNullable(oldFW.Rule.Name),
		}
		if err := readJSON(r, restFW); handleError(w, logger, err) {
			return
		}

		dbFW, convErr := filewatcherRESTtoDB(restFW, db)
		if handleError(w, logger, convErr) {
			return
		}

		dbFW.ID = oldFW.ID

		if err := db.Update(dbFW).Run(); handleError(w, logger, err) {
			return
		}

		if stopped {
			if _, err := filewatcher.Filewatchers.Start(dbFW); handleError(w, logger, err) {
				return
			}
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbFW.Flow))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteFilewatcher(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbFW, getErr := getDBFilewatcher(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(dbFW).Run(); handleError(w, logger, err) {
			return
		}

		if err := filewatcher.Filewatchers.Remove(r.Context(), dbFW); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

//nolint:dupl //duplicate is for a different type, keep separate
func stopFilewatcher(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbFW, getErr := getDBFilewatcher(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if _, err := filewatcher.Filewatchers.Stop(r.Context(), dbFW); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func startFilewatcher(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbFW, getErr := getDBFilewatcher(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := filewatcher.Filewatchers.Restart(r.Context(), dbFW); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}

func fireFilewatcher(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbFW, getErr := getDBFilewatcher(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		service, ok := filewatcher.Filewatchers.Get(dbFW)
		if !ok {
			handleError(w, logger, notFoundf("filewatcher service %q not found", dbFW.Flow))

			return
		}

		if err := service.FireOnce(dbFW); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}
}
