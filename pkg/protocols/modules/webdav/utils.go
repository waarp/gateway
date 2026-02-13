package webdav

import (
	"net/http"
)

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Add("WWW-Authenticate", "Basic")
	w.Header().Add("WWW-Authenticate", `Transport mode="tls-client-certificate"`)
	http.Error(w, msg, http.StatusUnauthorized)
}
