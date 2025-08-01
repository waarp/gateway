package model

import (
	"fmt"
	"mime"
	"net/mail"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

const DefaultEmailMimeType = "text/plain"

var (
	ErrEmailTemplateNoName      = database.NewValidationError("the email template's name cannot be empty")
	ErrEmailTemplateNoSubj      = database.NewValidationError("the email template's subject cannot be empty")
	ErrEmailTemplateNoBody      = database.NewValidationError("the email template's body cannot be empty")
	ErrEmailTemplateInvalidMIME = database.NewValidationError("the email template's MIME type is invalid")
	ErrEmailTemplateDuplicate   = database.NewValidationError("an email template with the same name already exists")

	ErrSMTPCredentialNoSender      = database.NewValidationError("the SMTP credential's sender address cannot be empty")
	ErrSMTPCredentialInvalidSender = database.NewValidationError("the SMTP credential's sender address is invalid")
	ErrSMTPCredentialNoAddr        = database.NewValidationError("the SMTP credential's server address cannot be empty")
	ErrSMTPCredentialDuplicate     = database.NewValidationError("an SMTP credential with the same sender already exists")
)

type EmailTemplate struct {
	ID          int64    `xorm:"<- id AUTOINCR"`
	Name        string   `xorm:"name"`
	Subject     string   `xorm:"subject"`
	MIMEType    string   `xorm:"mime_type"`
	Body        string   `xorm:"body"`
	Attachments []string `xorm:"attachments"`
}

func (*EmailTemplate) TableName() string   { return TableEmailTemplates }
func (*EmailTemplate) Appellation() string { return NameEmailTemplate }
func (e *EmailTemplate) GetID() int64      { return e.ID }

func (e *EmailTemplate) BeforeWrite(db database.Access) error {
	if e.Name == "" {
		return ErrEmailTemplateNoName
	}

	if e.Subject == "" {
		return ErrEmailTemplateNoSubj
	}

	if e.Body == "" {
		return ErrEmailTemplateNoBody
	}

	if e.MIMEType == "" {
		e.MIMEType = DefaultEmailMimeType
	}

	if _, _, err := mime.ParseMediaType(e.MIMEType); err != nil {
		return fmt.Errorf("%w: %q", ErrEmailTemplateInvalidMIME, e.MIMEType)
	}

	if n, err := db.Count(e).Where("id<>? AND name=?", e.ID, e.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate email templates: %w", err)
	} else if n != 0 {
		return fmt.Errorf("%w: %q", ErrEmailTemplateDuplicate, e.Name)
	}

	return nil
}

type SMTPCredential struct {
	ID            int64               `xorm:"<- id AUTOINCR"`
	Owner         string              `xorm:"owner"`
	EmailAddress  string              `xorm:"email_address"`
	ServerAddress types.Address       `xorm:"server_address"`
	Login         string              `xorm:"login"`
	Password      database.SecretText `xorm:"password"`
}

func (*SMTPCredential) TableName() string   { return TableSMTPCredentials }
func (*SMTPCredential) Appellation() string { return NameSMTPCredential }
func (s *SMTPCredential) GetID() int64      { return s.ID }

func (s *SMTPCredential) BeforeWrite(db database.Access) error {
	s.Owner = conf.GlobalConfig.GatewayName

	if s.EmailAddress == "" {
		return ErrSMTPCredentialNoSender
	}

	if _, err := mail.ParseAddress(s.EmailAddress); err != nil {
		return fmt.Errorf("%w: %q", ErrSMTPCredentialInvalidSender, s.EmailAddress)
	}

	if !s.ServerAddress.IsSet() {
		return ErrSMTPCredentialNoAddr
	}

	if n, err := db.Count(s).Owner().Where("id<>?", s.ID).
		Where("email_address=?", s.EmailAddress).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate SMTP credentials: %w", err)
	} else if n > 0 {
		return fmt.Errorf("%q: %w", s.EmailAddress, ErrSMTPCredentialDuplicate)
	}

	return nil
}
