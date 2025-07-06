package gui

import (
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func accountAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("account_authentication_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var partner *model.RemoteAgent
		var id int

		partnerID := r.URL.Query().Get("partnerID")
		if partnerID != "" {
			id, err = strconv.Atoi(partnerID)
			if err != nil {
				logger.Error("failed to convert id to int: %v", err)
			}

			partner, err = internal.GetPartnerByID(db, int64(id))
			if err != nil {
				logger.Error("failed to get id: %v", err)
			}
		}

		if err := accountAuthenticationTemplate.ExecuteTemplate(w, "account_authentication_page", map[string]any{
			"myPermission": myPermission,
			"username":     user.Username,
			"language":     userLanguage,
			"partner":      partner,
			"tab":          tTranslated,
			"hasPartnerID": true,
		}); err != nil {
			logger.Error("render account_authentication_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
