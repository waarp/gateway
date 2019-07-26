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

// init adds the 'certificate' command to the program arguments parser
func init() {
	var certificate certificateCommand
	p, err := parser.AddCommand("certificate", "Manage waarp-gateway certificates",
		"The command to manage the waarp-gateway certificates.",
		&certificate)
	if err != nil {
		panic(err.Error())
	}

	updateCert := certificateUpdateCommand{certificateCommand: &certificate}
	_, err = p.AddCommand("update", "Update certificate",
		"Updates a certificate entry in the database.",
		&updateCert)
	if err != nil {
		panic(err.Error())
	}

	deleteCert := certificateDeleteCommand{certificateCommand: &certificate}
	_, err = p.AddCommand("delete", "Delete certificate",
		"Removes a certificate entry from the database.",
		&deleteCert)
	if err != nil {
		panic(err.Error())
	}

	getCert := certificateGetCommand{certificateCommand: &certificate}
	_, err = p.AddCommand("get", "Get certificate",
		"Retrieve a certificate entry from the database.",
		&getCert)
	if err != nil {
		panic(err.Error())
	}

	listCert := certificateListCommand{certificateCommand: &certificate}
	_, err = p.AddCommand("list", "List certificates",
		"List the certificate entries.",
		&listCert)
	if err != nil {
		panic(err.Error())
	}

	createCert := certificateCreateCommand{certificateCommand: &certificate}
	_, err = p.AddCommand("create", "Create certificate",
		"Create a certificate and add it to the database.",
		&createCert)
	if err != nil {
		panic(err.Error())
	}
}

type certificateCommand struct {
	Partner string `required:"true" short:"p" long:"partner" description:"The name of the partner the account certificate belongs to"`
	Account string `required:"true" short:"a" long:"account" description:"The username of the account the certificate belongs to"`
}

func displayCertificate(out *os.File, certificate *model.CertChain) error {
	w := getColorable(out)

	fmt.Fprintf(w, "\033[97;1mCertificate '%s':\033[0m\n", certificate.Name)
	fmt.Fprintf(w, "├─\033[97mPrivate Key:\033[0m \033[37m%s\033[0m\n",
		string(certificate.PrivateKey))
	fmt.Fprintf(w, "├─\033[97mPublic Key:\033[0m \033[37m%v\033[0m\n",
		string(certificate.PublicKey))
	fmt.Fprintf(w, "├─\033[97mPrivate Cert:\033[0m \033[37m%v\033[0m\n",
		string(certificate.PrivateCert))
	fmt.Fprintf(w, "└─\033[97mPublic Cert:\033[0m \033[37m%s\033[0m\n",
		string(certificate.PublicCert))
	return nil
}

// ############################## GET #####################################

type certificateGetCommand struct {
	*certificateCommand `no-flag:"true"`
}

func (c *certificateGetCommand) getCertificate(in *os.File, out *os.File, name string) (*model.CertChain, error) {
	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + c.Partner +
		admin.AccountsURI + "/" + c.Account + admin.CertsURI + "/" + name

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
	partner := &model.CertChain{}
	if err := json.Unmarshal(body, partner); err != nil {
		return nil, err
	}

	return partner, nil
}

// Execute executes the 'certificate' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate name")
	}

	cert, err := c.getCertificate(os.Stdin, os.Stdout, args[0])
	if err != nil {
		return err
	}

	fmt.Println()
	if err := displayCertificate(os.Stdout, cert); err != nil {
		return err
	}

	return nil
}

// ############################### LIST #######################################

type certificateListCommand struct {
	*certificateCommand `no-flag:"true"`
	Limit               int    `short:"l" long:"limit" description:"The max number of entries which can be returned" default:"20"`
	Offset              int    `short:"o" long:"offset" description:"The offset from which the first entry is taken" default:"0"`
	Sort                string `short:"s" long:"sort" description:"The parameter used to sort the returned entries" choice:"name" default:"name"`
	Reverse             bool   `short:"d" long:"descending" description:"If present, the order of the sorting will be reversed"`
}

