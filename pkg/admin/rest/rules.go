package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// restRuleToDB transforms the JSON transfer rule into its database equivalent.
func restRuleToDB(rule *api.InRule, logger *log.Logger) (*model.Rule, error) {
	if !rule.IsSend.Valid {
		return nil, badRequest("missing rule direction")
	}

	local := rule.LocalDir.Value
	remote := rule.RemoteDir.Value
	tmp := rule.TmpLocalRcvDir.Value

	if rule.InPath.Valid {
		logger.Warning("JSON field 'inPath' is deprecated, use 'localDir' & 'remoteDir' instead")

		if rule.IsSend.Value && !rule.RemoteDir.Valid {
			remote = utils.DenormalizePath(rule.InPath.Value)
		} else if !rule.LocalDir.Valid {
			local = utils.DenormalizePath(rule.InPath.Value)
		}
	}

	if rule.OutPath.Valid {
		logger.Warning("JSON field 'outPath' is deprecated, use 'localDir' & 'remoteDir' instead")

		if rule.IsSend.Value && !rule.LocalDir.Valid {
			local = utils.DenormalizePath(rule.OutPath.Value)
		} else if !rule.RemoteDir.Valid {
			remote = utils.DenormalizePath(rule.OutPath.Value)
		}
	}

	if rule.WorkPath.Valid {
		logger.Warning("JSON field 'workPath' is deprecated, use 'localTmpDir' instead")

		if !rule.TmpLocalRcvDir.Valid {
			tmp = utils.DenormalizePath(rule.WorkPath.Value)
		}
	}

	dbRule := &model.Rule{
		LocalDir:       local,
		RemoteDir:      remote,
		TmpLocalRcvDir: tmp,
	}

	setIfValid(&dbRule.Name, rule.Name)
	setIfValid(&dbRule.IsSend, rule.IsSend)
	setIfValid(&dbRule.Comment, rule.Comment)
	setIfValid(&dbRule.Path, rule.Path)

	return dbRule, nil
}

// DBRuleToREST transforms the given database transfer rule into its JSON equivalent.
func DBRuleToREST(db *database.DB, dbRule *model.Rule) (*api.OutRule, error) {
	access, accErr := makeRuleAccess(db, dbRule)
	if accErr != nil {
		return nil, accErr
	}

	in := utils.NormalizePath(dbRule.LocalDir)
	out := utils.NormalizePath(dbRule.RemoteDir)

	if dbRule.IsSend {
		in = utils.NormalizePath(dbRule.RemoteDir)
		out = utils.NormalizePath(dbRule.LocalDir)
	}

	work := utils.NormalizePath(dbRule.TmpLocalRcvDir)

	rule := &api.OutRule{
		Name:           dbRule.Name,
		Comment:        dbRule.Comment,
		IsSend:         dbRule.IsSend,
		Path:           dbRule.Path,
		InPath:         in,
		OutPath:        out,
		WorkPath:       work,
		LocalDir:       dbRule.LocalDir,
		RemoteDir:      dbRule.RemoteDir,
		TmpLocalRcvDir: dbRule.TmpLocalRcvDir,
		Authorized:     *access,
	}
	if err := doListTasks(db, rule, dbRule.ID); err != nil {
		return nil, err
	}

	return rule, nil
}

// DBRulesToREST transforms the given list of database transfer rules into its JSON
// equivalent.
func DBRulesToREST(db *database.DB, dbRules []*model.Rule) ([]*api.OutRule, error) {
	restRules := make([]*api.OutRule, len(dbRules))

	for i, dbRule := range dbRules {
		var err error
		if restRules[i], err = DBRuleToREST(db, dbRule); err != nil {
			return nil, err
		}
	}

	return restRules, nil
}

func ruleDirection(rule *model.Rule) string {
	if rule.IsSend {
		return "send"
	}

	return "receive"
}

