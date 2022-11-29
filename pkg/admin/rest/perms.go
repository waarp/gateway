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
func alterPerm(old, perm string) (string, error) {
	res := &[3]byte{}
	if len(old) == 3 {
		res = (*[3]byte)([]byte(old))
	}

	set := func(i int, val byte) { res[i] = val }
	unset := func(i int, _ byte) { res[i] = '-' }

	do := set

	for i := range perm {
		switch b := perm[i]; b {
		case '+':
			do = set
		case '=':
			res = &[3]byte{'-', '-', '-'}
			do = set
		case '-':
			do = unset
		case 'r':
			do(0, 'r')
		case 'w':
			do(1, 'w')
		case 'd':
			do(2, 'd')
		default:
			return "", invalidMode(perm)
		}
	}

	return string(res[:]), nil
}

func alterPerms(old *model.Permissions, perms *api.Perms) error {
	if perms == nil {
		return nil
	}

	var err error

	if old.Transfers, err = alterPerm(old.Transfers, perms.Transfers); err != nil {
		return err
	}

	if old.Servers, err = alterPerm(old.Servers, perms.Servers); err != nil {
		return err
	}

	if old.Partners, err = alterPerm(old.Partners, perms.Partners); err != nil {
		return err
	}

	if old.Rules, err = alterPerm(old.Rules, perms.Rules); err != nil {
		return err
	}

	if old.Users, err = alterPerm(old.Users, perms.Users); err != nil {
		return err
	}

	if old.Administration, err = alterPerm(old.Administration, perms.Administration); err != nil {
		return err
	}

	return nil
}

func permsToMask(old model.PermsMask, perms *api.Perms) (model.PermsMask, error) {
	oldPerms := model.MaskToPerms(old)

	if err := alterPerms(oldPerms, perms); err != nil {
		return 0, err
	}

	return model.PermsToMask(oldPerms)
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
