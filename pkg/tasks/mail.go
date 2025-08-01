package tasks

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"path"

	"code.waarp.fr/lib/log"
	"gopkg.in/gomail.v2"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var (
	ErrMailNoTemplate   = errors.New("missing email message template")
	ErrMailNoRecipients = errors.New("missing email recipients")
	ErrMailNoAuth       = errors.New("missing email authentication")
)

type mailTask struct {
	Sender     string   `json:"sender"`
	Recipients jsonList `json:"recipients"`
	Template   string   `json:"template"`

	connInfo model.SMTPCredential
	template model.EmailTemplate
}

func (m *mailTask) parseParams(db database.ReadAccess, params map[string]string) error {
	if err := utils.JSONConvert(params, m); err != nil {
		return fmt.Errorf("failed to parse MAIL task params: %w", err)
	}

	if m.Sender == "" {
		return ErrMailNoAuth
	}

	if m.Recipients == nil {
		return ErrMailNoRecipients
	}

	if m.Template == "" {
		return ErrMailNoTemplate
	}

	if err := db.Get(&m.connInfo, "email_address=?", m.Sender).Owner().Run(); err != nil {
		return fmt.Errorf("failed to retrieve SMTP credential: %w", err)
	}

	if err := db.Get(&m.template, "name=?", m.Template).Run(); err != nil {
		return fmt.Errorf("failed to retrieve email template: %w", err)
	}

	return nil
}

func (m *mailTask) ValidateDB(db database.ReadAccess, args map[string]string) error {
	return m.parseParams(db, args)
}

func (m *mailTask) Run(_ context.Context, params map[string]string, db *database.DB,
	logger *log.Logger, transCtx *model.TransferContext,
) error {
	if err := m.parseParams(db, params); err != nil {
		logger.Errorf("%v", err)

		return err
	}

	if err := m.replaceTemplateVars(transCtx); err != nil {
		logger.Errorf("Failed to replace email template variables: %v", err)

		return fmt.Errorf("failed to replace email template variables: %w", err)
	}

	mail := gomail.NewMessage()
	mail.SetHeader("From", m.connInfo.EmailAddress)
	mail.SetHeader("To", m.Recipients...)
	mail.SetHeader("Subject", m.template.Subject)
	mail.SetBody("text/plain", m.template.Body)

	for _, attachement := range m.template.Attachments {
		mail.Attach(path.Base(attachement), m.copyFileFunc(attachement))
	}

	d := gomail.NewDialer(m.connInfo.ServerAddress.Host, int(m.connInfo.ServerAddress.Port),
		m.connInfo.Login, string(m.connInfo.Password))
	d.TLSConfig = &tls.Config{ServerName: m.connInfo.ServerAddress.Host}

	if err := auth.AddTLSAuthorities(db, d.TLSConfig); err != nil {
		logger.Warningf("Failed to add TLS authorities: %v", err)
	}

	if err := d.DialAndSend(mail); err != nil {
		logger.Errorf("Failed to send email: %v", err)

		return fmt.Errorf("failed to send email: %w", err)
	}

	logger.Debugf("Sent mail from %q to %q", m.connInfo.EmailAddress, m.Recipients)

	return nil
}

func (*mailTask) copyFileFunc(name string) gomail.FileSetting {
	//nolint:wrapcheck //wrapping adds nothing here
	return gomail.SetCopyFunc(func(w io.Writer) error {
		h, opErr := fs.Open(name)
		if opErr != nil {
			return opErr
		}
		defer h.Close()

		if _, err := io.Copy(w, h); err != nil {
			return err
		}

		return h.Close()
	})
}

func (m *mailTask) replaceTemplateVars(transCtx *model.TransferContext) error {
	var err error

	if m.template.Subject, err = replaceVars(m.template.Subject, transCtx); err != nil {
		return err
	}

	if m.template.Body, err = replaceVars(m.template.Body, transCtx); err != nil {
		return err
	}

	for i, attachement := range m.template.Attachments {
		if m.template.Attachments[i], err = replaceVars(attachement, transCtx); err != nil {
			return err
		}
	}

	return nil
}
