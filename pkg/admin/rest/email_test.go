package rest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

// Email templates
const (
	EmailTemplatesPath  = "/email/templates"
	EmailTemplatePath   = "/email/templates/{email_template}"
	SMTPCredentialsPath = "/email/credentials"
	SMTPCredentialPath  = "/email/credentials/{smtp_credential}"
)

func TestAddEmailTemplate(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := addEmailTemplate(logger, db)

	dbTemplate1 := &model.EmailTemplate{
		Name:        "template1",
		Subject:     "Email subject",
		Body:        "This is an email template",
		Attachments: nil,
	}
	require.NoError(t, db.Insert(dbTemplate1).Run())

	t.Run("Valid request", func(t *testing.T) {
		const (
			templateName     = "template2"
			templateSubject  = "Email subject 2"
			templateMimeType = "text/html"
			templateBody     = "This is another email template"

			expectedLocation = EmailTemplatesPath + "/" + templateName
		)
		templateAttachements := []string{"attachment1.txt", "attachment2.txt"}

		reqBody := bytes.Buffer{}
		require.NoError(t, json.NewEncoder(&reqBody).Encode(map[string]any{
			"name":        templateName,
			"subject":     templateSubject,
			"mimeType":    templateMimeType,
			"body":        templateBody,
			"attachments": templateAttachements,
		}))

		req := httptest.NewRequest(http.MethodPost, EmailTemplatesPath, &reqBody)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusCreated, w.Code, `201 response code`)
			assert.Equal(t, expectedLocation, w.Header().Get("Location"), `Correct Location header`)
			assert.Empty(t, w.Body.String(), `Empty response body`)
		})

		t.Run("Check database", func(t *testing.T) {
			var check model.EmailTemplate
			require.NoError(t, db.Get(&check, "name=?", templateName).Run())

			assert.Equal(t, templateName, check.Name)
			assert.Equal(t, templateSubject, check.Subject)
			assert.Equal(t, templateMimeType, check.MIMEType)
			assert.Equal(t, templateBody, check.Body)
			assert.Equal(t, templateAttachements, check.Attachments)
		})
	})
}

func TestListEmailTemplate(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := listEmailTemplates(logger, db)

	dbTemplate1 := &model.EmailTemplate{
		Name:        "template1",
		Subject:     "Email subject 1",
		MIMEType:    "text/plain",
		Body:        "This is an email template",
		Attachments: []string{"attachment1.txt", "attachment2.txt"},
	}
	require.NoError(t, db.Insert(dbTemplate1).Run())

	dbTemplate2 := &model.EmailTemplate{
		Name:     "template2",
		Subject:  "Email subject 2",
		MIMEType: "text/html",
		Body:     "This is another email template",
	}
	require.NoError(t, db.Insert(dbTemplate2).Run())

	t.Run("Valid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, EmailTemplatePath, nil)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code, `200 response code`)
			assert.JSONEq(t, marshal(t, map[string]any{
				"emailTemplates": []any{
					map[string]any{
						"name":        dbTemplate1.Name,
						"subject":     dbTemplate1.Subject,
						"mimeType":    dbTemplate1.MIMEType,
						"body":        dbTemplate1.Body,
						"attachments": dbTemplate1.Attachments,
					},
					map[string]any{
						"name":        dbTemplate2.Name,
						"subject":     dbTemplate2.Subject,
						"mimeType":    dbTemplate2.MIMEType,
						"body":        dbTemplate2.Body,
						"attachments": dbTemplate2.Attachments,
					},
				},
			}), w.Body.String(), `Correct response body`)
		})
	})
}

