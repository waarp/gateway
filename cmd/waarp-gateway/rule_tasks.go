package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type ruleTasksCommand struct {
	List   ruleTasksListCommand   `command:"list" description:"List the tasks assigned to a rule"`
	Change ruleTasksChangeCommand `command:"change" description:"Change the tasks assigned to a rule"`
}

func displayRuleTask(task model.Task) {
	w := getColorable()

	fmt.Fprintf(w, "  \033[37;1mCommand:\033[0m %s\n", task.Type)
	fmt.Fprintf(w, "    \033[37;1mArguments:\033[0m %s\n", task.Args)
}

// ######################## CHANGE ##########################

type ruleTasksChangeCommand struct {
	PreTasks   string `long:"pre" description:"The list of pre-transfer tasks in JSON format"`
	PostTasks  string `long:"post" description:"The list of post-transfer tasks in JSON format"`
	ErrorTasks string `long:"error" description:"The list of transfer error tasks in JSON format"`
}

func (r *ruleTasksChangeCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing rule ID")
	}

	var preTasks, postTasks, errTasks []model.Task
	if err := json.Unmarshal([]byte(r.PreTasks), &preTasks); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(r.PostTasks), &postTasks); err != nil {
		return err
	}
	if err := json.Unmarshal([]byte(r.ErrorTasks), &errTasks); err != nil {
		return err
	}

	tasks := map[string]interface{}{}
	tasks["preTasks"] = preTasks
	tasks["postTasks"] = postTasks
	tasks["errorTasks"] = errTasks

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.RulesPath + "/" + args[0] + admin.RuleTasksPath

	loc, err := sendBean(tasks, conn, http.MethodPut)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The task chains of rule n°\033[33m%s\033[0m were successfully "+
		"changed. The rule's chains can be consulted at the address: %s", args[0], loc)

	return nil
}

// ######################## LIST ##########################

type ruleTasksListCommand struct{}

func (r *ruleTasksListCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing rule ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.RulesPath + "/" + args[0] + admin.RuleTasksPath

	res := map[string][]model.Task{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()

	if preTasks := res["preTasks"]; len(preTasks) > 0 {
		fmt.Fprintf(w, "\033[33mPre tasks:\033[0m\n")
		for _, task := range preTasks {
			displayRuleTask(task)
		}
	} else {
		fmt.Fprintf(w, "\033[33mNo pre tasks.\033[0m\n")
	}

	if postTasks := res["postTasks"]; len(postTasks) > 0 {
		fmt.Fprintf(w, "\033[33mPost tasks:\033[0m\n")
		for _, task := range postTasks {
			displayRuleTask(task)
		}
	} else {
		fmt.Fprintf(w, "\033[33mNo post tasks.\033[0m\n")
	}

	if errorTasks := res["errorTasks"]; len(errorTasks) > 0 {
		fmt.Fprintf(w, "\033[33mError tasks:\033[0m\n")
		for _, task := range errorTasks {
			displayRuleTask(task)
		}
	} else {
		fmt.Fprintf(w, "\033[33mNo error tasks.\033[0m\n")
	}

	return nil
}
