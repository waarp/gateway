package webui

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/session"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/internal/translations"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/pages/home"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/pages/login"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/webui/pages/users"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"github.com/templui/templui/assets"
	"github.com/templui/templui/utils"
)

type Handler struct {
	db       *database.DB
	logger   *log.Logger
	isDev    bool
	sessions *session.Store
}

func NewHandler(db *database.DB, logger *log.Logger) *Handler {
	return &Handler{
		db:       db,
		logger:   logger,
		sessions: session.NewStore(),
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux := http.NewServeMux()

	h.AddPublicRoutes(mux)
	h.AddPrivateRoutes(mux)

	mux.ServeHTTP(w, r)
}

func (h *Handler) AddPublicRoutes(mux *http.ServeMux) {
	setupAssetsRoutes(mux, h.isDev)

	mux.HandleFunc("GET /login", login.ServeLoginPage)
	mux.HandleFunc("GET /set-language", translations.SetLanguage)

	mux.Handle("POST /login", login.HandleLogin(h.db, h.logger, h.sessions))
}

func (h *Handler) AddPrivateRoutes(rootMux *http.ServeMux) {
	rootMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		user := h.sessions.Get(r)
		if user == nil && !h.isDev {
			http.Redirect(w, r, "/login", http.StatusFound)

			return
		}

		r = session.WithUser(r, user)
		mux := http.NewServeMux()

		mux.Handle("GET /", home.Handle())
		mux.Handle("GET /logout", login.HandleLogout(h.db, h.logger, h.sessions))
		mux.Handle("GET /users", users.Handle())

		mux.ServeHTTP(w, r)
	})
}

func setupAssetsRoutes(mux *http.ServeMux, isDev bool) {
	assetHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isDev {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Cache-Control", "public, max-age=31536000")
		}

		var fs http.Handler
		if isDev {
			fs = http.FileServer(http.Dir("./assets"))
		} else {
			fs = http.FileServer(http.FS(assets.Assets))
		}

		fs.ServeHTTP(w, r)
	})

	mux.Handle("GET /assets/", http.StripPrefix("/assets/", assetHandler))

	// templUI embedded component scripts
	utils.SetupScriptRoutes(mux, isDev)
}
