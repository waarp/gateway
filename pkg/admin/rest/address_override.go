package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
)

type addrOverride struct{ target, real string }

func getAddrOverride(db *database.DB, r *http.Request) (*addrOverride, error) {
	target, ok := mux.Vars(r)["address"]
	if !ok {
		return nil, notFound("missing target address")
	}

	realAddress := db.Config.Overrides.GetIndirection(target)
	if realAddress == "" {
		return nil, notFound("target address does not exist")
	}

	return &addrOverride{target: target, real: realAddress}, nil
}

func listAddressOverrides(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		indirections := db.Config.Overrides.GetAllIndirections()
		handleError(w, logger, writeJSON(w, indirections))
	}
}

func addAddressOverride(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		indirections := map[string]string{}
		if err := readJSON(r, &indirections); handleError(w, logger, err) {
			return
		}

		for target, realAddr := range indirections {
			if err := db.Config.Overrides.AddIndirection(target, realAddr); handleError(w, logger, err) {
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func deleteAddressOverride(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		address, addrErr := getAddrOverride(db, r)
		if handleError(w, logger, addrErr) {
			return
		}

		if err := db.Config.Overrides.RemoveIndirection(address.target); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
