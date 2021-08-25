package rest

import (
	"net/http"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"

	"github.com/gorilla/mux"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
)

type addr struct{ target, real string }

func getAddrOverride(r *http.Request) (*addr, error) {
	target, ok := mux.Vars(r)["address"]
	if !ok {
		return nil, notFound("missing target address")
	}

	realAddress := conf.GetIndirection(target)
	if realAddress == "" {
		return nil, notFound("target address does not exist")
	}

	return &addr{target: target, real: realAddress}, nil
}

func getAddressOverride(logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address, err := getAddrOverride(r)
		if handleError(w, logger, err) {
			return
		}
		responseBody := map[string]string{address.target: address.real}
		handleError(w, logger, writeJSON(w, responseBody))
	}
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
