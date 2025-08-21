package gui

import (
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/internal"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

//nolint:dupl, gocyclo, cyclop, funlen, gocritic // method for pre-task (gocyclo 20 differents tasks)
func addPreTask(ruleID int, preTasks []*model.Task, db *database.DB, r *http.Request) error {
	var newPreTask model.Task

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newPreTaskType := r.FormValue("addPreTaskType"); newPreTaskType != "" {
		newPreTask.Type = newPreTaskType
	}

	switch newPreTask.Type {
	case TaskCopy:
		newPreTask.Args = taskCOPY(r)
	case TaskCopyRename:
		newPreTask.Args = taskCOPYRENAME(r)
	case TaskExec:
		newPreTask.Args = taskEXEC(r)
	case TaskExecMove:
		newPreTask.Args = taskEXECMOVE(r)
	case TaskExecOutput:
		newPreTask.Args = taskEXECOUTPUT(r)
	case TaskMove:
		newPreTask.Args = taskMOVE(r)
	case TaskMoveRename:
		newPreTask.Args = taskMOVERENAME(r)
	case TaskRename:
		newPreTask.Args = taskRENAME(r)
	case TaskTransfer:
		newPreTask.Args = taskTRANSFER(r)
	case TaskTranscode:
		newPreTask.Args = taskTRANSCODE(r)
	case TaskArchive:
		newPreTask.Args = taskARCHIVE(r)
	case TaskExtract:
		newPreTask.Args = taskEXTRACT(r)
	case TaskIcap:
		newPreTask.Args = taskICAP(r)
	case TaskEncrypt:
		newPreTask.Args = taskENCRYPT(r)
	case TaskDecrypt:
		newPreTask.Args = taskDECRYPT(r)
	case TaskSign:
		newPreTask.Args = taskSIGN(r)
	case TaskVerify:
		newPreTask.Args = taskVERIFY(r)
	case TaskEncryptSign:
		newPreTask.Args = taskENCRYPTandSIGN(r)
	case TaskDecryptVerify:
		newPreTask.Args = taskDECRYPTandVERIFY(r)
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

//nolint:gocyclo, cyclop, funlen, gocritic // 20 differents tasks
func editPreTask(ruleID int, preTasks []*model.Task, db *database.DB, r *http.Request) error {
	var editPreTask model.Task

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	preTaskRank := r.FormValue("editPreTaskRank")

	rank, err := strconv.ParseUint(preTaskRank, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get rank: %w", err)
	}

	if editPreTaskType := r.FormValue("editPreTaskType"); editPreTaskType != "" {
		editPreTask.Type = editPreTaskType
	}

	//nolint:dupl // switch for pre-task
	switch editPreTask.Type {
	case TaskCopy:
		editPreTask.Args = taskCOPY(r)
	case TaskCopyRename:
		editPreTask.Args = taskCOPYRENAME(r)
	case TaskExec:
		editPreTask.Args = taskEXEC(r)
	case TaskExecMove:
		editPreTask.Args = taskEXECMOVE(r)
	case TaskExecOutput:
		editPreTask.Args = taskEXECOUTPUT(r)
	case TaskMove:
		editPreTask.Args = taskMOVE(r)
	case TaskMoveRename:
		editPreTask.Args = taskMOVERENAME(r)
	case TaskRename:
		editPreTask.Args = taskRENAME(r)
	case TaskTransfer:
		editPreTask.Args = taskTRANSFER(r)
	case TaskTranscode:
		editPreTask.Args = taskTRANSCODE(r)
	case TaskArchive:
		editPreTask.Args = taskARCHIVE(r)
	case TaskExtract:
		editPreTask.Args = taskEXTRACT(r)
	case TaskIcap:
		editPreTask.Args = taskICAP(r)
	case TaskEncrypt:
		editPreTask.Args = taskENCRYPT(r)
	case TaskDecrypt:
		editPreTask.Args = taskDECRYPT(r)
	case TaskSign:
		editPreTask.Args = taskSIGN(r)
	case TaskVerify:
		editPreTask.Args = taskVERIFY(r)
	case TaskEncryptSign:
		editPreTask.Args = taskENCRYPTandSIGN(r)
	case TaskDecryptVerify:
		editPreTask.Args = taskDECRYPTandVERIFY(r)
	}

	preTasks[rank] = &editPreTask

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.SetPreTasks(db, rule, preTasks); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

func deletePreTask(ruleID int, preTasks []*model.Task, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	preTaskRank := r.FormValue("deletePreTask")

	rank, err := strconv.ParseUint(preTaskRank, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get rank: %w", err)
	}

	preTasksUpdated := slices.DeleteFunc(preTasks, func(preT *model.Task) bool {
		return int(preT.Rank) == int(rank)
	})

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.SetPreTasks(db, rule, preTasksUpdated); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

func newOrderPreTasks(db *database.DB, r *http.Request, tasks []*model.Task, ruleID int) error {
	newOrderTasks := strings.Split(r.FormValue("newOrderPreTasks"), ",")
	preTasks := make([]*model.Task, len(newOrderTasks))

	for i, str := range newOrderTasks {
		rank, err := strconv.ParseUint(str, 10, 64)
		if err != nil || int(rank) < 0 || int(rank) >= len(tasks) {
			continue
		}
		preTasks[i] = tasks[rank]
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to rule id: %w", err)
	}

	if taskErr := internal.SetPreTasks(db, rule, preTasks); taskErr != nil {
		return fmt.Errorf("failed set pre-tasks: %w", taskErr)
	}

	return nil
}

//nolint:dupl // method for preTasks
func callMethodsPreTasks(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	preTasks []*model.Task, ruleID int,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addPreTaskType") != "" {
		addPreTaskErr := addPreTask(ruleID, preTasks, db, r)
		if addPreTaskErr != nil {
			logger.Errorf("failed to add pre-task: %v", addPreTaskErr)

			return false, addPreTaskErr.Error(), "addPreTaskModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editPreTaskRank") != "" {
		editPreTaskErr := editPreTask(ruleID, preTasks, db, r)
		if editPreTaskErr != nil {
			logger.Errorf("failed to edit pre-task: %v", editPreTaskErr)

			return false, editPreTaskErr.Error(), "editPreTaskModal_" + r.FormValue("editPreTaskRank")
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deletePreTask") != "" {
		deletePreTaskErr := deletePreTask(ruleID, preTasks, db, r)
		if deletePreTaskErr != nil {
			logger.Errorf("failed to delete pre-task: %v", deletePreTaskErr)

			return false, deletePreTaskErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("newOrderPreTasks") != "" {
		orderPreTaskErr := newOrderPreTasks(db, r, preTasks, ruleID)
		if orderPreTaskErr != nil {
			logger.Errorf("failed to set new order pre-task: %v", orderPreTaskErr)

			return false, orderPreTaskErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

//nolint:dupl, gocyclo, cyclop, funlen, gocritic // method for post-task (20 differents tasks)
func addPostTask(ruleID int, postTasks []*model.Task, db *database.DB, r *http.Request) error {
	var newPostTask model.Task

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newPostTaskType := r.FormValue("addPostTaskType"); newPostTaskType != "" {
		newPostTask.Type = newPostTaskType
	}

	//nolint:dupl // switch for post-task
	switch newPostTask.Type {
	case TaskCopy:
		newPostTask.Args = taskCOPY(r)
	case TaskCopyRename:
		newPostTask.Args = taskCOPYRENAME(r)
	case TaskExec:
		newPostTask.Args = taskEXEC(r)
	case TaskExecMove:
		newPostTask.Args = taskEXECMOVE(r)
	case TaskExecOutput:
		newPostTask.Args = taskEXECOUTPUT(r)
	case TaskMove:
		newPostTask.Args = taskMOVE(r)
	case TaskMoveRename:
		newPostTask.Args = taskMOVERENAME(r)
	case TaskRename:
		newPostTask.Args = taskRENAME(r)
	case TaskTransfer:
		newPostTask.Args = taskTRANSFER(r)
	case TaskTranscode:
		newPostTask.Args = taskTRANSCODE(r)
	case TaskArchive:
		newPostTask.Args = taskARCHIVE(r)
	case TaskExtract:
		newPostTask.Args = taskEXTRACT(r)
	case TaskIcap:
		newPostTask.Args = taskICAP(r)
	case TaskEncrypt:
		newPostTask.Args = taskENCRYPT(r)
	case TaskDecrypt:
		newPostTask.Args = taskDECRYPT(r)
	case TaskSign:
		newPostTask.Args = taskSIGN(r)
	case TaskVerify:
		newPostTask.Args = taskVERIFY(r)
	case TaskEncryptSign:
		newPostTask.Args = taskENCRYPTandSIGN(r)
	case TaskDecryptVerify:
		newPostTask.Args = taskDECRYPTandVERIFY(r)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	postTasks = append(postTasks, &newPostTask)
	if err = internal.SetPostTasks(db, rule, postTasks); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

//nolint:gocyclo, cyclop, funlen, gocritic // 20 differents tasks includes
func editPostTask(ruleID int, postTasks []*model.Task, db *database.DB, r *http.Request) error {
	var editPostTask model.Task

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	postTaskRank := r.FormValue("editPostTaskRank")

	rank, err := strconv.ParseUint(postTaskRank, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get rank: %w", err)
	}

	if editPostTaskType := r.FormValue("editPostTaskType"); editPostTaskType != "" {
		editPostTask.Type = editPostTaskType
	}

	//nolint:dupl // switch for post-task
	switch editPostTask.Type {
	case TaskCopy:
		editPostTask.Args = taskCOPY(r)
	case TaskCopyRename:
		editPostTask.Args = taskCOPYRENAME(r)
	case TaskExec:
		editPostTask.Args = taskEXEC(r)
	case TaskExecMove:
		editPostTask.Args = taskEXECMOVE(r)
	case TaskExecOutput:
		editPostTask.Args = taskEXECOUTPUT(r)
	case TaskMove:
		editPostTask.Args = taskMOVE(r)
	case TaskMoveRename:
		editPostTask.Args = taskMOVERENAME(r)
	case TaskRename:
		editPostTask.Args = taskRENAME(r)
	case TaskTransfer:
		editPostTask.Args = taskTRANSFER(r)
	case TaskTranscode:
		editPostTask.Args = taskTRANSCODE(r)
	case TaskArchive:
		editPostTask.Args = taskARCHIVE(r)
	case TaskExtract:
		editPostTask.Args = taskEXTRACT(r)
	case TaskIcap:
		editPostTask.Args = taskICAP(r)
	case TaskEncrypt:
		editPostTask.Args = taskENCRYPT(r)
	case TaskDecrypt:
		editPostTask.Args = taskDECRYPT(r)
	case TaskSign:
		editPostTask.Args = taskSIGN(r)
	case TaskVerify:
		editPostTask.Args = taskVERIFY(r)
	case TaskEncryptSign:
		editPostTask.Args = taskENCRYPTandSIGN(r)
	case TaskDecryptVerify:
		editPostTask.Args = taskDECRYPTandVERIFY(r)
	}

	postTasks[rank] = &editPostTask

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.SetPostTasks(db, rule, postTasks); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

func deletePostTask(ruleID int, postTasks []*model.Task, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	postTaskRank := r.FormValue("deletePostTask")

	rank, err := strconv.ParseUint(postTaskRank, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get rank: %w", err)
	}

	postTasksUpdated := slices.DeleteFunc(postTasks, func(postT *model.Task) bool {
		return int(postT.Rank) == int(rank)
	})

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.SetPostTasks(db, rule, postTasksUpdated); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

func newOrderPostTasks(db *database.DB, r *http.Request, tasks []*model.Task, ruleID int) error {
	newOrderTasks := strings.Split(r.FormValue("newOrderPostTasks"), ",")
	postTasks := make([]*model.Task, len(newOrderTasks))

	for i, str := range newOrderTasks {
		rank, err := strconv.ParseUint(str, 10, 64)
		if err != nil || int(rank) < 0 || int(rank) >= len(tasks) {
			continue
		}
		postTasks[i] = tasks[rank]
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to rule id: %w", err)
	}

	if taskErr := internal.SetPostTasks(db, rule, postTasks); taskErr != nil {
		return fmt.Errorf("failed to set post tasks: %w", taskErr)
	}

	return nil
}

//nolint:dupl // method for postTasks
func callMethodsPostTasks(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	postTasks []*model.Task, ruleID int,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addPostTaskType") != "" {
		addPostTaskErr := addPostTask(ruleID, postTasks, db, r)
		if addPostTaskErr != nil {
			logger.Errorf("failed to add post-task: %v", addPostTaskErr)

			return false, addPostTaskErr.Error(), "addPostTaskModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editPostTaskRank") != "" {
		editPostTaskErr := editPostTask(ruleID, postTasks, db, r)
		if editPostTaskErr != nil {
			logger.Errorf("failed to edit post-task: %v", editPostTaskErr)

			return false, editPostTaskErr.Error(), "editPostTaskModal_" + r.FormValue("editPostTaskRank")
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deletePostTask") != "" {
		deletePostTaskErr := deletePostTask(ruleID, postTasks, db, r)
		if deletePostTaskErr != nil {
			logger.Errorf("failed to delete post-task: %v", deletePostTaskErr)

			return false, deletePostTaskErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("newOrderPostTasks") != "" {
		orderPostTaskErr := newOrderPostTasks(db, r, postTasks, ruleID)
		if orderPostTaskErr != nil {
			logger.Errorf("failed to set new order post-task: %v", orderPostTaskErr)

			return false, orderPostTaskErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

//nolint:dupl, gocyclo, cyclop, funlen, gocritic // method for error-task (20 differents tasks)
func addErrorTask(ruleID int, errorTasks []*model.Task, db *database.DB, r *http.Request) error {
	var newErrorTask model.Task

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	if newErrorTaskType := r.FormValue("addErrorTaskType"); newErrorTaskType != "" {
		newErrorTask.Type = newErrorTaskType
	}

	//nolint:dupl // switch for error-task
	switch newErrorTask.Type {
	case TaskCopy:
		newErrorTask.Args = taskCOPY(r)
	case TaskCopyRename:
		newErrorTask.Args = taskCOPYRENAME(r)
	case TaskExec:
		newErrorTask.Args = taskEXEC(r)
	case TaskExecMove:
		newErrorTask.Args = taskEXECMOVE(r)
	case TaskExecOutput:
		newErrorTask.Args = taskEXECOUTPUT(r)
	case TaskMove:
		newErrorTask.Args = taskMOVE(r)
	case TaskMoveRename:
		newErrorTask.Args = taskMOVERENAME(r)
	case TaskRename:
		newErrorTask.Args = taskRENAME(r)
	case TaskTransfer:
		newErrorTask.Args = taskTRANSFER(r)
	case TaskTranscode:
		newErrorTask.Args = taskTRANSCODE(r)
	case TaskArchive:
		newErrorTask.Args = taskARCHIVE(r)
	case TaskExtract:
		newErrorTask.Args = taskEXTRACT(r)
	case TaskIcap:
		newErrorTask.Args = taskICAP(r)
	case TaskEncrypt:
		newErrorTask.Args = taskENCRYPT(r)
	case TaskDecrypt:
		newErrorTask.Args = taskDECRYPT(r)
	case TaskSign:
		newErrorTask.Args = taskSIGN(r)
	case TaskVerify:
		newErrorTask.Args = taskVERIFY(r)
	case TaskEncryptSign:
		newErrorTask.Args = taskENCRYPTandSIGN(r)
	case TaskDecryptVerify:
		newErrorTask.Args = taskDECRYPTandVERIFY(r)
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	errorTasks = append(errorTasks, &newErrorTask)
	if err = internal.SetErrorTasks(db, rule, errorTasks); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

//nolint:gocyclo, cyclop, funlen, gocritic // 20 differents tasks includes
func editErrorTask(ruleID int, errorTasks []*model.Task, db *database.DB, r *http.Request) error {
	var editErrorTask model.Task

	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	errorTaskRank := r.FormValue("editErrorTaskRank")

	rank, err := strconv.ParseUint(errorTaskRank, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get rank: %w", err)
	}

	if editErrorTaskType := r.FormValue("editErrorTaskType"); editErrorTaskType != "" {
		editErrorTask.Type = editErrorTaskType
	}

	//nolint:dupl // switch for error-task
	switch editErrorTask.Type {
	case TaskCopy:
		editErrorTask.Args = taskCOPY(r)
	case TaskCopyRename:
		editErrorTask.Args = taskCOPYRENAME(r)
	case TaskExec:
		editErrorTask.Args = taskEXEC(r)
	case TaskExecMove:
		editErrorTask.Args = taskEXECMOVE(r)
	case TaskExecOutput:
		editErrorTask.Args = taskEXECOUTPUT(r)
	case TaskMove:
		editErrorTask.Args = taskMOVE(r)
	case TaskMoveRename:
		editErrorTask.Args = taskMOVERENAME(r)
	case TaskRename:
		editErrorTask.Args = taskRENAME(r)
	case TaskTransfer:
		editErrorTask.Args = taskTRANSFER(r)
	case TaskTranscode:
		editErrorTask.Args = taskTRANSCODE(r)
	case TaskArchive:
		editErrorTask.Args = taskARCHIVE(r)
	case TaskExtract:
		editErrorTask.Args = taskEXTRACT(r)
	case TaskIcap:
		editErrorTask.Args = taskICAP(r)
	case TaskEncrypt:
		editErrorTask.Args = taskENCRYPT(r)
	case TaskDecrypt:
		editErrorTask.Args = taskDECRYPT(r)
	case TaskSign:
		editErrorTask.Args = taskSIGN(r)
	case TaskVerify:
		editErrorTask.Args = taskVERIFY(r)
	case TaskEncryptSign:
		editErrorTask.Args = taskENCRYPTandSIGN(r)
	case TaskDecryptVerify:
		editErrorTask.Args = taskDECRYPTandVERIFY(r)
	}

	errorTasks[rank] = &editErrorTask

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.SetErrorTasks(db, rule, errorTasks); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

func deleteErrorTask(ruleID int, errorTasks []*model.Task, db *database.DB, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("failed to parse form: %w", err)
	}

	errorTaskRank := r.FormValue("deleteErrorTask")

	rank, err := strconv.ParseUint(errorTaskRank, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to get rank: %w", err)
	}

	errorTasksUpdated := slices.DeleteFunc(errorTasks, func(errorT *model.Task) bool {
		return int(errorT.Rank) == int(rank)
	})

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to get rule: %w", err)
	}

	if err = internal.SetErrorTasks(db, rule, errorTasksUpdated); err != nil {
		return fmt.Errorf("failed to set task: %w", err)
	}

	return nil
}

func newOrderErrorTasks(db *database.DB, r *http.Request, tasks []*model.Task, ruleID int) error {
	newOrderTasks := strings.Split(r.FormValue("newOrderErrorTasks"), ",")
	errorTasks := make([]*model.Task, len(newOrderTasks))

	for i, str := range newOrderTasks {
		rank, err := strconv.ParseUint(str, 10, 64)
		if err != nil || int(rank) < 0 || int(rank) >= len(tasks) {
			continue
		}
		errorTasks[i] = tasks[rank]
	}

	rule, err := internal.GetRuleByID(db, int64(ruleID))
	if err != nil {
		return fmt.Errorf("failed to rule id: %w", err)
	}

	if taskErr := internal.SetErrorTasks(db, rule, errorTasks); taskErr != nil {
		return fmt.Errorf("failed to set error tasks: %w", taskErr)
	}

	return nil
}

//nolint:dupl // method for errorTasks
func callMethodsErrorTasks(logger *log.Logger, db *database.DB, w http.ResponseWriter, r *http.Request,
	errorTasks []*model.Task, ruleID int,
) (value bool, errMsg, modalOpen string) {
	if r.Method == http.MethodPost && r.FormValue("addErrorTaskType") != "" {
		addErrorTaskErr := addErrorTask(ruleID, errorTasks, db, r)
		if addErrorTaskErr != nil {
			logger.Errorf("failed to add error-task: %v", addErrorTaskErr)

			return false, addErrorTaskErr.Error(), "addErrorTaskModal"
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("editErrorTaskRank") != "" {
		editErrorTaskErr := editErrorTask(ruleID, errorTasks, db, r)
		if editErrorTaskErr != nil {
			logger.Errorf("failed to edit error-task: %v", editErrorTaskErr)

			return false, editErrorTaskErr.Error(), "editErrorTaskModal_" + r.FormValue("editErrorTaskRank")
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("deleteErrorTask") != "" {
		deleteErrorTaskErr := deleteErrorTask(ruleID, errorTasks, db, r)
		if deleteErrorTaskErr != nil {
			logger.Errorf("failed to delete error-task: %v", deleteErrorTaskErr)

			return false, deleteErrorTaskErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	if r.Method == http.MethodPost && r.FormValue("newOrderErrorTasks") != "" {
		orderErrorTaskErr := newOrderErrorTasks(db, r, errorTasks, ruleID)
		if orderErrorTaskErr != nil {
			logger.Errorf("failed to set new order error-task: %v", orderErrorTaskErr)

			return false, orderErrorTaskErr.Error(), ""
		}

		http.Redirect(w, r, fmt.Sprintf("%s?ruleID=%d", r.URL.Path, ruleID), http.StatusSeeOther)

		return true, "", ""
	}

	return false, "", ""
}

//nolint:funlen // is for one page
func tasksTransferRulesPage(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userLanguage := r.Context().Value(ContextLanguageKey)
		tTranslated := //nolint:forcetypeassert //u
			pageTranslated("tasks_transfer_rules_page", userLanguage.(string)) //nolint:errcheck //u

		user, err := GetUserByToken(r, db)
		if err != nil {
			logger.Errorf("Internal error: %v", err)
		}

		myPermission := model.MaskToPerms(user.Permissions)
		var rule *model.Rule
		var id uint64

		ruleID := r.URL.Query().Get("ruleID")
		if ruleID != "" {
			id, err = strconv.ParseUint(ruleID, 10, 64)
			if err != nil {
				logger.Errorf("failed to convert id to int: %v", err)
			}

			rule, err = internal.GetRuleByID(db, int64(id))
			if err != nil {
				logger.Errorf("failed to get id: %v", err)
			}
		}

		listKeyname(db)

		preTasks, err := internal.ListPreTasks(db, rule)
		if err != nil {
			return
		}

		postTasks, err := internal.ListPostTasks(db, rule)
		if err != nil {
			return
		}

		errorTasks, err := internal.ListErrorTasks(db, rule)
		if err != nil {
			return
		}
		var errMsg, modalOpen string

		if handled, em, mo := callMethodsPreTasks(logger, db, w, r, preTasks, int(rule.ID)); handled {
			return
		} else if em != "" {
			errMsg, modalOpen = em, mo
		}

		if handled, em, mo := callMethodsPostTasks(logger, db, w, r, postTasks, int(rule.ID)); handled {
			return
		} else if em != "" {
			errMsg, modalOpen = em, mo
		}

		if handled, em, mo := callMethodsErrorTasks(logger, db, w, r, errorTasks, int(rule.ID)); handled {
			return
		} else if em != "" {
			errMsg, modalOpen = em, mo
		}

		fmt.Println("EncryptKeyTypes:", EncryptKeyTypes)
		fmt.Println("DecryptKeyTypes:", DecryptKeyTypes)
		fmt.Println("SignKeyTypes:", SignKeyTypes)
		fmt.Println("VerifyKeyTypes:", VerifyKeyTypes)
		fmt.Println("EncryptSignKeyTypes:", EncryptSignKeyTypes)
		fmt.Println("DecryptVerifyKeyTypes:", DecryptVerifyKeyTypes)

		if tmplErr := tasksTransferRulesTemplate.ExecuteTemplate(w, "tasks_transfer_rules_page", map[string]any{
			"myPermission":          myPermission,
			"tab":                   tTranslated,
			"username":              user.Username,
			"language":              userLanguage,
			"rule":                  rule,
			"taskTypes":             TaskTypes,
			"preTasks":              preTasks,
			"postTasks":             postTasks,
			"errorTasks":            errorTasks,
			"TranscodeFormats":      TranscodeFormats,
			"ArchiveExtensions":     ArchiveExtensions,
			"EncryptMethods":        EncryptMethods,
			"EncryptKeyTypes":       EncryptKeyTypes,
			"DecryptMethods":        DecryptMethods,
			"DecryptKeyTypes":       DecryptKeyTypes,
			"SignMethods":           SignMethods,
			"SignKeyTypes":          SignKeyTypes,
			"VerifyMethods":         VerifyMethods,
			"VerifyKeyTypes":        VerifyKeyTypes,
			"EncryptSignMethods":    EncryptSignMethods,
			"EncryptSignKeyTypes":   EncryptSignKeyTypes,
			"DecryptVerifyMethods":  DecryptVerifyMethods,
			"DecryptVerifyKeyTypes": DecryptVerifyKeyTypes,
			"IcapOnErrorOptions":    IcapOnErrorOptions,
			"CompressionLevelList":  CompressionLevelList,
			"listAesKey":            ListAesKeyName,
			"listHmacKey":           ListHmacKeyName,
			"listPgpPubKey":         ListPgpPubKeyName,
			"listPgpPrivKey":        ListPgpPrivKeyName,
			"errMsg":                errMsg,
			"modalOpen":             modalOpen,
			"hasRuleID":             true,
		}); tmplErr != nil {
			logger.Errorf("render tasks_transfer_rules_page: %v", tmplErr)
			http.Error(w, "Internal error", http.StatusInternalServerError)
		}
	}
}