func retrieveDBRule(r *http.Request, db *database.DB) (*model.Rule, error) {
	ruleName, ok := mux.Vars(r)["rule"]
	if !ok {
		return nil, notFound("missing rule name")
	}

	direction, ok := mux.Vars(r)["direction"]
	if !ok {
		return nil, notFound("missing rule direction")
	}

	var rule model.Rule
	if err := db.Get(&rule, "name=? AND is_send=?", ruleName,
		direction == "send").Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("%s rule %q not found", direction, ruleName)
		}

		return nil, fmt.Errorf("failed to retrieve rule %q: %w", ruleName, err)
	}

	return &rule, nil
}

func addRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restRule api.InRule
		if err := readJSON(r, &restRule); handleError(w, logger, err) {
			return
		}

		dbRule, convErr := restRuleToDB(&restRule, logger)
		if handleError(w, logger, convErr) {
			return
		}

		transErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Insert(dbRule).Run(); err != nil {
				return fmt.Errorf("failed to insert rule: %w", err)
			}

			return doTaskUpdate(ses, &restRule, dbRule.ID, true)
		})
		if handleError(w, logger, transErr) {
			return
		}

		w.Header().Set("Location", location(r.URL, dbRule.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func getRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbRule, getErr := retrieveDBRule(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restRule, convErr := DBRuleToREST(db, dbRule)
		if handleError(w, logger, convErr) {
			return
		}

		handleError(w, logger, writeJSON(w, restRule))
	}
}

func listRules(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{col: "name", asc: true},
		"name+":   order{col: "name", asc: true},
		"name-":   order{col: "name", asc: false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbRules model.Rules

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbRules)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restRules, convErr := DBRulesToREST(db, dbRules)
		if handleError(w, logger, convErr) {
			return
		}

		response := map[string][]*api.OutRule{"rules": restRules}
		handleError(w, logger, writeJSON(w, response))
	}
}

func updateRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldRule, getErr := retrieveDBRule(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restRule := &api.InRule{
			Name: asNullable(oldRule.Name),

			Comment:        asNullable(oldRule.Comment),
			IsSend:         asNullableBool(oldRule.IsSend),
			Path:           asNullable(oldRule.Path),
			LocalDir:       asNullable(oldRule.LocalDir),
			RemoteDir:      asNullable(oldRule.RemoteDir),
			TmpLocalRcvDir: asNullable(oldRule.TmpLocalRcvDir),
			PreTasks:       nil,
			PostTasks:      nil,
			ErrorTasks:     nil,
		}
		if err := readJSON(r, restRule); handleError(w, logger, err) {
			return
		}

		dbRule, convErr := restRuleToDB(restRule, logger)
		if handleError(w, logger, convErr) {
			return
		}

		dbRule.ID = oldRule.ID

		transErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Update(dbRule).Run(); err != nil {
				return fmt.Errorf("failed to update rule: %w", err)
			}

			return doTaskUpdate(ses, restRule, oldRule.ID, false)
		})
		if handleError(w, logger, transErr) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbRule.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func replaceRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldRule, getErr := retrieveDBRule(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restRule := &api.InRule{}
		if err2 := readJSON(r, restRule); handleError(w, logger, err2) {
			return
		}

		dbRule, convErr := restRuleToDB(restRule, logger)
		if handleError(w, logger, convErr) {
			return
		}

		dbRule.ID = oldRule.ID

		transErr := db.Transaction(func(ses *database.Session) error {
			if err := ses.Update(dbRule).Run(); handleError(w, logger, err) {
				return fmt.Errorf("failed to update rule: %w", err)
			}

			return doTaskUpdate(ses, restRule, oldRule.ID, true)
		})
		if handleError(w, logger, transErr) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, restRule.Name.Value))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbRule, getErr := retrieveDBRule(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(dbRule).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func allowAllRule(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbRule, getErr := retrieveDBRule(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.DeleteAll(&model.RuleAccess{}).Where("rule_id=?", dbRule.ID).
			Run(); handleError(w, logger, err) {
			return
		}

		fmt.Fprintf(w, "Usage of the %s rule %q is now unrestricted.",
			ruleDirection(dbRule), dbRule.Name)
	}
}