func TestGetEmailTemplate(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := getEmailTemplate(logger, db)

	dbTemplate1 := &model.EmailTemplate{
		Name:        "template1",
		Subject:     "Email subject 1",
		MIMEType:    "text/plain",
		Body:        "This is an email template",
		Attachments: []string{"attachment1.txt", "attachment2.txt"},
	}
	require.NoError(t, db.Insert(dbTemplate1).Run())

	dbTemplate2 := &model.EmailTemplate{
		Name:     "template2",
		Subject:  "Email subject 2",
		MIMEType: "text/html",
		Body:     "This is another email template",
	}
	require.NoError(t, db.Insert(dbTemplate2).Run())

	t.Run("Valid request", func(t *testing.T) {
		req := makeRequest(http.MethodGet, nil, EmailTemplatePath, dbTemplate1.Name)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code, `200 response code`)
			assert.JSONEq(t, marshal(t, map[string]any{
				"name":        dbTemplate1.Name,
				"subject":     dbTemplate1.Subject,
				"mimeType":    dbTemplate1.MIMEType,
				"body":        dbTemplate1.Body,
				"attachments": dbTemplate1.Attachments,
			}), w.Body.String(), `Correct response body`)
		})
	})
}

func TestUpdateEmailTemplate(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := updateEmailTemplate(logger, db)

	dbTemplate1 := &model.EmailTemplate{
		Name:        "template1",
		Subject:     "Email subject 1",
		MIMEType:    "text/plain",
		Body:        "This is an email template",
		Attachments: []string{"attachment1.txt", "attachment2.txt"},
	}
	require.NoError(t, db.Insert(dbTemplate1).Run())

	dbTemplate2 := &model.EmailTemplate{
		Name:     "template2",
		Subject:  "Email subject 2",
		MIMEType: "text/html",
		Body:     "This is another email template",
	}
	require.NoError(t, db.Insert(dbTemplate2).Run())

	t.Run("Valid request", func(t *testing.T) {
		const (
			templateName     = "template2_new"
			templateSubject  = "Email subject 2 new"
			templateMimeType = "text/json"
			templateBody     = "This is yet another email template"

			expectedLocation = EmailTemplatesPath + "/" + templateName
		)
		templateAttachments := []string{"attachment3.txt", "attachment4.txt"}

		reqBody := bytes.Buffer{}
		require.NoError(t, json.NewEncoder(&reqBody).Encode(map[string]any{
			"name":        templateName,
			"subject":     templateSubject,
			"mimeType":    templateMimeType,
			"body":        templateBody,
			"attachments": templateAttachments,
		}))

		req := makeRequest(http.MethodPatch, &reqBody, EmailTemplatePath, dbTemplate2.Name)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusCreated, w.Code, `201 response code`)
			assert.Equal(t, expectedLocation, w.Header().Get("Location"), `Correct Location header`)
			assert.Empty(t, w.Body.String(), `Empty response body`)
		})

		t.Run("Check database", func(t *testing.T) {
			var (
				check model.EmailTemplate
				nfe   *database.NotFoundError
			)

			assert.ErrorAs(t, db.Get(&check, "name=?", dbTemplate2.Name).Run(), &nfe)
			require.NoError(t, db.Get(&check, "name=?", templateName).Run())

			assert.Equal(t, templateName, check.Name)
			assert.Equal(t, templateSubject, check.Subject)
			assert.Equal(t, templateMimeType, check.MIMEType)
			assert.Equal(t, templateBody, check.Body)
			assert.Equal(t, templateAttachments, check.Attachments)
		})
	})
}

func TestDeleteEmailTemplate(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := deleteEmailTemplate(logger, db)

	dbTemplate1 := &model.EmailTemplate{
		Name:        "template1",
		Subject:     "Email subject 1",
		MIMEType:    "text/plain",
		Body:        "This is an email template",
		Attachments: []string{"attachment1.txt", "attachment2.txt"},
	}
	require.NoError(t, db.Insert(dbTemplate1).Run())

	dbTemplate2 := &model.EmailTemplate{
		Name:     "template2",
		Subject:  "Email subject 2",
		MIMEType: "text/html",
		Body:     "This is another email template",
	}
	require.NoError(t, db.Insert(dbTemplate2).Run())

	t.Run("Valid request", func(t *testing.T) {
		req := makeRequest(http.MethodDelete, nil, EmailTemplatePath, dbTemplate1.Name)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusNoContent, w.Code, `204 response code`)
			assert.Empty(t, w.Body.String(), `Empty response body`)
		})

		t.Run("Check database", func(t *testing.T) {
			var (
				check model.EmailTemplate
				nfe   *database.NotFoundError
			)

			assert.ErrorAs(t, db.Get(&check, "name=?", dbTemplate1.Name).Run(), &nfe)
			assert.NoError(t, db.Get(&check, "name=?", dbTemplate2.Name).Run())
		})
	})
}

