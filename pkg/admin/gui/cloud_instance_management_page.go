package gui

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:gochecknoglobals //a global var is required here
var ListTypeCloudInstance = []string{
	"s3",
}

func ListCLoudInstance(db *database.DB, r *http.Request) ([]*model.CloudInstance, Filters, string) {
	cloudFound := ""
	defaultFilter := Filters{
		Offset:          0,
		Limit:           DefaultLimitPagination,
		OrderAsc:        true,
		DisableNext:     false,
		DisablePrevious: false,
	}

	filter := defaultFilter
	if saved, ok := GetPageFilters(r, "cloud_instance_management_page"); ok {
		filter = saved
	}

	if r.URL.Query().Get("applyFilters") == True {
		filter = defaultFilter
	}

	urlParams := r.URL.Query()

	if urlParams.Get("orderAsc") != "" {
		filter.OrderAsc = urlParams.Get("orderAsc") == True
	}

	if limitRes := urlParams.Get("limit"); limitRes != "" {
		if l, err := internal.ParseUint[uint64](limitRes); err == nil {
			filter.Limit = l
		}
	}

	if offsetRes := urlParams.Get("offset"); offsetRes != "" {
		if o, err := internal.ParseUint[uint64](offsetRes); err == nil {
			filter.Offset = o
		}
	}

	cloudInstance, err := internal.ListClouds(db, "name", filter.OrderAsc, 0, 0)
	if err != nil {
		return nil, Filters{}, cloudFound
	}

	if search := urlParams.Get("search"); search != "" && searchCloudInstance(search, cloudInstance) == nil {
		cloudFound = "false"
	} else if search != "" {
		filter.DisableNext = true
		filter.DisablePrevious = true
		cloudFound = "true"

		return []*model.CloudInstance{searchCloudInstance(search, cloudInstance)}, filter, cloudFound
	}

	paginationPage(&filter, uint64(len(cloudInstance)), r)

	cloudInstances, err := internal.ListClouds(db, "name",
		filter.OrderAsc, int(filter.Limit), int(filter.Offset*filter.Limit))
	if err != nil {
		return nil, Filters{}, cloudFound
	}

	return cloudInstances, filter, cloudFound
}

func autocompletionCloudFunc(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		prefix := r.URL.Query().Get("q")

		clouds, err := internal.GetCloudInstanceLike(db, prefix)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		names := make([]string, len(clouds))
		for i, u := range clouds {
			names[i] = u.Name
		}

		w.Header().Set("Content-Type", "application/json")

		if jsonErr := json.NewEncoder(w).Encode(names); jsonErr != nil {
			http.Error(w, "error json", http.StatusInternalServerError)
		}
	}
}

func searchCloudInstance(cloudNameSearch string, listCloudSearch []*model.CloudInstance) *model.CloudInstance {
	for _, p := range listCloudSearch {
		if p.Name == cloudNameSearch {
			return p
		}
	}

	return nil
}

func addCloudInstance(db *database.DB, r *http.Request) error {
	var newCloudInstance model.CloudInstance

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if addCloudInstanceName := r.FormValue("addCloudInstanceName"); addCloudInstanceName != "" {
		newCloudInstance.Name = addCloudInstanceName
	}

	if addCloudInstanceType := r.FormValue("addCloudInstanceType"); addCloudInstanceType != "" {
		newCloudInstance.Type = addCloudInstanceType
	}

	if addCloudInstanceKey := r.FormValue("addCloudInstanceKey"); addCloudInstanceKey != "" {
		newCloudInstance.Key = addCloudInstanceKey
	}

	if addCloudInstanceSecret := r.FormValue("addCloudInstanceSecret"); addCloudInstanceSecret != "" {
		newCloudInstance.Secret = database.SecretText(addCloudInstanceSecret)
	}

	optionsMap := make(map[string]string)

	if addCloudInstanceBucket := r.FormValue("addCloudInstanceBucket"); addCloudInstanceBucket != "" {
		optionsMap["bucket"] = addCloudInstanceBucket
	}

	optionKeys := r.Form["optionsKey[]"]
	optionVals := r.Form["optionsValue[]"]

	for i := range optionKeys {
		if optionKeys[i] != "" && i < len(optionVals) {
			optionsMap[optionKeys[i]] = optionVals[i]
		}
	}
	newCloudInstance.Options = optionsMap

	if err := internal.InsertCloud(db, &newCloudInstance); err != nil {
		return fmt.Errorf("failed to add instance cloud: %w", err)
	}

	return nil
}

