package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type ruleAccessCommand struct {
	Grant  ruleAccessGrantCommand  `command:"grant" description:"Grant access to a rule"`
	List   ruleAccessListCommand   `command:"list" description:"List all the rule's accesses"`
	Revoke ruleAccessRevokeCommand `command:"revoke" description:"Revoke access to a rule"`
}

func displayRuleAccess(acc model.RuleAccess) {
	w := getColorable()

	fmt.Fprintf(w, "Access to \033[37;1m%s\033[0m n°\033[37;1m%v\033[0m\n",
		fromTableName(acc.ObjectType), acc.ObjectID)
}

// ######################## GRANT ##########################

type ruleAccessGrantCommand struct {
	ID   uint64 `required:"true" short:"i" long:"id" description:"The ID of the access' target"`
	Type string `required:"true" short:"t" long:"type" description:"The type of the access' target" choice:"local agent" choice:"remote agent" choice:"local account" choice:"remote account"`
}

func (r *ruleAccessGrantCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing rule ID")
	}

	acc := model.RuleAccess{
		ObjectID:   r.ID,
		ObjectType: toTableName(r.Type),
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.RulesPath + "/" + args[0] + admin.RulePermissionPath

	loc, err := addCommand(acc, conn)
	if err != nil {
		return err
	}

	w := getColorable()
	fmt.Fprintf(w, "Access to rule n°\033[33m%s\033[0m was successfully granted "+
		"to \033[33m%s\033[0m n°\033[33m%v\033[0m. Granted accesses can be "+
		"consulted at the address: %s", args[0], fromTableName(acc.ObjectType),
		acc.ObjectID, loc)

	return nil
}

// ######################## REVOKE ##########################

type ruleAccessRevokeCommand struct {
	ID   uint64 `required:"true" short:"i" long:"id" description:"The ID of the access' target"`
	Type string `required:"true" short:"t" long:"type" description:"The type of the access' target" choice:"local agent" choice:"remote agent" choice:"local accounts" choice:"remote account"`
}

func (r *ruleAccessRevokeCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing rule ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.RulesPath + "/" + args[0] + admin.RulePermissionPath

	acc := &model.RuleAccess{
		ObjectID:   r.ID,
		ObjectType: toTableName(r.Type),
	}

	content, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	body := bytes.NewReader(content)

	req, err := http.NewRequest(http.MethodDelete, conn.String(), body)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, conn)
	if err != nil {
		return err
	}

	if res.ContentLength > 0 {
		msg, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		fmt.Fprint(out, string(msg))
	}

	if res.StatusCode != http.StatusOK {
		return handleErrors(res, conn)
	}

	w := getColorable()
	fmt.Fprintf(w, "Access to rule n°\033[33m%s\033[0m was successfully revoked "+
		"from \033[33m%s\033[0m n°\033[33m%v\033[0m.", args[0],
		fromTableName(acc.ObjectType), acc.ObjectID)

	return nil
}

// ######################## LIST ##########################

type ruleAccessListCommand struct{}

func (r *ruleAccessListCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing rule ID")
	}

	conn, err := url.Parse(auth.DSN)
	if err != nil {
		return err
	}
	conn.Path = admin.APIPath + admin.RulesPath + "/" + args[0] + admin.RulePermissionPath

	res := map[string][]model.RuleAccess{}
	if err := getCommand(&res, conn); err != nil {
		return err
	}

	w := getColorable()
	perms := res["permissions"]
	if len(perms) > 0 {
		fmt.Fprintf(w, "\033[33mPermissions:\033[0m\n")
		for _, acc := range perms {
			displayRuleAccess(acc)
		}
	} else {
		fmt.Fprintf(w, "\033[31mAccess to rule %s is unrestricted\033[0m\n", args[0])
	}

	return nil
}
