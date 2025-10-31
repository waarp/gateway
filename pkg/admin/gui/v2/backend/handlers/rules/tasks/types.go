package tasks

import (
	"html/template"
	"net/http"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/locale"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	tasknames "code.waarp.fr/apps/gateway/gateway/pkg/tasks"
)

type taskInfo struct {
	Name         string
	template     string
	localization string
}

type taskCategory struct {
	Name  string
	Tasks []taskInfo
}

//nolint:gochecknoglobals,lll //arrays cannot be constants
var formFiles = []taskCategory{
	{
		Name: "fileOperations",
		Tasks: []taskInfo{
			{tasknames.Copy, "tasks/copy/copy.gohtml", "tasks/copy/copy.yaml"},
			{tasknames.CopyRename, "tasks/copy/copy.gohtml", "tasks/copy/copy_rename.yaml"},
			{tasknames.Move, "tasks/move/move.gohtml", "tasks/move/move.yaml"},
			{tasknames.MoveRename, "tasks/move/move.gohtml", "tasks/move/moverename.yaml"},
			{tasknames.Delete, "tasks/delete/delete.gohtml", "tasks/delete/delete.yaml"},
			{tasknames.RemoteDelete, "tasks/remote-delete/remote-delete.gohtml", "tasks/remote-delete/remote-delete.yaml"},
		},
	}, {
		Name: "executables",
		Tasks: []taskInfo{
			{tasknames.Exec, "tasks/exec/exec.gohtml", "tasks/exec/exec.yaml"},
			{tasknames.ExecMove, "tasks/exec/exec.gohtml", "tasks/exec/execmove.yaml"},
			{tasknames.ExecOutput, "tasks/exec/exec.gohtml", "tasks/exec/execoutput.yaml"},
		},
	}, {
		Name: "transfer",
		Tasks: []taskInfo{
			{tasknames.Rename, "tasks/rename/rename.gohtml", "tasks/rename/rename.yaml"},
			{tasknames.Transfer, "tasks/transfer/transfer.gohtml", "tasks/transfer/transfer.yaml"},
			{tasknames.Preregister, "tasks/preregister/preregister.gohtml", "tasks/preregister/preregister.yaml"},
		},
	}, {
		Name: "fileAlterations",
		Tasks: []taskInfo{
			{tasknames.Transcode, "tasks/transcode/transcode.gohtml", "tasks/transcode/transcode.yaml"},
			{tasknames.ChangeNewline, "tasks/chnewline/chnewline.gohtml", "tasks/chnewline/chnewline.yaml"},
		},
	}, {
		Name: "archiving",
		Tasks: []taskInfo{
			{tasknames.Archive, "tasks/archive/archive.gohtml", "tasks/archive/archive.yaml"},
			{tasknames.Extract, "tasks/extract/extract.gohtml", "tasks/extract/extract.yaml"},
		},
	}, {
		Name: "network",
		Tasks: []taskInfo{
			{tasknames.Icap, "tasks/icap/icap.gohtml", "tasks/icap/icap.yaml"},
			{tasknames.Email, "tasks/email/email.gohtml", "tasks/email/email.yaml"},
		},
	}, {
		Name: "crypto",
		Tasks: []taskInfo{
			{tasknames.Encrypt, "tasks/encrypt/encrypt.gohtml", "tasks/encrypt/encrypt.yaml"},
			{tasknames.Decrypt, "tasks/decrypt/decrypt.gohtml", "tasks/decrypt/decrypt.yaml"},
			{tasknames.Sign, "tasks/sign/sign.gohtml", "tasks/sign/sign.yaml"},
			{tasknames.Verify, "tasks/verify/verify.gohtml", "tasks/verify/verify.yaml"},
			{tasknames.EncryptAndSign, "tasks/encrypt-sign/encrypt-sign.gohtml", "tasks/encrypt-sign/encrypt-sign.yaml"},
			{tasknames.DecryptAndVerify, "tasks/decrypt-verify/decrypt-verify.gohtml", "tasks/decrypt-verify/decrypt-verify.yaml"},
		},
	},
}

type taskForm struct {
	template     *template.Template
	localization locale.LocalizationData
}

func initTasksForms() map[string]*taskForm {
	forms := map[string]*taskForm{
		"": {
			template:     template.Must(template.New("taskForm").Parse("")),
			localization: locale.LocalizationData{},
		},
	}

	for _, category := range formFiles {
		for _, task := range category.Tasks {
			forms[task.Name] = &taskForm{
				template:     common.ParseTemplate(task.template, commonTaskTemplatesFile),
				localization: locale.ParseLocalizationFile(task.localization),
			}
		}
	}

	return forms
}

type taskFormData struct {
	Text locale.Dictionary
	Task *model.Task
}

func GetNewTaskForm(db database.ReadAccess, logger *log.Logger) http.HandlerFunc {
	forms := initTasksForms()

	return func(w http.ResponseWriter, r *http.Request) {
		language := locale.GetLanguage(r)

		form, fErr := makeTaskForm(r, forms)
		if common.SendError(w, logger, fErr) {
			return
		}

		task, tErr := makeTask(db, r)
		if common.SendError(w, logger, tErr) {
			return
		}

		if err := form.template.Execute(w, &taskFormData{
			Text: locale.MakeLocalText(language, form.localization),
			Task: task,
		}); err != nil {
			logger.Errorf("failed to render task form: %v", err)
			http.Error(w, "Failed to render page", http.StatusInternalServerError)
		}
	}
}
