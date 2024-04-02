package wg

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/jessevdk/go-flags"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

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

func (c *CertGet) Execute([]string) error { return c.execute(stdOutput) }
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

type CertAdd struct {
	Name        string         `required:"true" short:"n" long:"name" description:"The certificate's name"`
	PrivateKey  flags.Filename `short:"p" long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   flags.Filename `short:"b" long:"public_key" description:"The path to the certificate's public key file"`
	Certificate flags.Filename `short:"c" long:"certificate" description:"The path to the certificate file"`
}

func (c *CertAdd) Execute([]string) error { return c.execute(stdOutput) }
func (c *CertAdd) execute(w io.Writer) error {
	inCrypto := &api.InCrypto{Name: &c.Name}

	if c.PrivateKey != "" {
		pk, err := os.ReadFile(string(c.PrivateKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PrivateKey, err)
		}

		inCrypto.PrivateKey = utils.StringPtr(string(pk))
	}

	if c.PublicKey != "" {
		pbk, err := os.ReadFile(string(c.PublicKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PublicKey, err)
		}

		inCrypto.PublicKey = utils.StringPtr(string(pbk))
	}

	if c.Certificate != "" {
		cert, err := os.ReadFile(string(c.Certificate))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.Certificate, err)
		}

		inCrypto.Certificate = utils.StringPtr(string(cert))
	}

	addr.Path = path.Join(getCertPath(), "certificates")

	if _, err := add(w, inCrypto); err != nil {
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

func (c *CertDelete) Execute([]string) error { return c.execute(stdOutput) }
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

func (c *CertList) Execute([]string) error { return c.execute(stdOutput) }
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

type CertUpdate struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
	Name        *string        `short:"n" long:"name" description:"The certificate's name"`
	PrivateKey  flags.Filename `short:"p" long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   flags.Filename `short:"b" long:"public_key" description:"The path to the certificate's public key file"`
	Certificate flags.Filename `short:"c" long:"certificate" description:"The path to the certificate file"`
}

func (c *CertUpdate) Execute([]string) error { return c.execute(stdOutput) }
func (c *CertUpdate) execute(w io.Writer) error {
	inCrypto := &api.InCrypto{
		Name: c.Name,
	}

	if c.PrivateKey != "" {
		pk, err := os.ReadFile(string(c.PrivateKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PrivateKey, err)
		}

		inCrypto.PrivateKey = utils.StringPtr(string(pk))
	}

	if c.PublicKey != "" {
		pbk, err := os.ReadFile(string(c.PublicKey))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.PublicKey, err)
		}

		inCrypto.PublicKey = utils.StringPtr(string(pbk))
	}

	if c.Certificate != "" {
		cert, err := os.ReadFile(string(c.Certificate))
		if err != nil {
			return fmt.Errorf("cannot read file %q: %w", c.Certificate, err)
		}

		inCrypto.Certificate = utils.StringPtr(string(cert))
	}

	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	if err := update(w, inCrypto); err != nil {
		return err
	}

	name := c.Args.Cert
	if inCrypto.Name != nil && *inCrypto.Name != "" {
		name = *inCrypto.Name
	}

	fmt.Fprintf(w, "The certificate %q was successfully updated.\n", name)

	return nil
}