// SMTP Credentials

func TestAddSMTPCredential(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := addSMTPCredential(logger, db)

	dbCredential1 := &model.SMTPCredential{
		EmailAddress:  "foo@example.com",
		ServerAddress: types.Addr("smtp.example.com", 123),
		Login:         "foo",
		Password:      "bar",
	}
	require.NoError(t, db.Insert(dbCredential1).Run())

	t.Run("Valid request", func(t *testing.T) {
		const (
			credSender   = "example@other.com"
			credServer   = "smtp.other.com:456"
			credLogin    = "fizz"
			credPassword = "buzz"

			expectedLocation = SMTPCredentialsPath + "/" + credSender
		)

		reqBody := bytes.Buffer{}
		require.NoError(t, json.NewEncoder(&reqBody).Encode(map[string]any{
			"emailAddress":  credSender,
			"serverAddress": credServer,
			"login":         credLogin,
			"password":      credPassword,
		}))

		req := httptest.NewRequest(http.MethodPost, SMTPCredentialsPath, &reqBody)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusCreated, w.Code, `201 response code`)
			assert.Equal(t, expectedLocation, w.Header().Get("Location"), `Correct Location header`)
			assert.Empty(t, w.Body.String(), `Empty response body`)
		})

		t.Run("Check database", func(t *testing.T) {
			var check model.SMTPCredential
			require.NoError(t, db.Get(&check, "email_address=?", credSender).Run())

			assert.Equal(t, credSender, check.EmailAddress)
			assert.Equal(t, credServer, check.ServerAddress.String())
			assert.Equal(t, credLogin, check.Login)
			assert.Equal(t, credPassword, check.Password.String())
		})
	})
}

func TestListSMTPCredentials(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := listSMPTCredentials(logger, db)

	dbCredential1 := &model.SMTPCredential{
		EmailAddress:  "1_foo@example1.com",
		ServerAddress: types.Addr("smtp.example1.com", 111),
		Login:         "foo",
		Password:      "bar",
	}
	require.NoError(t, db.Insert(dbCredential1).Run())

	dbCredential2 := &model.SMTPCredential{
		EmailAddress:  "2_fizz@example2.com",
		ServerAddress: types.Addr("smtp.example2.com", 222),
		Login:         "fizz",
		Password:      "buzz",
	}
	require.NoError(t, db.Insert(dbCredential2).Run())

	t.Run("Valid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, EmailTemplatePath, nil)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code, `200 response code`)
			assert.JSONEq(t, marshal(t, map[string]any{
				"smtpCredentials": []any{
					map[string]any{
						"emailAddress":  dbCredential1.EmailAddress,
						"serverAddress": dbCredential1.ServerAddress,
						"login":         dbCredential1.Login,
						"password":      dbCredential1.Password,
					},
					map[string]any{
						"emailAddress":  dbCredential2.EmailAddress,
						"serverAddress": dbCredential2.ServerAddress,
						"login":         dbCredential2.Login,
						"password":      dbCredential2.Password,
					},
				},
			}), w.Body.String(), `Correct response body`)
		})
	})
}

func TestGetSMTPCredential(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := getSMTPCredential(logger, db)

	dbCredential1 := &model.SMTPCredential{
		EmailAddress:  "foo@example1.com",
		ServerAddress: types.Addr("smtp.example1.com", 111),
		Login:         "foo",
		Password:      "bar",
	}
	require.NoError(t, db.Insert(dbCredential1).Run())

	dbCredential2 := &model.SMTPCredential{
		EmailAddress:  "fizz@example2.com",
		ServerAddress: types.Addr("smtp.example2.com", 222),
		Login:         "fizz",
		Password:      "buzz",
	}
	require.NoError(t, db.Insert(dbCredential2).Run())

	t.Run("Valid request", func(t *testing.T) {
		req := makeRequest(http.MethodPost, nil, SMTPCredentialPath, dbCredential1.EmailAddress)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusOK, w.Code, `200 response code`)
			assert.JSONEq(t, marshal(t, map[string]any{
				"emailAddress":  dbCredential1.EmailAddress,
				"serverAddress": dbCredential1.ServerAddress,
				"login":         dbCredential1.Login,
				"password":      dbCredential1.Password,
			}), w.Body.String(), `Correct response body`)
		})
	})
}

