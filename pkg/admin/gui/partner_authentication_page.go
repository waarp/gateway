package gui

import (
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func partnerAuthenticationPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tabTrans := pageTranslated("partner_authentication_page", userLanguage.(string)) //nolint:errcheck,forcetypeassert //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var partner *model.RemoteAgent

		partnerID := r.URL.Query().Get("partnerID")
		if partnerID != "" {
			id, err := strconv.Atoi(partnerID)
			if err != nil {
				logger.Error("failed to convert id to int: %v", err)
			}

			partner, err = internal.GetPartnerByID(db, int64(id))
			if err != nil {
				logger.Error("failed to get id: %v", err)
			}
		}

		if err := partnerAuthenticationTemplate.ExecuteTemplate(w, "partner_authentication_page", map[string]any{
			"myPermission": myPermission,
			"tab":          tabTrans,
			"username":     user.Username,
			"language":     userLanguage,
			"partner":      partner,
		}); err != nil {
			logger.Error("render partner_management_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
