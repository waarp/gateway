package wg

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// #####################################################################################################################
// ################################################## EMAIL TEMPLATES ##################################################
// #####################################################################################################################

func TestEmailTemplateAdd(t *testing.T) {
	const (
		templateName        = "template"
		templateSubject     = "Email subject"
		templateMimeType    = "text/html"
		templateBody        = "This is an email template"
		templateAttachment1 = "attachment1.txt"
		templateAttachment2 = "attachment2.txt"

		path     = "/api/email/templates"
		location = path + "/" + templateName
	)

	expectedBody := map[string]any{
		"name":        templateName,
		"subject":     templateSubject,
		"mimeType":    templateMimeType,
		"body":        templateBody,
		"attachments": []any{templateAttachment1, templateAttachment2},
	}

	t.Run(`Given the email template "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &EmailTemplateAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body:   expectedBody,
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", templateName,
					"--subject", templateSubject,
					"--mime-type", templateMimeType,
					"--body", templateBody,
					"--attachments", templateAttachment1,
					"--attachments", templateAttachment2,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The email template %q was successfully added.\n", templateName),
					w.String(),
					"Then it should display a message saying the template was added",
				)
			})
		})
	})
}

func TestEmailTemplateList(t *testing.T) {
	const (
		path = "/api/email/templates"

		sort   = "name+"
		limit  = "10"
		offset = "5"

		template1name    = "template1"
		template1Subject = "Email subject 1"
		template1Mime    = "text/html"
		template1text    = "This is an email template"

		template2name   = "template2"
		templateSubject = "Email subject 2"
		template2Mime   = "text/plain"
		template2text   = "This is another email template"
	)

	t.Run(`Given the email template "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &EmailTemplateList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			values: map[string][]string{
				"limit":  {limit},
				"offset": {offset},
				"sort":   {sort},
			},
			path: path,
		}

		templates := []map[string]any{{
			"name":     template1name,
			"subject":  template1Subject,
			"mimeType": template1Mime,
			"body":     template1text,
		}, {
			"name":     template2name,
			"subject":  templateSubject,
			"mimeType": template2Mime,
			"body":     template2text,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"emailTemplates": templates},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort,
				),
					"Then it should not return an error",
				)

				outputData := slices.Clone(templates)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`=== Email templates ===`,
						`{{- range . }}`,
						`-Email template "{{.name}}"`,
						`  -Subject: {{.subject}}`,
						`  -MIME type: {{.mimeType}}`,
						`  -Body: {{.body}}`,
						`  -Attachments: <none>`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the templates",
				)
			})
		})
	})
}

func TestEmailTemplateGet(t *testing.T) {
	const (
		templateName     = "template"
		templateSubject  = "Email subject"
		templateMimeType = "text/plain"
		templateBody     = `This is an email template
with multiple lines`
		templateAttachment1 = "attachment1.txt"
		templateAttachment2 = "attachment2.txt"

		path = "/api/email/templates/" + templateName
	)

	responseBody := map[string]any{
		"name":        templateName,
		"subject":     templateSubject,
		"mimeType":    templateMimeType,
		"body":        templateBody,
		"attachments": []string{templateAttachment1, templateAttachment2},
	}

	t.Run(`Given the email template "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &EmailTemplateGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   responseBody,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, templateName),
					"Then it should not return an error")

				assert.Equal(t,
					expectedOutput(t, result.body,
						`-Email template "{{.name}}"`,
						`  -Subject: {{.subject}}`,
						`  -MIME type: {{.mimeType}}`,
						`  -Body: {{- lines .body 5}}`,
						`  -Attachments: {{join .attachments}}`,
					),
					w.String(),
					"Then it should display the email template",
				)
			})
		})
	})
}

func TestEmailTemplateUpdate(t *testing.T) {
	const (
		oldTemplateName        = "old_template"
		newTemplateName        = "new_template"
		newTemplateSubject     = "New email subject"
		newTemplateMime        = "text/plain"
		newTemplateBody        = "This is an email template"
		newTemplateAttachment1 = "attachment1.txt"
		newTemplateAttachment2 = "attachment2.txt"

		path     = "/api/email/templates/" + oldTemplateName
		location = "/api/email/templates/" + newTemplateName
	)

	expectedBody := map[string]any{
		"name":        newTemplateName,
		"subject":     newTemplateSubject,
		"mimeType":    newTemplateMime,
		"body":        newTemplateBody,
		"attachments": []any{newTemplateAttachment1, newTemplateAttachment2},
	}

	t.Run(`Given the email template "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &EmailTemplateUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body:   expectedBody,
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, oldTemplateName,
					"--name", newTemplateName,
					"--subject", newTemplateSubject,
					"--mime-type", newTemplateMime,
					"--body", newTemplateBody,
					"--attachments", newTemplateAttachment1,
					"--attachments", newTemplateAttachment2,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The email template %q was successfully updated.\n", newTemplateName),
					w.String(),
					"Then it should display a message saying the template was updated",
				)
			})
		})
	})
}