func TestUpdateSMTPCredential(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := updateSMTPCredential(logger, db)

	dbCredential1 := &model.SMTPCredential{
		EmailAddress:  "foo@example1.com",
		ServerAddress: types.Addr("smtp.example1.com", 111),
		Login:         "foo",
		Password:      "bar",
	}
	require.NoError(t, db.Insert(dbCredential1).Run())

	dbCredential2 := &model.SMTPCredential{
		EmailAddress:  "fizz@example2.com",
		ServerAddress: types.Addr("smtp.example2.com", 222),
		Login:         "fizz",
		Password:      "buzz",
	}
	require.NoError(t, db.Insert(dbCredential2).Run())

	t.Run("Valid request", func(t *testing.T) {
		const (
			credSender   = "example@new.com"
			credServer   = "smtp.new.com:123"
			credLogin    = "toto"
			credPassword = "titi"

			expectedLocation = SMTPCredentialsPath + "/" + credSender
		)

		reqBody := bytes.Buffer{}
		require.NoError(t, json.NewEncoder(&reqBody).Encode(map[string]any{
			"emailAddress":  credSender,
			"serverAddress": credServer,
			"login":         credLogin,
			"password":      credPassword,
		}))

		req := makeRequest(http.MethodPost, &reqBody, SMTPCredentialPath, dbCredential2.EmailAddress)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusCreated, w.Code, `201 response code`)
			assert.Equal(t, expectedLocation, w.Header().Get("Location"), `Correct Location header`)
			assert.Empty(t, w.Body.String(), `Empty response body`)
		})

		t.Run("Check database", func(t *testing.T) {
			var (
				check model.SMTPCredential
				nfe   *database.NotFoundError
			)

			assert.ErrorAs(t, db.Get(&check, "email_address=?", dbCredential2.EmailAddress).Run(), &nfe)
			require.NoError(t, db.Get(&check, "email_address=?", credSender).Run())

			assert.Equal(t, credSender, check.EmailAddress)
			assert.Equal(t, credServer, check.ServerAddress.String())
			assert.Equal(t, credLogin, check.Login)
			assert.Equal(t, credPassword, check.Password.String())
		})
	})
}

func TestDeleteSMTPCredential(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)
	handle := deleteSMTPCredential(logger, db)

	dbCredential1 := &model.SMTPCredential{
		EmailAddress:  "foo@example1.com",
		ServerAddress: types.Addr("smtp.example1.com", 111),
		Login:         "foo",
		Password:      "bar",
	}
	require.NoError(t, db.Insert(dbCredential1).Run())

	dbCredential2 := &model.SMTPCredential{
		EmailAddress:  "fizz@example2.com",
		ServerAddress: types.Addr("smtp.example2.com", 222),
		Login:         "fizz",
		Password:      "buzz",
	}
	require.NoError(t, db.Insert(dbCredential2).Run())

	t.Run("Valid request", func(t *testing.T) {
		req := makeRequest(http.MethodDelete, nil, SMTPCredentialPath, dbCredential1.EmailAddress)
		w := httptest.NewRecorder()
		handle.ServeHTTP(w, req)

		t.Run("Check response", func(t *testing.T) {
			assert.Equal(t, http.StatusNoContent, w.Code, `204 response code`)
			assert.Empty(t, w.Body.String(), `Empty response body`)
		})

		t.Run("Check database", func(t *testing.T) {
			var (
				check model.SMTPCredential
				nfe   *database.NotFoundError
			)

			assert.ErrorAs(t, db.Get(&check, "email_address=?", dbCredential1.EmailAddress).Run(), &nfe)
			assert.NoError(t, db.Get(&check, "email_address=?", dbCredential2.EmailAddress).Run())
		})
	})
}
