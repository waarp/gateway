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

type interfaceCommand struct{}

func displayInterface(out *os.File, inter model.Interface) {
	w := getColorable(out)

	fmt.Fprintf(w, "\033[97;1mInterface n°%v:\033[0m\n", inter.ID)
	fmt.Fprintf(w, "├─\033[97mName:\033[0m \033[34;4m%s\033[0m\n", inter.Name)
	fmt.Fprintf(w, "├─\033[97mType:\033[0m \033[34;4m%v\033[0m\n", inter.Type)
	fmt.Fprintf(w, "└─\033[97mPort:\033[0m \033[33m%v\033[0m\n", inter.Port)
}

// ############################## GET #####################################

type interfaceGetCommand struct{}

// Execute executes the 'interface' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *interfaceGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing interface ID")
	}

	subpath := admin.InterfacesURI + "/" + args[0]
	inter := model.Interface{}
	err := getCommand(os.Stdin, os.Stdout, subpath, &inter)
	if err != nil {
		return err
	}

	fmt.Println()
	displayInterface(os.Stdout, inter)

	return nil
}

// ############################### LIST #######################################

type interfaceListCommand struct {
	Limit   int      `short:"l" long:"limit" description:"The max number of entries which can be returned" default:"20"`
	Offset  int      `short:"o" long:"offset" description:"The offset from which the first entry is taken" default:"0"`
	Sort    string   `short:"s" long:"sort" description:"The parameter used to sort the returned entries" choice:"name" choice:"type" choice:"port" default:"name"`
	Reverse bool     `short:"d" long:"descending" description:"If present, the order of the sorting will be reversed"`
	Types   []string `short:"t" long:"type" description:"Filter the interfaces based on the protocol they use" choice:"sftp" choice:"http" choice:"r66"`
	Ports   []uint16 `short:"p" long:"port" description:"Filter the interfaces based on the port they use"`
}

func (p *interfaceListCommand) listInterfaces(in *os.File, out *os.File) ([]byte, error) {
	addr := auth.Address + admin.RestURI + admin.InterfacesURI

	addr += "?limit=" + strconv.Itoa(p.Limit)
	addr += "&offset=" + strconv.Itoa(p.Offset)
	addr += "&sortby=" + p.Sort
	if p.Reverse {
		addr += orderDesc
	}

	for _, typ := range p.Types {
		addr += "&type=" + typ
	}

	for _, port := range p.Ports {
		addr += "&port=" + strconv.FormatUint(uint64(port), 10)
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

// Execute executes the 'interface' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *interfaceListCommand) Execute(_ []string) error {
	content, err := p.listInterfaces(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	interfaces := map[string][]model.Interface{}
	if err := json.Unmarshal(content, &interfaces); err != nil {
		return err
	}

	fmt.Println()
	for _, inter := range interfaces["interfaces"] {
		displayInterface(os.Stdout, inter)
	}

	return nil
}

// ############################### CREATE #######################################

type interfaceCreateCommand struct {
	Name string `required:"true" short:"n" long:"name" description:"The interface's name"`
	Port uint16 `required:"true" short:"p" long:"port" description:"The TCP port used by the interface"`
	Type string `required:"true" short:"t" long:"type" description:"The protocol the interface will use" choice:"sftp" choice:"http" choice:"r66"`
}

// Execute executes the 'create' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *interfaceCreateCommand) Execute(_ []string) error {
	addr := auth.Address + admin.RestURI + admin.InterfacesURI

	inter := &model.Interface{
		Name: p.Name,
		Type: p.Type,
		Port: p.Port,
	}

	path, err := sendBean(inter, os.Stdin, os.Stdout, addr, http.MethodPost)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mInterface successfully created at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

//################################### UPDATE ######################################

type interfaceUpdateCommand struct {
	Name string `short:"n" long:"name" description:"The interface's name"`
	Port uint16 `short:"p" long:"port" description:"The TCP port used by the interface"`
	Type string `short:"t" long:"type" description:"The protocol the interface will use" choice:"sftp" choice:"http" choice:"r66"`
}

// Execute executes the 'update' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *interfaceUpdateCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing interface ID")
	}

	addr := auth.Address + admin.RestURI + admin.InterfacesURI + "/" + args[0]

	inter := &struct {
		Name, Type string
		Port       uint16
	}{
		Name: p.Name,
		Type: p.Type,
		Port: p.Port,
	}

	path, err := sendBean(inter, os.Stdin, os.Stdout, addr, http.MethodPatch)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mInterface successfully updated at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

// ############################## DELETE #####################################

type interfaceDeleteCommand struct{}

// Execute executes the 'interface' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *interfaceDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing interface ID")
	}

	addr := auth.Address + admin.RestURI + admin.InterfacesURI + "/" + args[0]

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
	fmt.Fprintf(w, "\033[97mInterface\033[0m \033[33;1m'%s'\033[0m \033[97msuccessfully deleted\033[0m\n",
		args[0])
	return nil
}
