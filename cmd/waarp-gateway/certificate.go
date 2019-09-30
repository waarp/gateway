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

type certificateCommand struct{}

func displayCertificate(out *os.File, certificate model.CertChain) {
	w := getColorable(out)

	fmt.Fprintf(w, "\033[97;1mCertificate n°%v:\033[0m\n", certificate.ID)
	fmt.Fprintf(w, "├─\033[97mName:\033[0m \033[37m%s\033[0m\n", certificate.Name)
	fmt.Fprintf(w, "├─\033[97mAccountID:\033[0m \033[37m%v\033[0m\n", certificate.OwnerID)
	fmt.Fprintf(w, "├─\033[97mPrivate Key:\033[0m \033[37m%s\033[0m\n",
		string(certificate.PrivateKey))
	fmt.Fprintf(w, "├─\033[97mPublic Key:\033[0m \033[37m%v\033[0m\n",
		string(certificate.PublicKey))
	fmt.Fprintf(w, "└─\033[97mCert:\033[0m \033[37m%v\033[0m\n",
		string(certificate.Cert))
}

// ############################## GET #####################################

type certificateGetCommand struct{}

// Execute executes the 'certificate' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateGetCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate name")
	}

	subpath := admin.CertsURI + "/" + args[0]
	cert := model.CertChain{}
	err := getCommand(os.Stdin, os.Stdout, subpath, &cert)
	if err != nil {
		return err
	}

	fmt.Println()
	displayCertificate(os.Stdout, cert)

	return nil
}

// ############################### LIST #######################################

type certificateListCommand struct {
	Limit   int    `short:"l" long:"limit" description:"The max number of entries which can be returned" default:"20"`
	Offset  int    `short:"o" long:"offset" description:"The offset from which the first entry is taken" default:"0"`
	Sort    string `short:"s" long:"sort" description:"The parameter used to sort the returned entries" choice:"name" default:"name"`
	Reverse bool   `short:"d" long:"descending" description:"If present, the order of the sorting will be reversed"`
}

func (c *certificateListCommand) listCertificates(in *os.File, out *os.File) ([]byte, error) {
	addr := auth.Address + admin.RestURI + admin.CertsURI

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

	certificates := map[string][]model.CertChain{}
	if err := json.Unmarshal(content, &certificates); err != nil {
		return err
	}

	fmt.Println()
	for _, certificate := range certificates["certificates"] {
		displayCertificate(os.Stdout, certificate)
	}

	return nil
}

// ############################### CREATE #######################################

type certificateCreateCommand struct {
	Name       string `required:"true" short:"n" long:"name" description:"The certificate's name"`
	OwnerType  string `required:"true" short:"t" long:"type" description:"The type of owner the certificate is attached to (ACCOUNT or PARTNER)"`
	OwnerID    uint64 `required:"true" short:"i" long:"owner_id" description:"The account to which the certificate will be attached"`
	PrivateKey string `long:"private_key" description:"The private key"`
	PublicKey  string `long:"public_key" description:"The public key"`
	Cert       string `long:"cert" description:"The public key certificate"`
}

// Execute executes the 'create' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateCreateCommand) Execute(_ []string) error {
	addr := auth.Address + admin.RestURI + admin.CertsURI

	certificate := model.CertChain{
		Name:       c.Name,
		OwnerType:  c.OwnerType,
		OwnerID:    c.OwnerID,
		PrivateKey: []byte(c.PrivateKey),
		PublicKey:  []byte(c.PublicKey),
		Cert:       []byte(c.Cert),
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
	Name       string `short:"n" long:"name" description:"The certificate's name"`
	OwnerType  string `short:"t" long:"type" description:"The type of owner the certificate is attached to (ACCOUNT or PARTNER)"`
	OwnerID    uint64 `short:"i" long:"account_id" description:"The account to which the certificate will be attached"`
	PrivateKey string `long:"private_key" description:"The private key"`
	PublicKey  string `long:"public_key" description:"The public key"`
	Cert       string `long:"cert" description:"The public key certificate"`
}

// Execute executes the 'update' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateUpdateCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate name")
	}

	addr := auth.Address + admin.RestURI + admin.CertsURI + "/" + args[0]

	certificate := &struct {
		Name                        string
		OwnerType                   string
		OwnerID                     uint64
		PrivateKey, PublicKey, Cert []byte
	}{
		Name:       c.Name,
		OwnerType:  c.OwnerType,
		OwnerID:    c.OwnerID,
		PrivateKey: []byte(c.PrivateKey),
		PublicKey:  []byte(c.PublicKey),
		Cert:       []byte(c.Cert),
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

type certificateDeleteCommand struct{}

// Execute executes the 'certificate' command. The command flags are stored in
// the 's' parameter, while the program arguments are stored in the 'args'
// parameter.
func (c *certificateDeleteCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("missing certificate name")
	}

	addr := auth.Address + admin.RestURI + admin.CertsURI + "/" + args[0]

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
