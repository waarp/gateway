//nolint:dupl //duplicate is for a separate page
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

func EmailTemplateManagementPage(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)
		userLanguage := locale.GetLanguage(r)
		tTranslated := pageTranslated("email_templates_management_page", userLanguage)

		myPermission := model.MaskToPerms(user.Permissions)

		if tmplErr := EmailTemplateManagementTemplate.ExecuteTemplate(w, "email_templates_management_page",
			map[string]any{
				"appName":      constants.AppName,
				"version":      version.Num,
				"compileDate":  version.Date,
				"revision":     version.Commit,
				"docLink":      constants.DocLink(userLanguage),
				"myPermission": myPermission,
				"tab":          tTranslated,
				"username":     user.Username,
				"language":     userLanguage,
			}); tmplErr != nil {
			logger.Errorf("render email_templates_management_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