func TestEmailTemplateDelete(t *testing.T) {
	const (
		templateName = "template"
		path         = "/api/email/templates/" + templateName
	)

	t.Run(`Given the email template "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &EmailTemplateDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, templateName),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The email template %q was successfully deleted.\n", templateName),
					w.String(),
					"Then it should display a message saying the template was deleted",
				)
			})
		})
	})
}

// ####################################################################################################################
// ################################################# SMTP CREDENTIALS #################################################
// ####################################################################################################################

func TestSMTPCredentialAdd(t *testing.T) {
	const (
		smtpEmail    = "foobar@example.com"
		smtpServer   = "smtp.example.com:123"
		smtpLogin    = "foobar"
		smtpPassword = "sesame"

		path     = "/api/email/credentials"
		location = path + "/" + smtpEmail
	)

	expectedBody := map[string]any{
		"emailAddress":  smtpEmail,
		"serverAddress": smtpServer,
		"login":         smtpLogin,
		"password":      smtpPassword,
	}

	t.Run(`Given the SMTP credential "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SMTPCredentialAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body:   expectedBody,
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--email-address", smtpEmail,
					"--server-address", smtpServer,
					"--login", smtpLogin,
					"--password", smtpPassword,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The SMTP credential %q was successfully added.\n", smtpEmail),
					w.String(),
					"Then it should display a message saying the credential was added",
				)
			})
		})
	})
}

func TestSMTPCredentialList(t *testing.T) {
	const (
		path = "/api/email/credentials"

		sort   = "email+"
		limit  = "10"
		offset = "5"

		smtpEmail1    = "foo@example.com"
		smtpServer1   = "smtp.example.com:123"
		smtpLogin1    = "foo"
		smtpPassword1 = "sesame"

		smtpEmail2    = "bar@example.com"
		smtpServer2   = "smtp.example.com:456"
		smtpLogin2    = "bar"
		smtpPassword2 = "sesame"
	)

	t.Run(`Given the SMTP credential "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SMTPCredentialList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			values: map[string][]string{
				"limit":  {limit},
				"offset": {offset},
				"sort":   {sort},
			},
			path: path,
		}

		credentials := []map[string]any{{
			"emailAddress":  smtpEmail1,
			"serverAddress": smtpServer1,
			"login":         smtpLogin1,
			"password":      smtpPassword1,
		}, {
			"emailAddress":  smtpEmail2,
			"serverAddress": smtpServer2,
			"login":         smtpLogin2,
			"password":      smtpPassword2,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"smtpCredentials": credentials},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					expectedOutput(t, credentials,
						`=== SMTP credentials ===`,
						`{{- range . }}`,
						`-SMTP credential "{{.emailAddress}}"`,
						`  -Server address: {{.serverAddress}}`,
						`  -Login: {{.login}}`,
						`  -Password: {{.password}}`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the SMTP credentials",
				)
			})
		})
	})
}

func TestSMTPCredentialGet(t *testing.T) {
	const (
		smtpEmail    = "foobar"
		smtpServer   = "smtp.example.com:123"
		smtpLogin    = "foobar"
		smtpPassword = "sesame"

		path = "/api/email/credentials/" + smtpEmail
	)

	responseBody := map[string]any{
		"emailAddress":  smtpEmail,
		"serverAddress": smtpServer,
		"login":         smtpLogin,
		"password":      smtpPassword,
	}

	t.Run(`Given the SMTP credential "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SMTPCredentialGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   responseBody,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, smtpEmail),
					"Then it should not return an error")

				assert.Equal(t,
					expectedOutput(t, result.body,
						`-SMTP credential "{{.emailAddress}}"`,
						`  -Server address: {{.serverAddress}}`,
						`  -Login: {{.login}}`,
						`  -Password: {{.password}}`,
					),
					w.String(),
					"Then it should display the SMTP credential",
				)
			})
		})
	})
}

func TestSMTPCredentialUpdate(t *testing.T) {
	const (
		oldEmail    = "old@example.com"
		newEmail    = "new@example.com"
		newServer   = "smtp.example.com:123"
		newLogin    = "foobar"
		newPassword = "sesame"

		path     = "/api/email/credentials/" + oldEmail
		location = "/api/email/credentials/" + newEmail
	)

	expectedBody := map[string]any{
		"emailAddress":  newEmail,
		"serverAddress": newServer,
		"login":         newLogin,
		"password":      newPassword,
	}

	t.Run(`Given the SMTP credential "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SMTPCredentialUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body:   expectedBody,
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, oldEmail,
					"--email-address", newEmail,
					"--server-address", newServer,
					"--login", newLogin,
					"--password", newPassword,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The SMTP credential %q was successfully updated.\n", newEmail),
					w.String(),
					"Then it should display a message saying the credential was updated",
				)
			})
		})
	})
}

func TestSMTPCredentialDelete(t *testing.T) {
	const (
		email = "foobar"
		path  = "/api/email/credentials/" + email
	)

	t.Run(`Given the SMTP credential "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &SMTPCredentialDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusNoContent,
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, email),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The SMTP credential %q was successfully deleted.\n", email),
					w.String(),
					"Then it should display a message saying the credential was deleted",
				)
			})
		})
	})
}
