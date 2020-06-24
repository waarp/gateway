package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/admin/rest"
	"github.com/jessevdk/go-flags"
)

type certificateCommand struct {
	Get    certGet    `command:"get" description:"Retrieve a certificate's information"`
	Add    certAdd    `command:"add" description:"Add a new certificate"`
	Delete certDelete `command:"delete" description:"Delete a certificate"`
	List   certList   `command:"list" description:"List the known certificates"`
	Update certUpdate `command:"update" description:"Update an existing certificate"`
}

func displayCertificate(w io.Writer, cert *rest.OutCert) {
	fmt.Fprintln(w, orange(bold("● Certificate", cert.Name)))
	fmt.Fprintln(w, orange("    Private key:"), string(cert.PrivateKey))
	fmt.Fprintln(w, orange("    Public key: "), string(cert.PublicKey))
	fmt.Fprintln(w, orange("    Content:    "), string(cert.Certificate))
}

func getCertPath() string {
	if partner := commandLine.Partner.Cert.Args.Partner; partner != "" {
		return fmt.Sprintf("/api/partners/%s", partner)
	} else if server := commandLine.Server.Cert.Args.Server; server != "" {
		return fmt.Sprintf("/api/servers/%s", server)
	} else if partner := commandLine.Account.Remote.Args.Partner; partner != "" {
		account := commandLine.Account.Remote.Cert.Args.Account
		return fmt.Sprintf("/api/partners/%s/accounts/%s", partner, account)
	} else if server := commandLine.Account.Local.Args.Server; server != "" {
		account := commandLine.Account.Local.Cert.Args.Account
		return fmt.Sprintf("/api/servers/%s/accounts/%s", server, account)
	} else {
		panic("unknown certificate recipient")
	}
}

// ######################## GET ##########################

type certGet struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
}

func (c *certGet) Execute([]string) error {
	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	cert := &rest.OutCert{}
	if err := get(cert); err != nil {
		return err
	}
	displayCertificate(getColorable(), cert)
	return nil
}

// ######################## ADD ##########################

type certAdd struct {
	Name        string         `required:"true" short:"n" long:"name" description:"The certificate's name"`
	PrivateKey  flags.Filename `short:"p" long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   flags.Filename `short:"b" long:"public_key" description:"The path to the certificate's public key file"`
	Certificate flags.Filename `short:"c" long:"certificate" description:"The path to the certificate file"`
}

func (c *certAdd) Execute([]string) (err error) {
	cert := &rest.InCert{
		Name: &c.Name,
	}
	if c.PrivateKey != "" {
		cert.PrivateKey, err = ioutil.ReadFile(string(c.PrivateKey))
		if err != nil {
			return err
		}
	}
	if c.PublicKey != "" {
		cert.PublicKey, err = ioutil.ReadFile(string(c.PublicKey))
		if err != nil {
			return err
		}
	}
	if c.Certificate != "" {
		cert.Certificate, err = ioutil.ReadFile(string(c.Certificate))
		if err != nil {
			return err
		}
	}

	addr.Path = path.Join(getCertPath(), "certificates")

	if err := add(cert); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The certificate", bold(c.Name), "was successfully added.")
	return nil
}

// ######################## DELETE ##########################

type certDelete struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
}

func (c *certDelete) Execute([]string) error {
	uri := path.Join(getCertPath(), "certificates", c.Args.Cert)

	if err := remove(uri); err != nil {
		return err
	}
	fmt.Fprintln(getColorable(), "The certificate", c.Args.Cert, "was successfully deleted.")

	return nil
}

// ######################## LIST ##########################

type certList struct {
	listOptions
	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (c *certList) Execute([]string) error {
	addr.Path = path.Join(getCertPath(), "certificates")
	listURL(&c.listOptions, c.SortBy)

	body := map[string][]rest.OutCert{}
	if err := list(&body); err != nil {
		return err
	}

	certs := body["certificates"]
	w := getColorable()
	if len(certs) > 0 {
		fmt.Fprintln(w, bold("Certificates:"))
		for _, c := range certs {
			cert := c
			displayCertificate(w, &cert)
		}
	} else {
		fmt.Fprintln(w, "No certificates found.")
	}

	return nil
}

// ######################## UPDATE ##########################

type certUpdate struct {
	Args struct {
		Cert string `required:"yes" positional-arg-name:"cert" description:"The certificate's name"`
	} `positional-args:"yes"`
	Name        *string        `short:"n" long:"name" description:"The certificate's name"`
	PrivateKey  flags.Filename `short:"p" long:"private_key" description:"The path to the certificate's private key file"`
	PublicKey   flags.Filename `short:"b" long:"public_key" description:"The path to the certificate's public key file"`
	Certificate flags.Filename `short:"c" long:"certificate" description:"The path to the certificate file"`
}

func (c *certUpdate) Execute([]string) (err error) {
	cert := &rest.InCert{
		Name: c.Name,
	}
	if c.PrivateKey != "" {
		cert.PrivateKey, err = ioutil.ReadFile(string(c.PrivateKey))
		if err != nil {
			return err
		}
	}
	if c.PublicKey != "" {
		cert.PublicKey, err = ioutil.ReadFile(string(c.PublicKey))
		if err != nil {
			return err
		}
	}
	if c.Certificate != "" {
		cert.Certificate, err = ioutil.ReadFile(string(c.Certificate))
		if err != nil {
			return err
		}
	}

	addr.Path = path.Join(getCertPath(), "certificates", c.Args.Cert)

	if err := update(cert); err != nil {
		return err
	}
	name := c.Args.Cert
	if cert.Name != nil && *cert.Name != "" {
		name = *cert.Name
	}
	fmt.Fprintln(getColorable(), "The certificate", bold(name), "was successfully updated.")

	return nil
}
