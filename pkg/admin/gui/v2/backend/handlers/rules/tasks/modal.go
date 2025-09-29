package tasks

import (
	"html/template"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	taskModalTemplateFile     = "tasks/task_modal.gohtml"
	taskModalLocalizationFile = "tasks/task_modal.yaml"
	commonTaskTemplatesFile   = "tasks/common_templates.gohtml"
)

type taskModalData struct {
	Text      locale.Dictionary
	Title     string
	Task      *model.Task
	TaskTypes []taskCategory
	FormData  *taskFormData
}

func GetNewTaskModal(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	newTaskTemplate := common.ParseTemplate(taskModalTemplateFile, commonTaskTemplatesFile)
	pageLocalization := locale.ParseLocalizationFile(taskModalLocalizationFile)
	forms := initTasksForms()

	return func(w http.ResponseWriter, r *http.Request) {
		task, tErr := getTask(db, r)
		if common.IsNotFound(tErr) {
			task, tErr = makeTask(db, r)
		}

		if common.SendError(w, logger, tErr) {
			return
		}

		language := locale.GetLanguage(r)
		text := locale.MakeLocalText(language, pageLocalization)
		form := forms[task.Type]

		modalTemplate := template.Must(newTaskTemplate.Clone())
		template.Must(modalTemplate.AddParseTree("taskForm", form.template.Tree))

		if err := modalTemplate.Execute(w, &taskModalData{
			Text:      text,
			Title:     getModalTitle(text, task),
			Task:      task,
			TaskTypes: formFiles,
			FormData: &taskFormData{
				Text: locale.MakeLocalText(language, form.localization),
				Task: task,
			},
		}); err != nil {
			logger.Errorf("failed to render task modal: %v", err)
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}

func getModalTitle(text locale.Dictionary, task *model.Task) string {
	isEdit := task.Type != ""

	switch {
	case task.Chain == model.ChainPre && isEdit:
		return text["editPreTask"]
	case task.Chain == model.ChainPre:
		return text["addPreTask"]
	case task.Chain == model.ChainPost && isEdit:
		return text["editPostTask"]
	case task.Chain == model.ChainPost:
		return text["addPostTask"]
	case task.Chain == model.ChainError && isEdit:
		return text["editErrorTask"]
	case task.Chain == model.ChainError:
		return text["addErrorTask"]
	default:
		return text["addTask"]
	}
}
