package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

type accountCommand struct{}

func displayAccount(out *os.File, account *model.Account) error {
	w := getColorable(out)

	fmt.Fprintf(w, "\033[97;1mAccount n°%v:\033[0m\n", account.ID)
	fmt.Fprintf(w, "├─\033[97mUsername:\033[0m \033[37m%s\033[0m\n", account.Username)
	fmt.Fprintf(w, "└─\033[97mPartnerID:\033[0m \033[37m%v\033[0m\n", account.PartnerID)
	return nil
}

// ############################## GET #####################################

type accountGetCommand struct{}

// Execute executes the 'account' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (a *accountGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing account name")
	}

	addr := auth.Address + admin.RestURI + admin.AccountsURI + "/" + args[0]

	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, auth.Username, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	account := &model.Account{}
	if err := json.Unmarshal(body, account); err != nil {
		return err
	}

	fmt.Println()
	if err := displayAccount(os.Stdout, account); err != nil {
		return err
	}

	return nil
}

// ############################### LIST #######################################

type accountListCommand struct {
	Limit   int    `short:"l" long:"limit" description:"The max number of entries which can be returned" default:"20"`
	Offset  int    `short:"o" long:"offset" description:"The offset from which the first entry is taken" default:"0"`
	Sort    string `short:"s" long:"sort" description:"The parameter used to sort the returned entries" choice:"username" default:"username"`
	Reverse bool   `short:"d" long:"descending" description:"If present, the order of the sorting will be reversed"`
}

func (a *accountListCommand) listAccounts(in *os.File, out *os.File) ([]byte, error) {
	addr := auth.Address + admin.RestURI + admin.AccountsURI

	addr += "?limit=" + strconv.Itoa(a.Limit)
	addr += "&offset=" + strconv.Itoa(a.Offset)
	addr += "&sortby=" + a.Sort
	if a.Reverse {
		addr += orderDesc
	}

	req, err := http.NewRequest(http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}

	res, err := executeRequest(req, auth.Username, in, out)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Execute executes the 'account' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (a *accountListCommand) Execute(_ []string) error {
	content, err := a.listAccounts(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	accounts := map[string][]*model.Account{}
	if err := json.Unmarshal(content, &accounts); err != nil {
		return err
	}

	fmt.Println()
	for _, account := range accounts["accounts"] {
		if err := displayAccount(os.Stdout, account); err != nil {
			return err
		}
	}

	return nil
}

// ############################### CREATE #######################################

type accountCreateCommand struct {
	Username  string `required:"true" short:"n" long:"name" description:"The account's username'"`
	Password  string `required:"true" short:"p" long:"password" description:"The account's password"`
	PartnerID uint64 `required:"true" short:"i" long:"partner_id" description:"The partner to which the account will be attached"`
}

// Execute executes the 'create' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (a *accountCreateCommand) Execute(_ []string) error {
	addr := auth.Address + admin.RestURI + admin.AccountsURI

	account := &model.Account{
		Username:  a.Username,
		Password:  []byte(a.Password),
		PartnerID: a.PartnerID,
	}

	path, err := sendBean(account, os.Stdin, os.Stdout, addr, http.MethodPost)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mAccount successfully created at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

//################################### UPDATE ######################################

type accountUpdateCommand struct {
	Username  string `short:"n" long:"name" description:"The account's username'"`
	Password  string `short:"p" long:"password" description:"The account's password"`
	PartnerID uint64 `short:"i" long:"partner_id" description:"The partner to which the account will be attached"`
}

// Execute executes the 'update' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (a *accountUpdateCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing account name")
	}

	addr := auth.Address + admin.RestURI + admin.AccountsURI + "/" + args[0]

	account := &model.Account{
		Username:  a.Username,
		Password:  []byte(a.Password),
		PartnerID: a.PartnerID,
	}

	path, err := sendBean(account, os.Stdin, os.Stdout, addr, http.MethodPatch)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mAccount successfully updated at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

// ############################## DELETE #####################################

type accountDeleteCommand struct{}

// Execute executes the 'account' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (a *accountDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing account name")
	}

	addr := auth.Address + admin.RestURI + admin.AccountsURI + "/" + args[0]

	req, err := http.NewRequest(http.MethodDelete, addr, nil)
	if err != nil {
		return err
	}

	res, err := executeRequest(req, auth.Username, os.Stdin, os.Stdout)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mAccount\033[0m \033[33;1m'%s'\033[0m"+
		" \033[97msuccessfully deleted\033[0m\n", args[0])
	return nil
}
