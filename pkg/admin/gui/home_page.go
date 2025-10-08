//nolint:dupl // in progress
package gui

import (
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/version"
)

func homePage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)
		userLanguage := locale.GetLanguage(r)
		tabTranslated := pageTranslated("home_page", userLanguage)
		myPermission := model.MaskToPerms(user.Permissions)

		if tmplErr := homeTemplate.ExecuteTemplate(w, "home_page", map[string]any{
			"appName":      constants.AppName,
			"version":      version.Num,
			"compileDate":  version.Date,
			"revision":     version.Commit,
			"docLink":      constants.DocLink(userLanguage),
			"myPermission": myPermission,
			"tab":          tabTranslated,
			"username":     user.Username,
			"language":     userLanguage,
		}); tmplErr != nil {
			logger.Errorf("render home_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
