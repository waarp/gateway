package tasks

import (
	"net/http"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/constants"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/handlers/rules"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	tasksPageTemplateFile     = "tasks/tasks.gohtml"
	tasksPageLocalizationFile = "tasks/tasks.yaml"
)

type tasksData struct {
	Rule       *model.Rule
	PreTasks   *chainData
	PostTasks  *chainData
	ErrorTasks *chainData

	Text locale.Dictionary
}

type chainData struct {
	Rule       *model.Rule
	Chain      model.Chain
	Tasks      model.Tasks
	UserRights *common.Permissions
	Text       locale.Dictionary
}

func GetTasksPage(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	pageTemplate := common.InitBasePageTemplate(tasksPageTemplateFile)
	baseLocalization := common.InitBaseLocalization()
	tasksLocalization := locale.ParseLocalizationFile(tasksPageLocalizationFile)

	return func(w http.ResponseWriter, r *http.Request) {
		language := locale.GetLanguage(r)
		user := common.GetUser(r)

		rule, rErr := rules.GetRule(db, r)
		if common.SendError(w, logger, rErr) {
			return
		}

		preTasks, postTasks, errorTasks, tasksErr := listTasks(db, rule)
		if common.SendError(w, logger, tasksErr) {
			return
		}

		text := locale.MakeLocalText(language, tasksLocalization)
		contentData := &tasksData{
			Rule: rule,
			PreTasks: &chainData{
				Rule:       rule,
				Chain:      model.ChainPre,
				Tasks:      preTasks,
				UserRights: common.ParsePermissions(user.Permissions),
				Text:       text,
			},
			PostTasks: &chainData{
				Rule:       rule,
				Chain:      model.ChainPost,
				Tasks:      postTasks,
				UserRights: common.ParsePermissions(user.Permissions),
				Text:       text,
			},
			ErrorTasks: &chainData{
				Rule:       rule,
				Chain:      model.ChainError,
				Tasks:      errorTasks,
				UserRights: common.ParsePermissions(user.Permissions),
				Text:       text,
			},
			Text: text,
		}

		pageData := common.MakeBasePageData(user, language,
			constants.SidebarSectionTreatments, constants.SidebarMenuRules,
			baseLocalization, contentData)

		if err := pageTemplate.Execute(w, pageData); err != nil {
			logger.Errorf("failed to render tasks page: %v", err)
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}

func PostTask(db database.Access, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task, tErr := parseTaskForm(db, r)
		if common.SendError(w, logger, tErr) {
			return
		}

		if err := insertTask(db, task); common.SendError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}

func DeleteTask(db database.Access, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task, tErr := getTask(db, r)
		if common.SendError(w, logger, tErr) {
			return
		}

		if err := deleteTask(db, task); common.SendError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func ReorderTasks(db database.Access, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, bErr := common.ReadBody[reorderBody](r)
		if common.SendError(w, logger, bErr) {
			return
		}

		tasks, tErr := listTaskChain(db, body.RuleID, body.Chain)
		if common.SendError(w, logger, tErr) {
			return
		}

		if err := reorderTasks(db, tasks, body); common.SendError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusCreated)
	}
}
