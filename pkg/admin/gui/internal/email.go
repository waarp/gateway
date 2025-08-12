package internal

import (
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func GetEmailTemplate(db database.ReadAccess, name string) (*model.EmailTemplate, error) {
	var template model.EmailTemplate

	return &template, db.Get(&template, "name=?", name).Owner().Run()
}

func ListEmailTemplates(db database.ReadAccess) ([]*model.EmailTemplate, error) {
	var templates model.EmailTemplates

	return templates, db.Select(&templates).Owner().Run()
}

func GetSMTPCredential(db database.ReadAccess) (*model.SMTPCredential, error) {
	var credential model.SMTPCredential

	return &credential, db.Get(&credential, "name=?").Owner().Run()
}

func ListSMTPCredentials(db database.ReadAccess) ([]*model.SMTPCredential, error) {
	var credentials model.SMTPCredentials

	return credentials, db.Select(&credentials).Owner().Run()
}
