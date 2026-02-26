package rest

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/logging/log"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

// Email templates

func retrieveEmailTemplate(r *http.Request, db *database.DB) (*model.EmailTemplate, error) {
	name, ok := mux.Vars(r)["email_template"]
	if !ok {
		return nil, notFound("missing email template name")
	}

	var template model.EmailTemplate
	if err := db.Get(&template, "name=?", name).Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("email template %q not found", name)
		}

		return nil, fmt.Errorf("failed to retrieve email template %q: %w", name, err)
	}

	return &template, nil
}

func addEmailTemplate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restTemplate api.PostEmailTemplateObject
		if err := readJSON(r, &restTemplate); handleError(w, logger, err) {
			return
		}

		dbTemplate := model.EmailTemplate{
			Name:        restTemplate.Name,
			Subject:     restTemplate.Subject,
			MIMEType:    restTemplate.MIMEType,
			Body:        restTemplate.Body,
			Attachments: restTemplate.Attachments,
		}
		if err := db.Insert(&dbTemplate).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, restTemplate.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func listEmailTemplates(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"name", true},
		"name+":   order{"name", true},
		"name-":   order{"name", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbTemplates model.EmailTemplates

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbTemplates)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restTemplates := make([]*api.GetEmailTemplateObject, len(dbTemplates))
		for i, dbTemplate := range dbTemplates {
			restTemplates[i] = &api.GetEmailTemplateObject{
				Name:        dbTemplate.Name,
				Subject:     dbTemplate.Subject,
				MIMEType:    dbTemplate.MIMEType,
				Body:        dbTemplate.Body,
				Attachments: dbTemplate.Attachments,
			}
		}

		response := map[string][]*api.GetEmailTemplateObject{"emailTemplates": restTemplates}
		handleError(w, logger, writeJSON(w, response))
	}
}

func getEmailTemplate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbTemplate, getErr := retrieveEmailTemplate(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restTemplate := &api.GetEmailTemplateObject{
			Name:        dbTemplate.Name,
			Subject:     dbTemplate.Subject,
			MIMEType:    dbTemplate.MIMEType,
			Body:        dbTemplate.Body,
			Attachments: dbTemplate.Attachments,
		}
		handleError(w, logger, writeJSON(w, restTemplate))
	}
}

func updateEmailTemplate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldDBTemplate, getErr := retrieveEmailTemplate(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restTemplate api.PatchEmailTemplateObject
		if err := readJSON(r, &restTemplate); handleError(w, logger, err) {
			return
		}

		dbTemplate := model.EmailTemplate{ID: oldDBTemplate.ID}
		setIfValid(&dbTemplate.Name, restTemplate.Name)
		setIfValid(&dbTemplate.Subject, restTemplate.Subject)
		setIfValid(&dbTemplate.MIMEType, restTemplate.MIMEType)
		setIfValid(&dbTemplate.Body, restTemplate.Body)
		setIfValidList(&dbTemplate.Attachments, restTemplate.Attachments)

		if err := db.Update(&dbTemplate).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbTemplate.Name))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteEmailTemplate(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		template, getErr := retrieveEmailTemplate(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(template).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// SMTP credentials

func retrieveSMTPCredential(r *http.Request, db *database.DB) (*model.SMTPCredential, error) {
	name, ok := mux.Vars(r)["smtp_credential"]
	if !ok {
		return nil, notFound("missing SMTP credential sender address")
	}

	var credential model.SMTPCredential
	if err := db.Get(&credential, "email_address=?", name).Owner().Run(); err != nil {
		if database.IsNotFound(err) {
			return nil, notFoundf("SMTP credential %q not found", name)
		}

		return nil, fmt.Errorf("failed to retrieve SMTP credential %q: %w", name, err)
	}

	return &credential, nil
}

func addSMTPCredential(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var restCredential api.PostSMTPCredentialObject
		if err := readJSON(r, &restCredential); handleError(w, logger, err) {
			return
		}

		dbCredential := model.SMTPCredential{
			EmailAddress: restCredential.EmailAddress,
			Login:        restCredential.Login,
			Password:     database.SecretText(restCredential.Password),
		}
		if err := dbCredential.ServerAddress.Set(restCredential.ServerAddress); handleError(w, logger, err) {
			return
		}

		if err := db.Insert(&dbCredential).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", location(r.URL, restCredential.EmailAddress))
		w.WriteHeader(http.StatusCreated)
	}
}

func listSMPTCredentials(logger *log.Logger, db *database.DB) http.HandlerFunc {
	validSorting := orders{
		"default": order{"email_address", true},
		"email+":  order{"email_address", true},
		"email-":  order{"email_address", false},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var dbCredentials model.SMTPCredentials

		query, queryErr := parseSelectQuery(r, db, validSorting, &dbCredentials)
		if handleError(w, logger, queryErr) {
			return
		}

		if err := query.Run(); handleError(w, logger, err) {
			return
		}

		restCredentials := make([]*api.GetSMTPCredentialObject, len(dbCredentials))
		for i, dbCredential := range dbCredentials {
			restCredentials[i] = &api.GetSMTPCredentialObject{
				EmailAddress:  dbCredential.EmailAddress,
				ServerAddress: dbCredential.ServerAddress.String(),
				Login:         dbCredential.Login,
				Password:      dbCredential.Password.String(),
			}
		}

		response := map[string][]*api.GetSMTPCredentialObject{"smtpCredentials": restCredentials}
		handleError(w, logger, writeJSON(w, response))
	}
}

func getSMTPCredential(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbCredential, getErr := retrieveSMTPCredential(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		restCredential := &api.GetSMTPCredentialObject{
			EmailAddress:  dbCredential.EmailAddress,
			ServerAddress: dbCredential.ServerAddress.String(),
			Login:         dbCredential.Login,
			Password:      dbCredential.Password.String(),
		}
		handleError(w, logger, writeJSON(w, restCredential))
	}
}

func updateSMTPCredential(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		oldDBCredential, getErr := retrieveSMTPCredential(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		var restCredential api.PatchSMTPCredentialObject
		if err := readJSON(r, &restCredential); handleError(w, logger, err) {
			return
		}

		dbCredential := model.SMTPCredential{ID: oldDBCredential.ID}
		setIfValid(&dbCredential.EmailAddress, restCredential.EmailAddress)
		setIfValid(&dbCredential.Login, restCredential.Login)
		setIfValidSecret(&dbCredential.Password, restCredential.Password)

		if restCredential.ServerAddress.Valid {
			if err := dbCredential.ServerAddress.Set(restCredential.ServerAddress.
				Value); handleError(w, logger, err) {
				return
			}
		}

		if err := db.Update(&dbCredential).Run(); handleError(w, logger, err) {
			return
		}

		w.Header().Set("Location", locationUpdate(r.URL, dbCredential.EmailAddress))
		w.WriteHeader(http.StatusCreated)
	}
}

func deleteSMTPCredential(logger *log.Logger, db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		credential, getErr := retrieveSMTPCredential(r, db)
		if handleError(w, logger, getErr) {
			return
		}

		if err := db.Delete(credential).Run(); handleError(w, logger, err) {
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
