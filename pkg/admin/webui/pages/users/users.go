package users

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/common"
	"github.com/a-h/templ"
)

type handler struct{}

func Handle() http.Handler {
	return &handler{}
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := common.GetPageData(r)
	page := Users(data)

	templ.Handler(page).ServeHTTP(w, r)
}
