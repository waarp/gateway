package backup

import (
	"fmt"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func exportEmailConf(logger *log.Logger, db database.ReadAccess) (*file.EmailConfig, error) {
	credentials, err := exportSMTPCredentials(logger, db)
	if err != nil {
		return nil, err
	}

	templates, err := exportEmailTemplates(logger, db)
	if err != nil {
		return nil, err
	}

	return &file.EmailConfig{
		Credentials: credentials,
		Templates:   templates,
	}, nil
}

func exportSMTPCredentials(logger *log.Logger, db database.ReadAccess) ([]*file.SMTPCredential, error) {
	var dbCreds model.SMTPCredentials
	if err := db.Select(&dbCreds).Owner().Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve SMTP credentials: %w", err)
	}

	fileCreds := make([]*file.SMTPCredential, len(dbCreds))

	for i, dbCred := range dbCreds {
		fileCreds[i] = &file.SMTPCredential{
			EmailAddress:  dbCred.EmailAddress,
			ServerAddress: dbCred.ServerAddress.String(),
			Login:         dbCred.Login,
			Password:      dbCred.Password.String(),
		}

		logger.Infof("Exported SMTP credential %q", dbCred.EmailAddress)
	}

	return fileCreds, nil
}

func exportEmailTemplates(logger *log.Logger, db database.ReadAccess) ([]*file.EmailTemplate, error) {
	var dbTemplates model.EmailTemplates
	if err := db.Select(&dbTemplates).Run(); err != nil {
		return nil, fmt.Errorf("failed to retrieve email templates: %w", err)
	}

	fileTemplates := make([]*file.EmailTemplate, len(dbTemplates))

	for i, dbTemplate := range dbTemplates {
		fileTemplates[i] = &file.EmailTemplate{
			Name:        dbTemplate.Name,
			Subject:     dbTemplate.Subject,
			MIMEType:    dbTemplate.MIMEType,
			Body:        dbTemplate.Body,
			Attachments: dbTemplate.Attachments,
		}

		logger.Infof("Exported email template %q", dbTemplate.Name)
	}

	return fileTemplates, nil
}
