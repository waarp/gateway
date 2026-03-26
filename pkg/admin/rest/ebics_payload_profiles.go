package rest

import (
	"database/sql"
	"fmt"
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func ebicsPayloadProfileRESTToDB(
	db database.ReadAccess,
	input *api.InEbicsPayloadProfile,
) (*model.EbicsPayloadProfile, error) {
	profile := &model.EbicsPayloadProfile{
		Name:                   input.Name,
		Label:                  input.Label,
		Description:            input.Description,
		OrderType:              input.OrderType,
		Direction:              input.Direction,
		ServiceName:            input.ServiceName,
		ServiceOption:          input.ServiceOption,
		Scope:                  input.Scope,
		MsgName:                input.MsgName,
		ContainerType:          input.ContainerType,
		DefaultTargetDirectory: input.DefaultTargetDirectory,
		RequiresDeclaredAmount: input.RequiresDeclaredAmount,
		DefaultCurrency:        input.DefaultCurrency,
		AllowedExtensionsList:  input.AllowedExtensions,
		FilenamePattern:        input.FilenamePattern,
		MetadataMap:            input.Metadata,
	}

	if input.StrictContractCheck != nil {
		profile.StrictContractCheck = *input.StrictContractCheck
	}

	if input.IsEnabled != nil {
		profile.IsEnabled = *input.IsEnabled
	}

	if input.DefaultRule != "" {
		rule := &model.Rule{}
		if err := db.Get(rule, "name=?", input.DefaultRule).Owner().Run(); err != nil {
			if database.IsNotFound(err) {
				return nil, badRequestf("Gateway rule %q not found", input.DefaultRule)
			}

			return nil, fmt.Errorf(
				"failed to retrieve default Gateway rule %q: %w",
				input.DefaultRule,
				err,
			)
		}

		profile.DefaultRuleID = sql.NullInt64{Int64: rule.ID, Valid: true}
	}

	return profile, nil
}

func addEbicsPayloadProfile(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		in := &api.InEbicsPayloadProfile{}
		if err := readJSON(r, in); handleError(w, logger, err) {
			return
		}

		profile, err := ebicsPayloadProfileRESTToDB(db, in)
		if handleError(w, logger, err) {
			return
		}

		if err = db.Insert(profile).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, profile.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getEbicsPayloadProfile(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, err := getDBEbicsPayloadProfile(r, db)
		if handleError(w, logger, err) {
			return
		}

		out, err := DBEbicsPayloadProfileToREST(db, profile)
		if handleError(w, logger, err) {
			return
		}

		handleError(w, logger, writeJSON(w, out))
	}
}

//nolint:dupl // list handlers stay explicit per EBICS resource
func listEbicsPayloadProfiles(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var profiles model.EbicsPayloadProfiles

		query, err := parseSelectQuery(r, db, validSorting, &profiles)
		if handleError(w, logger, err) {
			return
		}

		query.Owner()
		if err = query.Run(); handleError(w, logger, err) {
			return
		}

		out := make([]*api.OutEbicsPayloadProfile, len(profiles))
		for i, profile := range profiles {
			out[i], err = DBEbicsPayloadProfileToREST(db, profile)
			if handleError(w, logger, err) {
				return
			}
		}

		handleError(w, logger, writeJSON(w, map[string]any{"payloadProfiles": out}))
	}
}

func updateEbicsPayloadProfile(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := getDBEbicsPayloadProfile(r, db)
		if handleError(w, logger, err) {
			return
		}

		currentOut, err := DBEbicsPayloadProfileToREST(db, current)
		if handleError(w, logger, err) {
			return
		}

		strict := current.StrictContractCheck
		enabled := current.IsEnabled
		in := &api.InEbicsPayloadProfile{
			Name:                   current.Name,
			Label:                  current.Label,
			Description:            current.Description,
			OrderType:              current.OrderType,
			Direction:              current.Direction,
			ServiceName:            current.ServiceName,
			ServiceOption:          current.ServiceOption,
			Scope:                  current.Scope,
			MsgName:                current.MsgName,
			ContainerType:          current.ContainerType,
			DefaultRule:            currentOut.DefaultRule,
			DefaultTargetDirectory: current.DefaultTargetDirectory,
			RequiresDeclaredAmount: current.RequiresDeclaredAmount,
			DefaultCurrency:        current.DefaultCurrency,
			AllowedExtensions:      current.AllowedExtensionsList,
			FilenamePattern:        current.FilenamePattern,
			StrictContractCheck:    &strict,
			IsEnabled:              &enabled,
			Metadata:               current.MetadataMap,
		}
		if err = readJSON(r, in); handleError(w, logger, err) {
			return
		}

		updated, err := ebicsPayloadProfileRESTToDB(db, in)
		if handleError(w, logger, err) {
			return
		}

		updated.ID = current.ID

		if err = db.Update(updated).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, updated.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceEbicsPayloadProfile(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		current, err := getDBEbicsPayloadProfile(r, db)
		if handleError(w, logger, err) {
			return
		}

		in := &api.InEbicsPayloadProfile{}
		if err = readJSON(r, in); handleError(w, logger, err) {
			return
		}

		replacement, err := ebicsPayloadProfileRESTToDB(db, in)
		if handleError(w, logger, err) {
			return
		}

		replacement.ID = current.ID

		if err = db.Update(replacement).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, replacement.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteEbicsPayloadProfile(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		profile, err := getDBEbicsPayloadProfile(r, db)
		if handleError(w, logger, err) {
			return
		}

		if err = db.Delete(profile).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
