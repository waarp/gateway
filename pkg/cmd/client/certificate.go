package wg

import (
	"fmt"
	"io"
	"path"

	"github.com/jedib0t/go-pretty/v6/text"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const certDeprecatedMsg = `‼WARNING‼ The "certificate" command is deprecated.` +
	`Please use the "credentials" command instead.`

func warnCertDeprecated() {
	fmt.Fprintln(asColorable(stdOutput), text.FgRed.Sprint(certDeprecatedMsg))
}

func DisplayCrypto(w io.Writer, cert *api.OutCrypto) {
	f := NewFormatter(w)
	defer f.Render()

	displayCrypto(f, cert)
}

func displayCrypto(f *Formatter, cert *api.OutCrypto) {
	switch {
	case cert.Certificate != "":
		displayTLSInfo(f, cert.Name, cert.Certificate)
	case cert.PublicKey != "":
		displaySSHKeyInfo(f, cert.Name, cert.PublicKey)
	case cert.PrivateKey != "":
		displayPrivateKeyInfo(f, cert.Name, cert.PrivateKey)
	default:
		f.Title("Entry %q: <unknown authentication type>", cert.Name)
	}
}

func getCertPath() string {
	switch {
	case LocalAccount != "":
		return fmt.Sprintf("/api/servers/%s/accounts/%s", Server, LocalAccount)
	case RemoteAccount != "":
		return fmt.Sprintf("/api/partners/%s/accounts/%s", Partner, RemoteAccount)
	case Server != "":
		return fmt.Sprintf("/api/servers/%s", Server)
	case Partner != "":
		return fmt.Sprintf("/api/partners/%s", Partner)
	default:
		panic("unknown certificate recipient")
	}
}

// ######################## GET ##########################

type CertGet struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
}

func (c *CertGet) Execute([]string) error {
	warnCertDeprecated()

	return c.execute(stdOutput)
}

func (c *CertGet) execute(w io.Writer) error {
	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	cert := &api.OutCrypto{}
	if err := get(cert); err != nil {
		return err
	}

	DisplayCrypto(w, cert)

	return nil
}

// ######################## ADD ##########################

// TODO: replace underscores "_" with hyphens "-" in flags names.

//nolint:lll //tags are long
type CertAdd struct {
	Name        string `required:"true" short:"n" long:"name" description:"The certificate's name" json:"name,omitempty"`
	PrivateKey  file   `short:"p" long:"private_key" description:"The path to the certificate's private key file" json:"privateKey,omitempty"`
	PublicKey   file   `short:"b" long:"public_key" description:"The path to the certificate's public key file" json:"publicKey,omitempty"`
	Certificate file   `short:"c" long:"certificate" description:"The path to the certificate file" json:"certificate,omitempty"`
}

func (c *CertAdd) Execute([]string) error {
	warnCertDeprecated()

	return c.execute(stdOutput)
}

func (c *CertAdd) execute(w io.Writer) error {
	addr.Path = path.Join(getCertPath(), "certificates")

	if _, err := add(w, c); err != nil {
		return err
	}

	fmt.Fprintf(w, "The certificate %q was successfully added.\n", c.Name)

	return nil
}

// ######################## DELETE ##########################

type CertDelete struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
}

func (c *CertDelete) Execute([]string) error {
	warnCertDeprecated()

	return c.execute(stdOutput)
}

func (c *CertDelete) execute(w io.Writer) error {
	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The certificate %q was successfully deleted.\n", c.Args.Cert)

	return nil
}

// ######################## LIST ##########################

//nolint:lll // struct tags for command line arguments can be long
type CertList struct {
	ListOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (c *CertList) Execute([]string) error {
	warnCertDeprecated()

	return c.execute(stdOutput)
}

func (c *CertList) execute(w io.Writer) error {
	addr.Path = path.Join(getCertPath(), "certificates")

	listURL(&c.ListOptions, c.SortBy)

	body := map[string][]*api.OutCrypto{}
	if err := list(&body); err != nil {
		return err
	}

	if certs := body["certificates"]; len(certs) > 0 {
		f := NewFormatter(w)
		defer f.Render()

		f.MainTitle("Certificates:")

		for _, cert := range certs {
			displayCrypto(f, cert)
		}
	} else {
		fmt.Fprintln(w, "No certificates found.")
	}

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll //tags are long
type CertUpdate struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes" json:"-"`
	Name        string `short:"n" long:"name" description:"The certificate's name" json:"name,omitempty"`
	PrivateKey  file   `short:"p" long:"private_key" description:"The path to the certificate's private key file" json:"privateKey,omitempty"`
	PublicKey   file   `short:"b" long:"public_key" description:"The path to the certificate's public key file" json:"publicKey,omitempty"`
	Certificate file   `short:"c" long:"certificate" description:"The path to the certificate file" json:"certificate,omitempty"`
}

func (c *CertUpdate) Execute([]string) error {
	warnCertDeprecated()

	return c.execute(stdOutput)
}

func (c *CertUpdate) execute(w io.Writer) error {
	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	if err := update(w, c); err != nil {
		return err
	}

	name := c.Args.Cert
	if c.Name != "" {
		name = c.Name
	}

	fmt.Fprintf(w, "The certificate %q was successfully updated.\n", name)

	return nil
}
