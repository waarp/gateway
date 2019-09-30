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

type partnerCommand struct{}

func displayPartner(out *os.File, partner model.Partner) {
	w := getColorable(out)

	fmt.Fprintf(w, "\033[97;1mPartner n°%v:\033[0m\n", partner.ID)
	fmt.Fprintf(w, "├─\033[97mName:\033[0m \033[34;4m%s\033[0m\n", partner.Name)
	fmt.Fprintf(w, "├─\033[97mInterfaceID:\033[0m \033[34;4m%v\033[0m\n", partner.InterfaceID)
	fmt.Fprintf(w, "├─\033[97mAddress:\033[0m \033[34;4m%s\033[0m\n", partner.Address)
	fmt.Fprintf(w, "└─\033[97mPort:\033[0m \033[33m%v\033[0m\n", partner.Port)
}

// ############################## GET #####################################

type partnerGetCommand struct{}

// Execute executes the 'partner' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner name")
	}

	subpath := admin.PartnersURI + "/" + args[0]
	partner := model.Partner{}
	err := getCommand(os.Stdin, os.Stdout, subpath, &partner)
	if err != nil {
		return err
	}

	fmt.Println()
	displayPartner(os.Stdout, partner)

	return nil
}

// ############################### LIST #######################################

type partnerListCommand struct {
	Limit     int      `short:"l" long:"limit" description:"The max number of entries which can be returned" default:"20"`
	Offset    int      `short:"o" long:"offset" description:"The offset from which the first entry is taken" default:"0"`
	Sort      string   `short:"s" long:"sort" description:"The parameter used to sort the returned entries" choice:"name" choice:"address" default:"name"`
	Reverse   bool     `short:"d" long:"descending" description:"If present, the order of the sorting will be reversed"`
	Addresses []string `short:"a" long:"address" description:"Filter the partners based on their host addresses"`
	Interface uint64   `short:"i" long:"interface" description:"Filter the partners based on the interface they are attached to"`
}

func (p *partnerListCommand) listPartners(in *os.File, out *os.File) ([]byte, error) {
	addr := auth.Address + admin.RestURI + admin.PartnersURI

	addr += "?limit=" + strconv.Itoa(p.Limit)
	addr += "&offset=" + strconv.Itoa(p.Offset)
	addr += "&sortby=" + p.Sort
	if p.Reverse {
		addr += orderDesc
	}

	for _, host := range p.Addresses {
		addr += "&address=" + host
	}

	if p.Interface != 0 {
		addr += "&interface=" + strconv.FormatUint(p.Interface, 10)
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

	partners := map[string][]model.Partner{}
	if err := json.Unmarshal(content, &partners); err != nil {
		return err
	}

	fmt.Println()
	for _, partner := range partners["partners"] {
		displayPartner(os.Stdout, partner)
	}

	return nil
}

// ############################### CREATE #######################################

type partnerCreateCommand struct {
	Name        string `required:"true" short:"n" long:"name" description:"The partner's name'"`
	Host        string `required:"true" short:"a" long:"address" description:"The address of the partner's host'"`
	Port        uint16 `required:"true" short:"p" long:"port" description:"The TCP port used by the partner"`
	InterfaceID uint64 `required:"true" short:"i" long:"interface_id" description:"The interface to which the partner will be attached"`
}

// Execute executes the 'create' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerCreateCommand) Execute(_ []string) error {
	addr := auth.Address + admin.RestURI + admin.PartnersURI

	partner := &model.Partner{
		Name:        p.Name,
		Address:     p.Host,
		Port:        p.Port,
		InterfaceID: p.InterfaceID,
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
	Name        string `short:"n" long:"name" description:"The partner's name'"`
	Host        string `short:"a" long:"address" description:"The address of the partner's host'"`
	Port        uint16 `short:"p" long:"port" description:"The TCP port used by the partner"`
	InterfaceID uint64 `short:"i" long:"interface_id" description:"The interface to which the partner will be attached"`
}

// Execute executes the 'update' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (p *partnerUpdateCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing partner name")
	}

	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + args[0]

	partner := &struct {
		Name, Address string
		Port          uint16
		InterfaceID   uint64
	}{
		Name:        p.Name,
		Address:     p.Host,
		Port:        p.Port,
		InterfaceID: p.InterfaceID,
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
