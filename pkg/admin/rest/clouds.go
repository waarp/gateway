package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api/jsontypes"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func retrieveCloud(r *http.Request, db *database.DB) (*model.CloudInstance, error) {
	cloudName, ok := mux.Vars(r)["cloud"]
	if !ok {
		return nil, notFound("missing cloud name")
	}

	var cloud model.CloudInstance
	if err := db.Get(&cloud, "owner=? AND name=?", conf.GlobalConfig.GatewayName,
		cloudName).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFound("cloud %q not found", cloudName)
		}
	}

	return &cloud, nil
}

func dbCloudToREST(cloud *model.CloudInstance) *api.GetCloudRespObject {
	return &api.GetCloudRespObject{
		Name:    cloud.Name,
		Type:    cloud.Type,
		Key:     cloud.Key,
		Options: cloud.Options,
	}
}

func dbCloudsToREST(dbClouds []*model.CloudInstance) []*api.GetCloudRespObject {
	restClouds := make([]*api.GetCloudRespObject, len(dbClouds))

	for i, dbCloud := range dbClouds {
		restClouds[i] = dbCloudToREST(dbCloud)
	}

	return restClouds
}

func restCloudToDB(cloud *api.PostCloudReqObject) *model.CloudInstance {
	return &model.CloudInstance{
		Name:    cloud.Name,
		Type:    cloud.Type,
		Key:     cloud.Key,
		Secret:  types.CypherText(cloud.Secret),
		Options: cloud.Options,
	}
}

func listClouds(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
		"type+":   order{"type", true},
		"type-":   order{"type", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbClouds model.CloudInstances

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbClouds)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Where("owner=?", conf.GlobalConfig.GatewayName).
			Run(); handleError(w, logger, err) {
			return
		}

		restClouds := dbCloudsToREST(dbClouds)
		response := map[string][]*api.GetCloudRespObject{"clouds": restClouds}

		handleError(w, logger, writeJSON(w, response))
	}
}

func getCloud(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbCloud, getErr := retrieveCloud(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		handleError(w, logger, writeJSON(w, dbCloudToREST(dbCloud)))
	}
}

//nolint:dupl //duplicate is for a different type, keep separate
func addCloud(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restCloud api.PostCloudReqObject
		if err := readJSON(r, &restCloud); handleError(w, logger, err) {
			return
		}

		dbCloud := restCloudToDB(&restCloud)
		if err := db.Insert(dbCloud).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbCloud.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func updateCloud(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldCloud, getErr := retrieveCloud(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restCloud := api.PatchCloudReqObject{
			Name:    oldCloud.Name,
			Type:    oldCloud.Type,
			Key:     jsontypes.NewNullString(oldCloud.Key),
			Secret:  jsontypes.NewNullString(string(oldCloud.Secret)),
			Options: oldCloud.Options,
		}
		if err := readJSON(r, &restCloud); handleError(w, logger, err) {
			return
		}

		dbCloud := &model.CloudInstance{
			ID:      oldCloud.ID,
			Owner:   oldCloud.Owner,
			Name:    restCloud.Name,
			Type:    restCloud.Type,
			Key:     restCloud.Key.String,
			Secret:  types.CypherText(restCloud.Secret.String),
			Options: restCloud.Options,
		}

		if err := db.Update(dbCloud).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbCloud.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceCloud(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldCloud, getErr := retrieveCloud(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restCloud := api.PostCloudReqObject{}
		if err := readJSON(r, &restCloud); handleError(w, logger, err) {
			return
		}

		dbCloud := restCloudToDB(&restCloud)
		dbCloud.ID = oldCloud.ID

		if err := db.Update(dbCloud).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbCloud.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteCloud(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbCloud, getErr := retrieveCloud(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(dbCloud).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
