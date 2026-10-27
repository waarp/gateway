package wg

import (
	"errors"
	"fmt"
	"io"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/r66"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/modules/sftp"
)

var errUnknownCredentialRecipient = errors.New("unknown credential recipient")

func getCredentialPath() (string, []string, error) {
	switch {
	case Partner != "" && RemoteAccount != "":
		return "/api/partners/%s/accounts/%s", []string{Partner, RemoteAccount}, nil
	case Server != "" && LocalAccount != "":
		return "/api/servers/%s/accounts/%s", []string{Server, LocalAccount}, nil
	case Partner != "":
		return "/api/partners/%s", []string{Partner}, nil
	case Server != "":
		return "/api/servers/%s", []string{Server}, nil
	default:
		return "", nil, errUnknownCredentialRecipient
	}
}

func setCredentialAddrPath(suffix string, args ...string) error {
	base, baseArgs, err := getCredentialPath()
	if err != nil {
		return err
	}

	SetAddrPath(base+suffix, append(baseArgs, args...)...)

	return nil
}

func displayCredential(w io.Writer, cred *api.OutCred) error {
	switch cred.Type {
	case auth.Password:
		Style1.PrintL(w, fmt.Sprintf("Password %q", cred.Name), cred.Value)
	case auth.TLSCertificate, auth.TLSTrustedCertificate:
		return displayTLSInfo(w, Style1, cred.Name, cred.Value)
	case sftp.AuthSSHPublicKey:
		return displaySSHKeyInfo(w, Style1, cred.Name, cred.Value)
	case sftp.AuthSSHPrivateKey:
		return displayPrivateKeyInfo(w, Style1, cred.Name, cred.Value)
	case r66.AuthLegacyCertificate:
		Style1.Printf(w, "Legacy R66 certificate %q", cred.Name)
	default:
		//nolint:err113 //too specific
		return fmt.Errorf("unknown credential type %q", cred.Type)
	}

	return nil
}

//nolint:lll //flags tags are long
type CredentialAdd struct {
	Name   string          `short:"n" long:"name" description:"The credential's name" json:"name,omitempty"`
	Type   string          `required:"true" short:"t" long:"type" description:"The type of credential" json:"type,omitempty"`
	Value  textFileOrValue `short:"v" long:"value" description:"The credential value. Can also be a file path." json:"value,omitzero"`
	Value2 textFileOrValue `short:"s" long:"secondary-value" description:"The secondary credential value (when applicable). Can also be a filepath." json:"value2,omitzero"`
}

func (a *CredentialAdd) Execute([]string) error { return a.execute(stdOutput) }
func (a *CredentialAdd) execute(w io.Writer) error {
	if pathErr := setCredentialAddrPath("/credentials"); pathErr != nil {
		return pathErr
	}

	if _, err := add(w, a); err != nil {
		return err
	}

	name := a.Name
	if name == "" {
		name = a.Type
	}

	fmt.Fprintf(w, "The %q credential was successfully added.\n", name)

	return nil
}

type CredentialGet struct {
	OutputFormat

	Args struct {
		Credential string `required:"yes" positional-arg-name:"credential" description:"The credential's name"`
	} `positional-args:"yes"`
}

func (a *CredentialGet) Execute([]string) error { return a.execute(stdOutput) }
func (a *CredentialGet) execute(w io.Writer) error {
	if pathErr := setCredentialAddrPath("/credentials/%s", a.Args.Credential); pathErr != nil {
		return pathErr
	}

	var cred api.OutCred
	if err := get(&cred); err != nil {
		return err
	}

	return outputObject(w, &cred, &a.OutputFormat, displayCredential)
}

type CredentialDelete struct {
	Args struct {
		Credential string `required:"yes" positional-arg-name:"credential" description:"The credential's name"`
	} `positional-args:"yes"`
}

func (a *CredentialDelete) Execute([]string) error { return a.execute(stdOutput) }
func (a *CredentialDelete) execute(w io.Writer) error {
	if pathErr := setCredentialAddrPath("/credentials/%s", a.Args.Credential); pathErr != nil {
		return pathErr
	}

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The %q credential was successfully removed.\n", a.Args.Credential)

	return nil
}
