package admin

import (
	"net/http"
	"net/http/pprof"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

// MakeHandler returns the router for the REST & Admin http interface.
func MakeHandler(logger *log.Logger, db *database.DB) http.Handler {
	adminHandler := mux.NewRouter()
	adminHandler.Use(
		mux.CORSMethodMiddleware(adminHandler),
		authentication(logger, db),
		requestLogging(logger),
		serverInfo(),
	)

	rest.MakeRESTHandler(logger, db, adminHandler)
	makePprofHandler(adminHandler)

	return adminHandler
}

func makePprofHandler(router *mux.Router) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/profile", pprof.Profile)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)

	router.HandleFunc("/debug/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	router.HandleFunc("/debug/pprof/heap", pprof.Handler("allocs").ServeHTTP)
	router.HandleFunc("/debug/pprof/allocs", pprof.Handler("allocs").ServeHTTP)
	router.HandleFunc("/debug/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
	router.HandleFunc("/debug/pprof/block", pprof.Handler("block").ServeHTTP)
	router.HandleFunc("/debug/pprof/mutex", pprof.Handler("mutex").ServeHTTP)
}
