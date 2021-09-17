package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func invalidMode(mode string) error {
	return badRequest("invalid permission mode '%s'", mode)
}

func permToMask(old *model.PermsMask, perm string, off int) error {
	if len(perm) == 0 {
		return nil
	}

	var process func(int)

	ops := map[rune]func(){
		'=': func() {
			*old &^= 0b111 << (32 - 1 - off - 2) //nolint:gomnd // too specific
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
			process(2) //nolint:gomnd // too specific
		},
	}

	for i := range perm {
		if procOp, ok := ops[rune(perm[i])]; ok {
			procOp()

			continue
		}

		if process == nil {
			*old &^= 0b111 << (32 - 1 - off - 2) //nolint:gomnd // too specific
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

	return old, nil
}

func maskToStr(m model.PermsMask, s int) string {
	const rwd = "rwd"

	buf := make([]byte, 3) //nolint:gomnd // too specific

	for i, c := range rwd {
		if m&(1<<uint(32-1-s-i)) != 0 {
			buf[i] = byte(c)
		} else {
			buf[i] = '-'
		}
	}

	return string(buf)
}

func maskToPerms(m model.PermsMask) *api.Perms {
	//nolint:gomnd // too specific
	return &api.Perms{
		Transfers: maskToStr(m, 0),
		Servers:   maskToStr(m, 3),
		Partners:  maskToStr(m, 6),
		Rules:     maskToStr(m, 9),
		Users:     maskToStr(m, 12),
	}
}

type (
	handler        func(*log.Logger, *database.DB) http.HandlerFunc
	handlerFactory func(string, handler, model.PermsMask, ...string)
)

func makeHandlerFactory(l *log.Logger, db *database.DB, router *mux.Router) handlerFactory {
	return func(path string, h handler, perm model.PermsMask, methods ...string) {
		var auth http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
			login, _, ok := r.BasicAuth()
			if !ok {
				w.Header().Set("WWW-Authenticate", "Basic")
				http.Error(w, "the request is missing credentials", http.StatusUnauthorized)

				return
			}

			var user model.User
			if err := db.Get(&user, "username=? AND owner=?", login, database.Owner).
				Run(); err != nil {
				l.Errorf("Database error: %s", err)
				http.Error(w, "internal database error", http.StatusInternalServerError)

				return
			}

			if perm&user.Permissions != perm {
				l.Warningf("User '%s' tried method '%s' on '%s' without sufficient privileges",
					login, r.Method, r.URL)
				http.Error(w, "you do not have sufficient privileges to perform this action",
					http.StatusForbidden)

				return
			}

			h(l, db).ServeHTTP(w, r)
		}

		router.HandleFunc(path, auth).Methods(methods...)
	}
}
