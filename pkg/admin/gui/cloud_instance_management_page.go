package gui

import (
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func ListCLoudInstance(db *database.DB, r *http.Request) ([]*model.CloudInstance, Filters, string) {
	cloudFound := ""
	filter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}
	urlParams := r.URL.Query()

	filter.OrderAsc = urlParams.Get("orderAsc") == "true"

	if limitRes := urlParams.Get("limit"); limitRes != "" {
		if l, err := strconv.ParseUint(limitRes, 10, 64); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := strconv.ParseUint(offsetRes, 10, 64); err == nil {
			filter.Offset = o
		}
	}

	cloudInstance, err := internal.ListClouds(db, "name", filter.OrderAsc, 0, 0)
	if err != nil {
		return nil, Filters{}, cloudFound
	}

	if search := urlParams.Get("search"); search != "" && searchInstanceCloud(search, cloudInstance) == nil {
		cloudFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		cloudFound = "true"

		return []*model.CloudInstance{searchInstanceCloud(search, cloudInstance)}, filter, cloudFound
	}

	paginationPage(&filter, uint64(len(cloudFound)), r)

	cloudInstances, err := internal.ListClouds(db, "name",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, cloudFound
	}

	return cloudInstances, filter, cloudFound
}

func searchInstanceCloud(cloudNameSearch string, listCloudSearch []*model.CloudInstance) *model.CloudInstance {
	for _, p := range listCloudSearch {
		if p.Name == cloudNameSearch {
			return p
		}
	}

	return nil
}

func cloudInstanceManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("cloud_instance_management_page", userLanguage.(string)) //nolint:errcheck //u
		cloudInstanceList, filter, cloudFound := ListCLoudInstance(db, r)

		// value, errMsg, modalOpen := callMethodsPartnerManagement(logger, db, w, r)
		// if value {
		// 	return
		// }

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if err := cloudInstanceManagementTemplate.ExecuteTemplate(w, "cloud_instance_management_page", map[string]any{
			"myPermission":      myPermission,
			"tab":               tTranslated,
			"username":          user.Username,
			"language":          userLanguage,
			"cloudInstanceList": cloudInstanceList,
			"cloudFound":        cloudFound,
			"filter":            filter,
			"currentPage":       currentPage,
			// "errMsg":                 errMsg,
			// "modalOpen":              modalOpen,
		}); err != nil {
			logger.Error("render cloud_instance_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
