package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
)

type ruleCommand struct {
	Get    ruleGet      `command:"get" description:"Retrieve a rule's information"`
	Add    ruleAdd      `command:"add" description:"Add a new rule"`
	Delete ruleDelete   `command:"delete" description:"Delete a rule"`
	List   ruleList     `command:"list" description:"List the known rules"`
	Update ruleUpdate   `command:"update" description:"Update an existing rule"`
	Allow  ruleAllowAll `command:"allow" description:"Remove all usage restriction on a rule"`
}

func displayTasks(w io.Writer, rule *rest.OutRule) {
	fmt.Fprintln(w, orange("    Pre tasks:"))
	for i, t := range rule.PreTasks {
		prefix := "    ├─Command"
		if i == len(rule.PreTasks)-1 {
			prefix = "    └─Command"
		}
		fmt.Fprintln(w, bold(prefix), t.Type, bold("with args:"), string(t.Args))
	}
	fmt.Fprintln(w, orange("    Post tasks:"))
	for i, t := range rule.PreTasks {
		prefix := "    ├─Command"
		if i == len(rule.PreTasks)-1 {
			prefix = "    └─Command"
		}
		fmt.Fprintln(w, bold(prefix), t.Type, bold("with args:"), string(t.Args))
	}
	fmt.Fprintln(w, orange("    Error tasks:"))
	for i, t := range rule.PreTasks {
		prefix := "    ├─Command"
		if i == len(rule.PreTasks)-1 {
			prefix = "    └─Command"
		}
		fmt.Fprintln(w, bold(prefix), t.Type, bold("with args:"), string(t.Args))
	}
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

	fmt.Fprintln(w, orange(bold("● Rule", rule.Name, "("+way+")")))
	fmt.Fprintln(w, orange("    Comment:"), rule.Comment)
	fmt.Fprintln(w, orange("    Path:   "), rule.Path)
	fmt.Fprintln(w, orange("    InPath: "), rule.InPath)
	fmt.Fprintln(w, orange("    OutPath:"), rule.OutPath)
	displayTasks(w, rule)
	fmt.Fprintln(w, orange("    Authorized agents:"))
	fmt.Fprintln(w, bold("    ├─Servers:         "), servers)
	fmt.Fprintln(w, bold("    ├─Partners:        "), partners)
	fmt.Fprintln(w, bold("    ├─Server accounts: "), locAcc)
	fmt.Fprintln(w, bold("    └─Partner accounts:"), remAcc)
}

func parseTasks(rule *rest.UptRule, pre, post, errs []string) error {
	preDecoder := json.NewDecoder(strings.NewReader("[" + strings.Join(pre, ",") + "]"))
	preDecoder.DisallowUnknownFields()
	if err := preDecoder.Decode(&rule.PreTasks); err != nil && err != io.EOF {
		return fmt.Errorf("invalid pre task: %s", err)
	}
	postDecoder := json.NewDecoder(strings.NewReader("[" + strings.Join(post, ",") + "]"))
	postDecoder.DisallowUnknownFields()
	if err := postDecoder.Decode(&rule.PostTasks); err != nil && err != io.EOF {
		return fmt.Errorf("invalid post task: %s", err)
	}
	errDecoder := json.NewDecoder(strings.NewReader("[" + strings.Join(errs, ",") + "]"))
	errDecoder.DisallowUnknownFields()
	if err := errDecoder.Decode(&rule.ErrorTasks); err != nil && err != io.EOF {
		return fmt.Errorf("invalid error task: %s", err)
	}
	return nil
}

// ######################## GET ##########################

type ruleGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (r *ruleGet) Execute([]string) error {
	path := admin.APIPath + rest.RulesPath + "/" + r.Args.Name

	rule := &rest.OutRule{}
	if err := get(path, rule); err != nil {
		return err
	}
	displayRule(getColorable(), rule)
	return nil
}

// ######################## ADD ##########################

