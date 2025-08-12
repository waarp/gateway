package wg

import (
	"fmt"
	"io"
	"path"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

// #####################################################################################################################
// ################################################## EMAIL TEMPLATES ##################################################
// #####################################################################################################################

func displayEmailTemplate(w io.Writer, template *api.GetEmailTemplateObject) {
	Style1.Printf(w, "Email template %q", template.Name)
	Style22.PrintL(w, "Subject", template.Subject)
	Style22.PrintL(w, "MIME type", template.MIMEType)
	Style22.MultiL(w, "Body", template.Body)
	Style22.Defaul(w, "Attachments", strings.Join(template.Attachments, ", "), none)
}

// ######################## ADD ##########################

//nolint:lll //struct tags are long
type EmailTemplateAdd struct {
	Name        string      `required:"yes" short:"n" long:"name" description:"The template's name" json:"name,omitempty"`
	Subject     string      `required:"yes" short:"s" long:"subject" description:"The email's subject" json:"subject,omitempty"`
	MIMEType    string      `short:"m" long:"mime-type" description:"The email's MIME type" default:"text/plain" json:"mimeType,omitempty"`
	Body        fileOrValue `required:"yes" short:"b" long:"body" description:"The email's body" json:"body,omitempty"`
	Attachments []string    `short:"a" long:"attachments" description:"The email's attachments. Can be repeated." json:"attachments,omitempty"`
}

func (e *EmailTemplateAdd) Execute([]string) error { return execute(e) }
func (e *EmailTemplateAdd) execute(w io.Writer) error {
	addr.Path = "/api/email/templates"

	if _, err := add(w, e); err != nil {
		return err
	}

	fmt.Fprintf(w, "The email template %q was successfully added.\n", e.Name)

	return nil
}

// ######################## LIST ##########################

//nolint:lll //struct tags are long
type EmailTemplateList struct {
	ListOptions

	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"name+" choice:"name-" default:"name+"`
}

func (e *EmailTemplateList) Execute([]string) error { return execute(e) }
func (e *EmailTemplateList) execute(w io.Writer) error {
	addr.Path = "/api/email/templates"

	listURL(&e.ListOptions, e.SortBy)

	var body map[string][]*api.GetEmailTemplateObject
	if err := list(&body); err != nil {
		return err
	}

	if templates := body["emailTemplates"]; len(templates) > 0 {
		Style0.PrintV(w, "=== Email templates ===")

		for _, template := range templates {
			displayEmailTemplate(w, template)
		}
	} else {
		fmt.Fprintln(w, "No email templates found.")
	}

	return nil
}

// ######################## GET ##########################

type EmailTemplateGet struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The template's name"`
	} `positional-args:"yes"`
}

func (e *EmailTemplateGet) Execute([]string) error { return execute(e) }
func (e *EmailTemplateGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/email/templates", e.Args.Name)

	var template api.GetEmailTemplateObject
	if err := get(&template); err != nil {
		return err
	}

	displayEmailTemplate(w, &template)

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll //struct tags are long
type EmailTemplateUpdate struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The old template's name"`
	} `positional-args:"yes" json:"-"`

	Name        string      `short:"n" long:"name" description:"The template's name" json:"name,omitempty"`
	Subject     string      `short:"s" long:"subject" description:"The email's subject" json:"subject,omitempty"`
	MIMEType    string      `short:"m" long:"mime-type" description:"The email's MIME type" json:"mimeType,omitempty"`
	Body        fileOrValue `short:"b" long:"body" description:"The email's body" json:"body,omitempty"`
	Attachments []string    `short:"a" long:"attachments" description:"The email's attachments. Can be repeated." json:"attachments,omitempty"`
}

func (e *EmailTemplateUpdate) Execute([]string) error { return execute(e) }
func (e *EmailTemplateUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/email/templates", e.Args.Name)

	if err := update(w, e); err != nil {
		return err
	}

	name := e.Args.Name
	if e.Name != "" {
		name = e.Name
	}

	fmt.Fprintf(w, "The email template %q was successfully updated.\n", name)

	return nil
}

// ######################## DELETE ##########################

type EmailTemplateDelete struct {
	Args struct {
		Name string `required:"yes" positional-arg-name:"name" description:"The template's name"`
	} `positional-args:"yes"`
}

func (e *EmailTemplateDelete) Execute([]string) error { return execute(e) }
func (e *EmailTemplateDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/email/templates", e.Args.Name)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The email template %q was successfully deleted.\n", e.Args.Name)

	return nil
}

// ####################################################################################################################
// ################################################# SMTP CREDENTIALS #################################################
// ####################################################################################################################

