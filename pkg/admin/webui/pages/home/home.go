package home

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/common"
	"github.com/a-h/templ"
)

type homeHandler struct{}

func Handle() http.Handler {
	return homeHandler{}
}

func (h homeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := common.GetPageData(r)

	login := Home(data)
	templ.Handler(login).ServeHTTP(w, r)
}