type ruleAdd struct {
	Name       string   `required:"true" short:"n" long:"name" description:"The rule's name"`
	Comment    string   `short:"c" long:"comment" description:"A short comment describing the rule"`
	Direction  string   `required:"true" short:"d" long:"direction" description:"The direction of the file transfer" choice:"SEND" choice:"RECEIVE"`
	Path       string   `required:"true" short:"p" long:"path" description:"The path used to identify the rule"`
	InPath     string   `short:"i" long:"in_path" description:"The path to the destination of the file"`
	OutPath    string   `short:"o" long:"out_path" description:"The path to the source of the file"`
	PreTasks   []string `short:"r" long:"pre" description:"A pre-transfer task in JSON format, can be repeated"`
	PostTasks  []string `short:"s" long:"post" description:"A post-transfer task in JSON format, can be repeated"`
	ErrorTasks []string `short:"e" long:"err" description:"A transfer error task in JSON format, can be repeated"`
}

func (r *ruleAdd) Execute([]string) error {
	rule := &rest.InRule{
		UptRule: &rest.UptRule{
			Name:    r.Name,
			Comment: r.Comment,
			Path:    r.Path,
		},
		IsSend: r.Direction == "SEND",
	}
	if err := parseTasks(rule.UptRule, r.PreTasks, r.PostTasks, r.ErrorTasks); err != nil {
		return err
	}
	path := admin.APIPath + rest.RulesPath

	if err := add(path, rule); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The rule", bold(rule.Name), "was successfully added.")
	return nil
}

// ######################## DELETE ##########################

type ruleDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (r *ruleDelete) Execute([]string) error {
	path := admin.APIPath + rest.RulesPath + "/" + r.Args.Name

	if err := remove(path); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The rule", bold(r.Args.Name), "was successfully deleted.")
	return nil
}

// ######################## LIST ##########################

type ruleList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (r *ruleList) Execute([]string) error {
	addr, err := listURL(rest.APIPath+rest.RulesPath, &r.listOptions, r.SortBy)
	if err != nil {
		return err
	}

	body := map[string][]rest.OutRule{}
	if err := list(addr, &body); err != nil {
		return err
	}

	rules := body["rules"]
	w := getColorable()
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
}

// ######################## UPDATE ##########################

type ruleUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The server's name"`
	} `positional-args:"yes"`
	Name       string   `short:"n" long:"name" description:"The rule's name"`
	Comment    string   `short:"c" long:"comment" description:"A short comment describing the rule"`
	Path       string   `short:"p" long:"path" description:"The path used to identify the rule"`
	InPath     string   `short:"i" long:"in_path" description:"The path to the destination of the file"`
	OutPath    string   `short:"o" long:"out_path" description:"The path to the source of the file"`
	PreTasks   []string `short:"r" long:"pre" description:"A pre-transfer task in JSON format, can be repeated"`
	PostTasks  []string `short:"s" long:"post" description:"A post-transfer task in JSON format, can be repeated"`
	ErrorTasks []string `short:"e" long:"err" description:"A transfer error task in JSON format, can be repeated"`
}

func (r *ruleUpdate) Execute([]string) error {
	rule := &rest.UptRule{
		Name:    r.Name,
		Comment: r.Comment,
		Path:    r.Path,
		InPath:  r.InPath,
		OutPath: r.OutPath,
	}
	if err := parseTasks(rule, r.PreTasks, r.PostTasks, r.ErrorTasks); err != nil {
		return err
	}
	path := admin.APIPath + rest.RulesPath + "/" + r.Args.Name

	if err := update(path, rule); err != nil {
		return err
	}
	name := r.Args.Name
	if rule.Name != "" {
		name = rule.Name
	}
	fmt.Fprintln(getColorable(), "The rule", bold(name), "was successfully updated.")
	return nil
}

// ######################## RESTRICT ##########################

type ruleAllowAll struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The rule's name"`
	} `positional-args:"yes"`
}

func (r *ruleAllowAll) Execute([]string) error {
	addr, err := url.Parse(commandLine.Args.Address)
	if err != nil {
		return err
	}
	addr.Path = admin.APIPath + rest.RulesPath + "/" + r.Args.Name + "/allow_all"

	resp, err := sendRequest(addr, nil, http.MethodPut)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	w := getColorable()
	switch resp.StatusCode {
	case http.StatusOK:
		fmt.Fprintln(w, "The use of rule", bold(r.Args.Name), "is now unrestricted.")
		return nil
	case http.StatusNotFound:
		return getResponseMessage(resp)
	default:
		return fmt.Errorf("unexpected error: %s", getResponseMessage(resp))
	}
}