func displaySMTPCredential(w io.Writer, cred *api.GetSMTPCredentialObject) {
	Style1.Printf(w, "SMTP credential %q", cred.EmailAddress)
	Style22.PrintL(w, "Server address", cred.ServerAddress)
	Style22.Defaul(w, "Login", cred.Login, none)
	Style22.Defaul(w, "Password", cred.Password, none)
}

// ######################## ADD ##########################

//nolint:lll //struct tags are long
type SMTPCredentialAdd struct {
	EmailAddress  string `required:"yes" short:"e" long:"email-address" description:"The email address" json:"emailAddress,omitempty"`
	ServerAddress string `required:"yes" short:"s" long:"server-address" description:"The SMTP server address" json:"serverAddress,omitempty"`
	Login         string `short:"l" long:"login" description:"The SMTP server login" json:"login,omitempty"`
	Password      string `short:"p" long:"password" description:"The SMTP password" json:"password,omitempty"`
}

func (e *SMTPCredentialAdd) Execute([]string) error { return execute(e) }
func (e *SMTPCredentialAdd) execute(w io.Writer) error {
	addr.Path = "/api/email/credentials"

	if _, err := add(w, e); err != nil {
		return err
	}

	fmt.Fprintf(w, "The SMTP credential %q was successfully added.\n", e.EmailAddress)

	return nil
}

// ######################## LIST ##########################

//nolint:lll //struct tags are long
type SMTPCredentialList struct {
	ListOptions

	SortBy string `short:"s" long:"sort" description:"Attribute used to sort the returned entries" choice:"email+" choice:"email-" default:"email+"`
}

func (e *SMTPCredentialList) Execute([]string) error { return execute(e) }
func (e *SMTPCredentialList) execute(w io.Writer) error {
	addr.Path = "/api/email/credentials"

	listURL(&e.ListOptions, e.SortBy)

	var body map[string][]*api.GetSMTPCredentialObject
	if err := list(&body); err != nil {
		return err
	}

	if credentials := body["smtpCredentials"]; len(credentials) > 0 {
		Style0.Printf(w, "=== SMTP credentials ===")

		for _, credential := range credentials {
			displaySMTPCredential(w, credential)
		}
	} else {
		fmt.Fprintln(w, "No SMTP credentials found.")
	}

	return nil
}

// ######################## GET ##########################

type SMTPCredentialGet struct {
	Args struct {
		Email string `required:"yes" positional-arg-name:"email-address" description:"The email address"`
	} `positional-args:"yes"`
}

func (e *SMTPCredentialGet) Execute([]string) error { return execute(e) }
func (e *SMTPCredentialGet) execute(w io.Writer) error {
	addr.Path = path.Join("/api/email/credentials", e.Args.Email)

	var credential api.GetSMTPCredentialObject
	if err := get(&credential); err != nil {
		return err
	}

	displaySMTPCredential(w, &credential)

	return nil
}

// ######################## UPDATE ##########################

//nolint:lll //struct tags are long
type SMTPCredentialUpdate struct {
	Args struct {
		Email string `required:"yes" positional-arg-name:"email" description:"The old email address"`
	} `positional-args:"yes" json:"-"`

	EmailAddress  string `short:"e" long:"email-address" description:"The email address" json:"emailAddress,omitempty"`
	ServerAddress string `short:"s" long:"server-address" description:"The SMTP server address" json:"serverAddress,omitempty"`
	Login         string `short:"l" long:"login" description:"The SMTP server login" json:"login,omitempty"`
	Password      string `short:"p" long:"password" description:"The SMTP password" json:"password,omitempty"`
}

func (e *SMTPCredentialUpdate) Execute([]string) error { return execute(e) }
func (e *SMTPCredentialUpdate) execute(w io.Writer) error {
	addr.Path = path.Join("/api/email/credentials", e.Args.Email)

	if err := update(w, e); err != nil {
		return err
	}

	email := e.Args.Email
	if e.EmailAddress != "" {
		email = e.EmailAddress
	}

	fmt.Fprintf(w, "The SMTP credential %q was successfully updated.\n", email)

	return nil
}

// ######################## DELETE ##########################

type SMTPCredentialDelete struct {
	Args struct {
		Email string `required:"yes" positional-arg-name:"email" description:"The email address"`
	} `positional-args:"yes"`
}

func (e *SMTPCredentialDelete) Execute([]string) error { return execute(e) }
func (e *SMTPCredentialDelete) execute(w io.Writer) error {
	addr.Path = path.Join("/api/email/credentials", e.Args.Email)

	if err := remove(w); err != nil {
		return err
	}

	fmt.Fprintf(w, "The SMTP credential %q was successfully deleted.\n", e.Args.Email)

	return nil
}
