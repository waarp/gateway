package backup

import (
	"fmt"

	"code.waarp.fr/lib/log"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func importEmailConf(logger *log.Logger, db database.Access, config *file.EmailConfig,
	reset bool,
) error {
	if reset {
		if err := db.DeleteAll(&model.EmailTemplate{}).Run(); err != nil {
			return fmt.Errorf("failed to delete existing email templates: %w", err)
		}

		if err := db.DeleteAll(&model.SMTPCredential{}).Owner().Run(); err != nil {
			return fmt.Errorf("failed to delete existing SMTP credentials: %w", err)
		}
	}

	if config == nil {
		return nil
	}

	if err := importSMTPCredentials(logger, db, config.Credentials); err != nil {
		return err
	}

	return importEmailTemplates(logger, db, config.Templates)
}

func importSMTPCredentials(logger *log.Logger, db database.Access,
	credentials []*file.SMTPCredential,
) error {
	for _, fileCred := range credentials {
		var dbCred model.SMTPCredential
		if err := db.Get(&dbCred, "email_address=?", fileCred.EmailAddress).
			Owner().Run(); err != nil &&
			!database.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve SMTP credential %q: %w", fileCred.EmailAddress, err)
		}

		dbCred.EmailAddress = fileCred.EmailAddress
		dbCred.Login = fileCred.Login
		dbCred.Password = database.SecretText(fileCred.Password)

		if err := dbCred.ServerAddress.Set(fileCred.ServerAddress); err != nil {
			return fmt.Errorf("invalid SMTP server address: %w", err)
		}

		var dbErr error

		if dbCred.ID == 0 {
			logger.Infof("Insert new SMTP credential %q", dbCred.EmailAddress)
			dbErr = db.Insert(&dbCred).Run()
		} else {
			logger.Infof("Update existing SMTP credential %q", dbCred.EmailAddress)
			dbErr = db.Update(&dbCred).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import SMTP credential %q: %w", dbCred.EmailAddress, dbErr)
		}
	}

	return nil
}

func importEmailTemplates(logger *log.Logger, db database.Access, templates []*file.EmailTemplate) error {
	for _, fileTemplate := range templates {
		var dbTemplate model.EmailTemplate
		if err := db.Get(&dbTemplate, "name=?", fileTemplate.Name).Run(); err != nil &&
			!database.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve email template %q: %w", fileTemplate.Name, err)
		}

		dbTemplate.Name = fileTemplate.Name
		dbTemplate.Subject = fileTemplate.Subject
		dbTemplate.MIMEType = fileTemplate.MIMEType
		dbTemplate.Body = fileTemplate.Body
		dbTemplate.Attachments = fileTemplate.Attachments

		var dbErr error

		if dbTemplate.ID == 0 {
			logger.Infof("Insert new email template %q", dbTemplate.Name)
			dbErr = db.Insert(&dbTemplate).Run()
		} else {
			logger.Infof("Update existing email template %q", dbTemplate.Name)
			dbErr = db.Update(&dbTemplate).Run()
		}

		if dbErr != nil {
			return fmt.Errorf("failed to import email template %q: %w", dbTemplate.Name, dbErr)
		}
	}

	return nil
}
