package rest

import (
	"net/http"

	"code.waarp.fr/lib/log"
	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
)

type addrOverride struct{ target, real string }

func getAddrOverride(r *http.Request) (*addrOverride, error) {
	target, ok := mux.Vars(r)["address"]
	if !ok {
		return nil, notFound("missing target address")
	}

	realAddress := conf.GetIndirection(target)
	if realAddress == "" {
		return nil, notFound("target address does not exist")
	}

	return &addrOverride{target: target, real: realAddress}, nil
}

func listAddressOverrides(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		indirections := conf.GetAllIndirections()
		handleError(w, logger, writeJSON(w, indirections))
	}
}

func addAddressOverride(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		indirections := map[string]string{}
		if err := readJSON(r, &indirections); handleError(w, logger, err) {
			return
		}

		for target, realAddr := range indirections {
			if err := conf.AddIndirection(target, realAddr); handleError(w, logger, err) {
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func deleteAddressOverride(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address, err := getAddrOverride(r)
		if handleError(w, logger, err) {
			return
		}

		if err := conf.RemoveIndirection(address.target); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
