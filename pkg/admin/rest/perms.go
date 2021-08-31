package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func invalidMode(mode string) error {
	return badRequest("invalid permission mode '%s'", mode)
}

//nolint:gomnd // too specific
func permToMask(old *model.PermsMask, perm string, off int) error {
	if len(perm) == 0 {
		return nil
	}

	var process func(int)

	ops := map[rune]func(){
		'=': func() {
			*old &^= 0b111 << (32 - 1 - off - 2)
			process = func(o int) {
				*old |= 1 << (32 - 1 - off - o)
			}
		},
		'+': func() {
			process = func(o int) {
				*old |= 1 << (32 - 1 - off - o)
			}
		},
		'-': func() {
			process = func(o int) {
				*old &^= 1 << (32 - 1 - off - o)
			}
		},
	}
	modes := map[byte]func(){
		'r': func() {
			process(0)
		},
		'w': func() {
			process(1)
		},
		'd': func() {
			process(2)
		},
	}

	for i := range perm {
		if procOp, ok := ops[rune(perm[i])]; ok {
			procOp()

			continue
		}

		if process == nil {
			*old &^= 0b111 << (32 - 1 - off - 2)
			process = func(o int) {
				*old |= 1 << (32 - 1 - off - o)
			}
		}

		if procMode, ok := modes[perm[i]]; ok {
			procMode()

			continue
		}

		return invalidMode(perm)
	}

	return nil
}

func permsToMask(old model.PermsMask, perms *api.Perms) (model.PermsMask, error) {
	if perms == nil {
		return old, nil
	}

	if err := permToMask(&old, perms.Transfers, 0); err != nil {
		return 0, err
	}

	if err := permToMask(&old, perms.Servers, 3); err != nil { //nolint:gomnd // too specific
		return 0, err
	}

	if err := permToMask(&old, perms.Partners, 6); err != nil { //nolint:gomnd // too specific
		return 0, err
	}

	if err := permToMask(&old, perms.Rules, 9); err != nil { //nolint:gomnd // too specific
		return 0, err
	}

	if err := permToMask(&old, perms.Users, 12); err != nil { //nolint:gomnd // too specific
		return 0, err
	}

	if err := permToMask(&old, perms.Administration, 15); err != nil { //nolint:gomnd // too specific
		return 0, err
	}

	return old, nil
}

func maskToPerms(m model.PermsMask) *api.Perms {
	perms := model.MaskToPerms(m)

	return &api.Perms{
		Transfers:      perms.Transfers,
		Servers:        perms.Servers,
		Partners:       perms.Partners,
		Rules:          perms.Rules,
		Users:          perms.Users,
		Administration: perms.Administration,
	}
}

type (
	handler        func(*log.Logger, *database.DB) http.HandlerFunc
	handlerNoDB    func(*log.Logger) http.HandlerFunc
	handlerFactory func(string, handler, model.PermsMask, ...string)
)

func makeHandlerFactory(logger *log.Logger, db *database.DB, router *mux.Router) handlerFactory {
	return func(path string, handle handler, perm model.PermsMask, methods ...string) {
		var auth http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			login, _, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the request is missing credentials", http.StatusUnauthorized)

				return
			}

			var user model.User
			if err := db.Get(&user, "username=? AND owner=?", login, conf.GlobalConfig.GatewayName).
				Run(); err != nil {
				logger.Error("Database error: %s", err)
				http.Error(w, "internal database error", http.StatusInternalServerError)

				return
			}

			if perm&user.Permissions != perm {
				logger.Warning("User '%s' tried method '%s' on '%s' without sufficient privileges",
					login, r.Method, r.URL)
				http.Error(w, "you do not have sufficient privileges to perform this action",
					http.StatusForbidden)

				return
			}

			handle(logger, db).ServeHTTP(w, r)
		}

		for _, method := range methods {
			router.HandleFunc(path, auth).Methods(method).Name(method + " " + path)
		}
	}
}

func (f handlerFactory) noDB(path string, handle handlerNoDB, perm model.PermsMask, methods ...string) {
	f(path, func(logger *log.Logger, _ *database.DB) http.HandlerFunc {
		return handle(logger)
	}, perm, methods...)
}