func editCloudInstance(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	cloudInstanceID := r.FormValue("editCloudInstanceID")

	id, err := internal.ParseUint[uint64](cloudInstanceID)
	if err != nil {
		return fmt.Errorf("failed to convert id to int: %w", err)
	}

	editCloudInstance, err := internal.GetCloudByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get id: %w", err)
	}

	if editCloudInstanceName := r.FormValue("editCloudInstanceName"); editCloudInstanceName != "" {
		editCloudInstance.Name = editCloudInstanceName
	}

	if editCloudInstanceType := r.FormValue("editCloudInstanceType"); editCloudInstanceType != "" {
		editCloudInstance.Type = editCloudInstanceType
	}

	if editCloudInstanceKey := r.FormValue("editCloudInstanceKey"); editCloudInstanceKey != "" {
		editCloudInstance.Key = editCloudInstanceKey
	}

	if editCloudInstanceSecret := r.FormValue("editCloudInstanceSecret"); editCloudInstanceSecret != "" {
		editCloudInstance.Secret = database.SecretText(editCloudInstanceSecret)
	}

	optionsMap := make(map[string]string)

	if editCloudInstanceBucket := r.FormValue("editCloudInstanceBucket"); editCloudInstanceBucket != "" {
		optionsMap["bucket"] = editCloudInstanceBucket
	}

	optionKeys := r.Form["editOptionsKey[]"]
	optionVals := r.Form["editOptionsValue[]"]

	for i := range optionKeys {
		if optionKeys[i] != "" && i < len(optionVals) {
			optionsMap[optionKeys[i]] = optionVals[i]
		}
	}
	editCloudInstance.Options = optionsMap

	if editErr := internal.UpdateCloud(db, editCloudInstance); editErr != nil {
		return fmt.Errorf("failed to edit cloud instance: %w", editErr)
	}

	return nil
}

func deleteCloudInstance(db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}
	cloudID := r.FormValue("deleteCloudInstance")

	id, err := internal.ParseUint[uint64](cloudID)
	if err != nil {
		return fmt.Errorf("failed to parse int: %w", err)
	}

	cloud, err := internal.GetCloudByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("failed to get cloud instance: %w", err)
	}

	if err = internal.DeleteCloud(db, cloud); err != nil {
		return fmt.Errorf("failed to delete cloud instance: %w", err)
	}

	return nil
}

//nolint:dupl // method for cloud instance
func callMethodsCloudInstance(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
) (value bool, errMsg, modalOpen string, modalElement map[string]any) {
	if r.Method == http.MethodPost && r.FormValue("addCloudInstanceName") != "" {
		if newCloudInstanceErr := addCloudInstance(db, r); newCloudInstanceErr != nil {
			logger.Errorf("failed to add instance cloud: %v", newCloudInstanceErr)
			modalElement = getFormValues(r)

			return false, newCloudInstanceErr.Error(), "addCloudInstanceModal", modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("deleteCloudInstance") != "" {
		deleteCloudInstanceErr := deleteCloudInstance(db, r)
		if deleteCloudInstanceErr != nil {
			logger.Errorf("failed to delete instance cloud: %v", deleteCloudInstanceErr)

			return false, deleteCloudInstanceErr.Error(), "", nil
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	if r.Method == http.MethodPost && r.FormValue("editCloudInstanceID") != "" {
		idEdit := r.FormValue("editCloudInstanceID")

		id, err := internal.ParseUint[uint64](idEdit)
		if err != nil {
			logger.Errorf("failed to convert id to int: %v", err)

			return false, "", "", nil
		}

		if editCloudInstanceErr := editCloudInstance(db, r); editCloudInstanceErr != nil {
			logger.Errorf("failed to edit cloud instance: %v", editCloudInstanceErr)
			modalElement = getFormValues(r)

			return false, editCloudInstanceErr.Error(), fmt.Sprintf("editCloudInstanceModal_%d", id), modalElement
		}

		http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

		return true, "", "", nil
	}

	return false, "", "", nil
}

func cloudInstanceManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("cloud_instance_management_page", userLanguage.(string)) //nolint:errcheck //u
		cloudInstanceList, filter, cloudFound := ListCLoudInstance(db, r)

		if pageName := r.URL.Query().Get("clearFiltersPage"); pageName != "" {
			ClearPageFilters(r, pageName)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		PersistPageFilters(r, "cloud_instance_management_page", &filter)

		value, errMsg, modalOpen, modalElement := callMethodsCloudInstance(logger, db, w, r)
		if value {
			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		currentPage := filter.Offset + 1

		if tmplErr := cloudInstanceManagementTemplate.ExecuteTemplate(w, "cloud_instance_management_page", map[string]any{
			"myPermission":          myPermission,
			"tab":                   tTranslated,
			"username":              user.Username,
			"language":              userLanguage,
			"cloudInstanceList":     cloudInstanceList,
			"ListTypeCloudInstance": ListTypeCloudInstance,
			"cloudFound":            cloudFound,
			"filter":                filter,
			"currentPage":           currentPage,
			"errMsg":                errMsg,
			"modalOpen":             modalOpen,
			"modalElement":          modalElement,
		}); tmplErr != nil {
			logger.Errorf("render cloud_instance_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
