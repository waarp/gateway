package main

import (
	"fmt"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type ruleCommand struct {
	Get    ruleGetCommand    `command:"get" description:"Retrieve a rule's information"`
	Add    ruleAddCommand    `command:"add" description:"Add a new rule"`
	Delete ruleDeleteCommand `command:"delete" description:"Delete a rule"`
	List   ruleListCommand   `command:"list" description:"List the known rules"`
	Access ruleAccessCommand `command:"access" description:"Manage the permissions for a rule"`
	Tasks  ruleTasksCommand  `command:"tasks" description:"Manage the rule's task chain"`
}

func displayRule(rule rest.OutRule) {
	w := getColorable()

	fmt.Fprintf(w, "\033[37;1;4mRule n°%v:\033[0m\n", rule.ID)
	fmt.Fprintf(w, "      \033[37mName:\033[0m \033[37;1m%s\033[0m\n", rule.Name)
	fmt.Fprintf(w, "   \033[37mComment:\033[0m \033[33m%s\033[0m\n", rule.Comment)
	if rule.IsSend {
		fmt.Fprint(w, " \033[37mDirection:\033[0m \033[90mSEND\033[0m\n")
	} else {
		fmt.Fprint(w, " \033[37mDirection:\033[0m \033[90mRECEIVE\033[0m\n")
	}
	fmt.Fprintf(w, "      \033[37mPath:\033[0m \033[90m%v\033[0m\n", rule.Path)
}

// ######################## GET ##########################

type ruleGetCommand struct{}

func (r *ruleGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing rule ID")
	}

	res := rest.OutRule{}
	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RulesPath + "/" + args[0]

	if err := getCommand(&res, conn); err != nil {
		return err
	}

	displayRule(res)

	return nil
}

// ######################## ADD ##########################

type ruleAddCommand struct {
	Name      string `required:"true" short:"n" long:"name" description:"The rule's name"`
	Comment   string `short:"c" long:"comment" description:"A short comment describing the rule"`
	Direction string `required:"true" short:"d" long:"direction" description:"The direction of the file transfer" choice:"SEND" choice:"RECEIVE"`
	Path      string `required:"true" short:"p" long:"path" description:"The path to the destination of the file"`
}

func (r *ruleAddCommand) Execute([]string) error {
	rule := rest.InRule{
		Name:    r.Name,
		Comment: r.Comment,
		IsSend:  r.Direction == "SEND",
		Path:    r.Path,
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RulesPath

	loc, err := addCommand(rule, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The rule \033[33m'%s'\033[0m was successfully added. "+
		"It can be consulted at the address: \033[37m%s\033[0m\n", rule.Name, loc)

	return nil
}

// ######################## DELETE ##########################

type ruleDeleteCommand struct{}

func (r *ruleDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing rule ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RulesPath + "/" + args[0]

	if err := deleteCommand(conn); err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "The rule n°\033[33m%s\033[0m was successfully deleted from "+
		"the database\n", args[0])

	return nil
}

// ######################## LIST ##########################

type ruleListCommand struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name" default:"name"`
}

func (r *ruleListCommand) Execute([]string) error {
	conn, err := listURL(rest.RulesPath, &r.listOptions, r.SortBy)
	if err != nil {
		return err
	}

	res := map[string][]rest.OutRule{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	rules := res["rules"]
	if len(rules) > 0 {
		fmt.Fprintf(w, "\033[33mRules:\033[0m\n")
		for _, rule := range rules {
			displayRule(rule)
		}
	} else {
		fmt.Fprintln(w, "\033[31mNo rules found\033[0m")
	}

	return nil
}
