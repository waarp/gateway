package rest

import (
	"code.waarp.fr/waarp/gateway-ng/pkg/log"

	"net/http"
)

type authHandler struct {
	http.ServeMux

	logger *log.Logger
}

func (auth *authHandler) checkHttpBasic(request *http.Request) bool {
	user, pswd, ok := request.BasicAuth()
	if !ok || user != "admin" || pswd != "adminpassword" {
		auth.logger.Debug("Authentication failed")
		return false
	} else {
		auth.logger.Debug("Authentication successful")
		return true
	}
}

func (auth *authHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if origin := request.Header.Get("Origin"); origin != "" {
		writer.Header().Add("Access-Control-Allow-Origin", origin)
		writer.Header().Add("Access-Control-Allow-Methods",
			"POST, GET, PATCH, PUT, DELETE, OPTIONS")
		writer.Header().Add("Access-Control-Allow-Headers",
			"Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")

		if request.Method == http.MethodOptions {
			auth.logger.Debug("Received CORS preflight request")
			writer.WriteHeader(http.StatusOK)
			return
		}
	}

	if !auth.checkHttpBasic(request) {
		writer.Header().Add("WWW-Authenticate", "Basic")
		writer.WriteHeader(http.StatusUnauthorized)
	} else {
		auth.ServeMux.ServeHTTP(writer, request)
	}
}

