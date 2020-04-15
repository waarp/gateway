package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type ruleCommand struct {
	Get    ruleGet    `command:"get" description:"Retrieve a rule's information"`
	Add    ruleAdd    `command:"add" description:"Add a new rule"`
	Delete ruleDelete `command:"delete" description:"Delete a rule"`
	List   ruleList   `command:"list" description:"List the known rules"`
	Update ruleUpdate `command:"update" description:"Update an existing rule"`
	//Access ruleAccess `command:"access" description:"Manage the permissions for a rule"`
	Tasks ruleTasksCommand `command:"tasks" description:"Manage the rule's task chain"`
}

func displayRule(w io.Writer, rule *rest.OutRule) {
	way := "RECEIVE"
	if rule.IsSend {
		way = "SEND"
	}

	servers := strings.Join(rule.Authorized.LocalServers, ", ")
	partners := strings.Join(rule.Authorized.RemotePartners, ", ")
	la := []string{}
	for server, accounts := range rule.Authorized.LocalAccounts {
		for _, account := range accounts {
			la = append(la, fmt.Sprint(server, ".", account))
		}
	}
	ra := []string{}
	for partner, accounts := range rule.Authorized.RemoteAccounts {
		for _, account := range accounts {
			ra = append(ra, fmt.Sprint(partner, ".", account))
		}
	}
	locAcc := strings.Join(la, ", ")
	remAcc := strings.Join(ra, ", ")

	fmt.Fprintln(w, bold("● Rule", rule.Name, "("+way+")"))
	fmt.Fprintln(w, orange("   Comment:"), rule.Comment)
	fmt.Fprintln(w, orange("      Path:"), rule.Path)
	fmt.Fprintln(w, orange("    InPath:"), rule.InPath)
	fmt.Fprintln(w, orange("   OutPath:"), rule.OutPath)
	fmt.Fprintln(w, orange("   Authorized agents"))
	fmt.Fprintln(w, orange("   ├─         Servers:"), servers)
	fmt.Fprintln(w, orange("   ├─        Partners:"), partners)
	fmt.Fprintln(w, orange("   ├─ Server accounts:"), locAcc)
	fmt.Fprintln(w, orange("   └─Partner accounts:"), remAcc)
}

// ######################## GET ##########################

type ruleGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (r *ruleGet) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RulesPath + "/" + r.Args.Name

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		rule := &rest.OutRule{}
		if err := unmarshalBody(resp.Body, rule); err != nil {
			return err
		}
		displayRule(w, rule)
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## ADD ##########################

type ruleAdd struct {
	Name      string `required:"true" short:"n" long:"name" description:"The rule's name"`
	Comment   string `short:"c" long:"comment" description:"A short comment describing the rule"`
	Direction string `required:"true" short:"d" long:"direction" description:"The direction of the file transfer" choice:"SEND" choice:"RECEIVE"`
	Path      string `required:"true" short:"p" long:"path" description:"The path used to identify the rule"`
	InPath    string `short:"i" long:"in_path" description:"The path to the source of the file"`
	OutPath   string `short:"o" long:"out_path" description:"The path to the destination of the file"`
}

func (r *ruleAdd) Execute([]string) error {
	rule := rest.InRule{
		Name:    r.Name,
		Comment: r.Comment,
		IsSend:  r.Direction == "SEND",
		Path:    r.Path,
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RulesPath

	resp, err := sendRequest(conn, rule, http.MethodPost)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The rule", bold(rule.Name), "was successfully added.")
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## DELETE ##########################

type ruleDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (r *ruleDelete) Execute([]string) error {
	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RulesPath + "/" + r.Args.Name

	resp, err := sendRequest(conn, nil, http.MethodDelete)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusNoContent:
		fmt.Fprintln(w, "The rule", bold(r.Args.Name), "was successfully deleted.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}

// ######################## LIST ##########################

type ruleList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (r *ruleList) Execute([]string) error {
	conn, err := listURL(rest.RulesPath, &r.listOptions, r.SortBy)
	if err != nil {
		return err
	}

	resp, err := sendRequest(conn, nil, http.MethodGet)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		body := map[string][]rest.OutRule{}
		if err := unmarshalBody(resp.Body, &body); err != nil {
			return err
		}
		rules := body["rules"]
		if len(rules) > 0 {
			fmt.Fprintln(w, bold("Rules:"))
			for _, r := range rules {
				rule := r
				displayRule(w, &rule)
			}
		} else {
			fmt.Fprintln(w, "No rules found.")
		}
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error (%s): %s", resp.Status, getResponseMessage(resp).Error())
	}
}

// ######################## UPDATE ##########################

type ruleUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
	Name    string `short:"n" long:"name" description:"The rule's name"`
	Comment string `short:"c" long:"comment" description:"A short comment describing the rule"`
	Path    string `short:"p" long:"path" description:"The path used to identify the rule"`
	InPath  string `short:"i" long:"in_path" description:"The path to the source of the file"`
	OutPath string `short:"o" long:"out_path" description:"The path to the destination of the file"`
}

func (r *ruleUpdate) Execute([]string) error {
	update := rest.UptRule{
		Name:    r.Name,
		Comment: r.Comment,
		Path:    r.Path,
		InPath:  r.InPath,
		OutPath: r.OutPath,
	}

	conn, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + rest.RulesPath + "/" + r.Args.Name

	resp, err := sendRequest(conn, update, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusCreated:
		fmt.Fprintln(w, "The rule", bold(update.Name), "was successfully updated.")
		return nil
	case http.StatusBadRequest:
		return getResponseMessage(resp)
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %v - %s", resp.StatusCode,
			getResponseMessage(resp).Error())
	}
}