func (c *certificateListCommand) listCertificates(in *os.File, out *os.File) ([]byte, error) {
	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + c.Partner +
		admin.AccountsURI + "/" + c.Account + admin.CertsURI

	addr += "?limit=" + strconv.Itoa(c.Limit)
	addr += "&offset=" + strconv.Itoa(c.Offset)
	addr += "&sortby=" + c.Sort
	if c.Reverse {
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

// Execute executes the 'certificate' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateListCommand) Execute(_ []string) error {
	content, err := c.listCertificates(os.Stdin, os.Stdout)
	if err != nil {
		return err
	}

	certificates := map[string][]*model.CertChain{}
	if err := json.Unmarshal(content, &certificates); err != nil {
		return err
	}

	fmt.Println()
	for _, certificate := range certificates["certificates"] {
		if err := displayCertificate(os.Stdout, certificate); err != nil {
			return err
		}
	}

	return nil
}

// ############################### CREATE #######################################

type certificateCreateCommand struct {
	*certificateCommand `no-flag:"true"`
	Name                string `required:"true" short:"n" long:"name" description:"The certificate's name"`
	PrivateKey          string `long:"private_key" description:"The certificate's password"`
	PublicKey           string `long:"public_key" description:"The certificate's password"`
	PrivateCert         string `long:"private_cert" description:"The certificate's password"`
	PublicCert          string `long:"public_cert" description:"The certificate's password"`
}

// Execute executes the 'create' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateCreateCommand) Execute(_ []string) error {
	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + c.Partner +
		admin.AccountsURI + "/" + c.Account + admin.CertsURI

	certificate := &model.CertChain{
		Name:        c.Name,
		PrivateKey:  []byte(c.PrivateKey),
		PublicKey:   []byte(c.PublicKey),
		PrivateCert: []byte(c.PrivateCert),
		PublicCert:  []byte(c.PublicCert),
	}

	path, err := sendBean(certificate, os.Stdin, os.Stdout, addr, http.MethodPost)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mCertificate successfully created at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

//################################### UPDATE ######################################

type certificateUpdateCommand struct {
	*certificateCommand `no-flag:"true"`
	Name                string `short:"n" long:"name" description:"The certificate's name"`
	PrivateKey          string `long:"private_key" description:"The certificate's password"`
	PublicKey           string `long:"public_key" description:"The certificate's password"`
	PrivateCert         string `long:"private_cert" description:"The certificate's password"`
	PublicCert          string `long:"public_cert" description:"The certificate's password"`
}

// Execute executes the 'update' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateUpdateCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate name")
	}

	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + c.Partner +
		admin.AccountsURI + "/" + c.Account + admin.CertsURI + "/" + args[0]

	certificate := &model.CertChain{
		Name:        c.Name,
		PrivateKey:  []byte(c.PrivateKey),
		PublicKey:   []byte(c.PublicKey),
		PrivateCert: []byte(c.PrivateCert),
		PublicCert:  []byte(c.PublicCert),
	}

	path, err := sendBean(certificate, os.Stdin, os.Stdout, addr, http.MethodPatch)
	if err != nil {
		return err
	}

	w := getColorable(os.Stdout)
	fmt.Fprintf(w, "\033[97mCertificate successfully updated at:\033[0m \033[34;4m%s\033[0m\n",
		auth.Address+path)
	return nil
}

// ############################## DELETE #####################################

type certificateDeleteCommand struct {
	*certificateCommand `no-flag:"true"`
}

// Execute executes the 'certificate' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate name")
	}

	addr := auth.Address + admin.RestURI + admin.PartnersURI + "/" + c.Partner +
		admin.AccountsURI + "/" + c.Account + admin.CertsURI + "/" + args[0]

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
	fmt.Fprintf(w, "\033[97mCertificate\033[0m \033[33;1m'%s'\033[0m"+
		" \033[97msuccessfully deleted\033[0m\n", args[0])
	return nil
}
