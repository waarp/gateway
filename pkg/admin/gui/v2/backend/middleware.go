package backend

import (
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const errorsTranslationFile = "errors.yaml"

func withPermissions(perm model.PermsMask, next http.Handler) http.HandlerFunc {
	errorsLocalization := locale.ParseLocalizationFile(errorsTranslationFile)

	return func(w http.ResponseWriter, r *http.Request) {
		user := common.GetUser(r)
		language := locale.GetLanguage(r)

		if !user.Permissions.HasPermission(perm) {
			text := locale.MakeLocalText(language, errorsLocalization)

			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintln(w, text["unauthorized"])
		}

		next.ServeHTTP(w, r)
	}
}
