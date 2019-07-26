package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
)

// init adds the 'partner' command to the program arguments parser
func init() {
	var partner partnerCommand
	p, err := parser.AddCommand("partner", "Manage waarp-gateway partners",
		"The command to manage the waarp-gateway partners.",
		&partner)
	if err != nil {
		panic(err.Error())
	}

	var get partnerGetCommand
	_, err = p.AddCommand("get", "Get partner",
		"Retrieve a single partner entry.",
		&get)
	if err != nil {
		panic(err.Error())
	}

	var list partnerListCommand
	_, err = p.AddCommand("list", "List partners",
		"List the partner entries.",
		&list)
	if err != nil {
		panic(err.Error())
	}

	var create partnerCreateCommand
	_, err = p.AddCommand("create", "Create partner",
		"Create a partner and add it to the database.",
		&create)
	if err != nil {
		panic(err.Error())
	}

	var update partnerUpdateCommand
	_, err = p.AddCommand("update", "Update partner",
		"Updates a partner entry in the database.",
		&update)
	if err != nil {
		panic(err.Error())
	}

	var del partnerDeleteCommand
	_, err = p.AddCommand("delete", "Delete partner",
		"Removes a partner entry from the database.",
		&del)
	if err != nil {
		panic(err.Error())
	}
}

type partnerCommand struct{}

func displayPartner(out *os.File, partner *model.Partner) error {
	w := getColorable(out)

	fmt.Fprintf(w, "\033[97;1mPartner '%s':\033[0m\n", partner.Name)
	fmt.Fprintf(w, "├─\033[97mAddress:\033[0m \033[34;4m%s\033[0m\n", partner.Address)
	fmt.Fprintf(w, "├─\033[97mPort:\033[0m \033[33m%v\033[0m\n", partner.Port)
	fmt.Fprintf(w, "└─\033[97mType:\033[0m \033[37m%s\033[0m\n", partner.Type)

	return nil
}

func sendBean(bean interface{}, in, out *os.File, addr, method string) (string, error) {

	content, err := json.Marshal(bean)
	if err != nil {
		return "", err
	}
	body := bytes.NewReader(content)

	req, err := http.NewRequest(method, addr, body)
	if err != nil {
		return "", err
	}

	res, err := executeRequest(req, auth.Username, in, out)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	return res.Header.Get("Location"), nil
}

// ############################## GET #####################################

type partnerGetCommand struct{}

func (p *partnerGetCommand) getPartner(in *os.File, out *os.File, name string) (*model.Partner, error) {
	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + name

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
	partner := &model.Partner{}
	if err := json.Unmarshal(body, partner); err != nil {
		return nil, err
	}

	return partner, nil
}

// Execute executes the 'partner' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner name")
	}

	partner, err := p.getPartner(os.Stdin, os.Stdout, args[0])
	if err != nil {
		return err
	}

	fmt.Println()
	if err := displayPartner(os.Stdout, partner); err != nil {
		return err
	}

	return nil
}

// ############################### LIST #######################################

type partnerListCommand struct {
	Limit     int      `short:"l" long:"limit" description:"The max number of entries which can be returned" default:"20"`
	Offset    int      `short:"o" long:"offset" description:"The offset from which the first entry is taken" default:"0"`
	Sort      string   `short:"s" long:"sort" description:"The parameter used to sort the returned entries" choice:"name" choice:"address" choice:"type" default:"name"`
	Reverse   bool     `short:"d" long:"descending" description:"If present, the order of the sorting will be reversed"`
	Types     []string `short:"t" long:"type" description:"Filter the partners based on their types"`
	Addresses []string `short:"a" long:"address" description:"Filter the partners based on their host addresses"`
}

func (p *partnerListCommand) listPartners(in *os.File, out *os.File) ([]byte, error) {
	addr := auth.Address + admin.RestURI + admin.PartnersURI

	addr += "?limit=" + strconv.Itoa(p.Limit)
	addr += "&offset=" + strconv.Itoa(p.Offset)
	addr += "&sortby=" + p.Sort
	if p.Reverse {
		addr += orderDesc
	}
	for _, typ := range p.Types {
		addr += "&type=" + typ
	}
	for _, host := range p.Addresses {
		addr += "&address=" + host
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

// Execute executes the 'partner' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerListCommand) Execute(_ []string) error {
	content, err := p.listPartners(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	partners := map[string][]*model.Partner{}
	if err := json.Unmarshal(content, &partners); err != nil {
		return err
	}

	fmt.Println()
	for _, partner := range partners["partners"] {
		if err := displayPartner(os.Stdout, partner); err != nil {
			return err
		}
	}

	return nil
}

// ############################### CREATE #######################################

type partnerCreateCommand struct {
	Name string `required:"true" short:"n" long:"name" description:"The partner's name'"`
	Host string `required:"true" short:"a" long:"address" description:"The address of the partner's host'"`
	Port uint16 `required:"true" short:"p" long:"port" description:"The TCP port used by the partner"`
	Type string `required:"true" short:"t" long:"type" description:"The type of the partner" choice:"sftp"`
}

// Execute executes the 'create' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerCreateCommand) Execute(_ []string) error {
	addr := auth.Address + admin.RestURI + admin.PartnersURI

	partner := &model.Partner{
		Name:    p.Name,
		Address: p.Host,
		Port:    p.Port,
		Type:    p.Type,
	}

	path, err := sendBean(partner, os.Stdin, os.Stdout, addr, http.MethodPost)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mPartner successfully created at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

//################################### UPDATE ######################################

type partnerUpdateCommand struct {
	Name string `short:"n" long:"name" description:"The partner's name'"`
	Host string `short:"a" long:"address" description:"The address of the partner's host'"`
	Port uint16 `short:"p" long:"port" description:"The TCP port used by the partner"`
	Type string `short:"t" long:"type" description:"The type of the partner" choice:"sftp"`
}

// Execute executes the 'update' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerUpdateCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner name")
	}

	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + args[0]

	partner := &model.Partner{
		Name:    p.Name,
		Address: p.Host,
		Port:    p.Port,
		Type:    p.Type,
	}

	path, err := sendBean(partner, os.Stdin, os.Stdout, addr, http.MethodPatch)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mPartner successfully updated at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

// ############################## DELETE #####################################

type partnerDeleteCommand struct{}

// Execute executes the 'partner' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner name")
	}

	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + args[0]

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
	fmt.Fprintf(w, "\033[97mPartner\033[0m \033[33;1m'%s'\033[0m \033[97msuccessfully deleted\033[0m\n",
		args[0])
	return nil
}
