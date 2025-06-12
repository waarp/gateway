package gui

import (
	"net/http"
	"strconv"
	"fmt"
	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

var partnerFound = true //nolint:gochecknoglobals //partnerFound

func ListPartner(db *database.DB, r *http.Request) []*model.RemoteAgent {
	partnerFound = true
	orderAsc := false
	limit := 0
	var err error

	orderAscRes := r.URL.Query().Get("orderAsc")
	if orderAscRes == "true" {
		orderAsc = true
	}

	limitRes := r.URL.Query().Get("limit")
	if limitRes != "" {
		limit, err = strconv.Atoi(limitRes)
		if err != nil {
			limit = 0
		}
	}
	
	partner, err := internal.ListPartners(db, "name", orderAsc, limit, 0)
	if err != nil {
		return nil
	}

	search := r.URL.Query().Get("search")
	if search != "" {
		partnerSearch := searchPartner(search, partner)
		if partnerSearch != nil {
			return []*model.RemoteAgent{partnerSearch}
		} else {
			partnerFound = false
		}
	}

	return partner
}

func searchPartner(partnerNameSearch string, listPartnerSearch []*model.RemoteAgent) *model.RemoteAgent {
	for _, p := range listPartnerSearch {
		if p.Name == partnerNameSearch {
			return p
		}
	}

	return nil
}

func deletePartner(db *database.DB, r *http.Request) error {
	partnerID := r.URL.Query().Get("deletePartner")
	id, err := strconv.Atoi(partnerID)
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	partner, err := internal.GetPartnerByID(db, int64(id))
	if err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	if err = internal.DeletePartner(db, partner); err != nil {
		return fmt.Errorf("internal error: %w", err)
	}

	return nil
}


func partnerManagementPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTranslated := pageTranslated("partner_management_page", userLanguage.(string))
		partnerList := ListPartner(db, r)
        for _, partner := range partnerList {
            fmt.Printf("%s:\n", partner.Name)
			for key, value := range partner.ProtoConfig {
				fmt.Printf("%s: %v\n", key, value)
			}
        }

		if r.Method == http.MethodGet && r.URL.Query().Get("deletePartner") != "" {
			deletePartnerErr := deletePartner(db, r)
			if deletePartnerErr != nil {
				logger.Error("Internal error: %v", deletePartnerErr)
			}

			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)

			return
		}

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)

		if err := partnerManagementTemplate.ExecuteTemplate(w, "partner_management_page", map[string]any{
			"myPermission":    myPermission,
			"tab":             tabTranslated,
			"username":        user.Username,
			"language":        userLanguage,
			"partner":		   partnerList,
			"partnerFound":	   partnerFound,
		}); err != nil {
			logger.Error("render partner_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
