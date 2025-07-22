package gui

import (
	"fmt"
	"net/http"
	"strconv"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func addPreTask(ruleID int, preTasks []*model.Task, db *database.DB, r *http.Request) error {
	var newPreTask model.Task

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newPreTaskType := r.FormValue("addPreTaskType"); newPreTaskType != "" {
		newPreTask.Type = newPreTaskType
	}

	switch newPreTask.Type {
	case "COPY":
		newPreTask.Args = taskCOPY(r)
	case "COPYRENAME":
		newPreTask.Args = taskCOPYRENAME(r)
	case "EXEC":
		newPreTask.Args = taskEXEC(r)
	case "EXECMOVE":
		newPreTask.Args = taskEXECMOVE(r)
	case "EXECOUTPUT":
		newPreTask.Args = taskEXECOUTPUT(r)
	case "MOVE":
		newPreTask.Args = taskMOVE(r)
	case "MOVERENAME":
		newPreTask.Args = taskMOVERENAME(r)
	case "RENAME":
		newPreTask.Args = taskRENAME(r)
	case "TRANSFER":
		newPreTask.Args = taskTRANSFER(r)
	case "TRANSCODE":
		newPreTask.Args = taskTRANSCODE(r)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	preTasks = append(preTasks, &newPreTask)
	if err = internal.SetPreTasks(db, rule, preTasks); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

func callMethodsTasksTransferRules(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	preTasks []*model.Task, ruleID int,
) (bool, string, string) {
	if r.Method == http.MethodPost && r.FormValue("addPreTaskType") != "" {
		addPreTaskErr := addPreTask(ruleID, preTasks, db, r)
		if addPreTaskErr != nil {
			logger.Error("failed to add pre-task: %v", addPreTaskErr)

			return false, addPreTaskErr.Error(), "addPreTaskModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID),
			http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

func tasksTransferRulesPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("tasks_transfer_rules_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Error("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var rule *model.Rule
		var id int

		ruleID := r.URL.Query().Get("ruleID")
		if ruleID != "" {
			id, err = strconv.Atoi(ruleID)
			if err != nil {
				logger.Error("failed to convert id to int: %v", err)
			}

			rule, err = internal.GetRuleByID(db, int64(id))
			if err != nil {
				logger.Error("failed to get id: %v", err)
			}
		}

		preTasks, err := internal.ListPreTasks(db, rule)
		if err != nil {
			return
		}

		value, errMsg, modalOpen := callMethodsTasksTransferRules(logger, db, w, r, preTasks, int(rule.ID))
		if value {
			return
		}

		if err := tasksTransferRulesTemplate.ExecuteTemplate(w, "tasks_transfer_rules_page", map[string]any{
			"myPermission":  myPermission,
			"tab":           tTranslated,
			"username":      user.Username,
			"language":      userLanguage,
			"rule":          rule,
			"taskTypes":     TaskTypes,
			"preTasks":      preTasks,
			"TranscodeList": SupportedTranscode,
			"errMsg":        errMsg,
			"modalOpen":     modalOpen,
			"hasRuleID":     true,
		}); err != nil {
			logger.Error("render tasks_transfer_rules_page: %v", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
