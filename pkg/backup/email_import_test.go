package backup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/backup/file"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestNewEmailConfImport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	dbTemplate1 := &model.EmailTemplate{
		Name:        "template1",
		Subject:     "Subject 1",
		MIMEType:    "text/plain",
		Body:        "This is a test email template",
		Attachments: []string{"attachment1.txt", "attachment2.txt"},
	}
	dbTemplate2 := &model.EmailTemplate{
		Name:        "template2",
		Subject:     "Subject 2",
		MIMEType:    "text/html",
		Body:        "This is another test email template",
		Attachments: []string{"attachment3.txt"},
	}

	require.NoError(t, db.Insert(dbTemplate1).Run())
	require.NoError(t, db.Insert(dbTemplate2).Run())

	dbCred1 := &model.SMTPCredential{
		EmailAddress:  "foo@example.com",
		ServerAddress: types.Addr("smtp.example.com", 111),
		Login:         "foo",
		Password:      "bar",
	}
	dbCred2 := &model.SMTPCredential{
		EmailAddress:  "fizz@example.com",
		ServerAddress: types.Addr("smtp.example.com", 222),
		Login:         "fizz",
		Password:      "buzz",
	}

	require.NoError(t, db.Insert(dbCred1).Run())
	require.NoError(t, dbtest.ChangeOwner("waarp", db.Insert(dbCred2).Run))

	t.Run("No reset", func(t *testing.T) {
		const (
			newCredAddr     = "foobar@new.com"
			newTemplateName = "template3"
		)

		config := &file.EmailConfig{
			Credentials: []*file.SMTPCredential{{
				EmailAddress:  newCredAddr,
				ServerAddress: "smtp.new.com:123",
				Login:         "foobar",
				Password:      "sesame",
			}},
			Templates: []*file.EmailTemplate{
				{
					Name:        newTemplateName,
					Subject:     "New subject",
					MIMEType:    "text/plain",
					Body:        "This is a new test email template",
					Attachments: []string{"attachment4.txt", "attachment5.txt"},
				},
			},
		}

		require.NoError(t, importEmailConf(logger, db, config, false))

		t.Run("New credentials are imported", func(t *testing.T) {
			var check model.SMTPCredential
			require.NoError(t, db.Get(&check, "email_address=?", newCredAddr).Run())
			assert.Equal(t, check.Password.String(), config.Credentials[0].Password)
		})

		t.Run("New templates are imported", func(t *testing.T) {
			var check model.EmailTemplate
			require.NoError(t, db.Get(&check, "name=?", newTemplateName).Run())

			assert.Equal(t, config.Templates[0].Name, check.Name)
			assert.Equal(t, config.Templates[0].Subject, check.Subject)
			assert.Equal(t, config.Templates[0].MIMEType, check.MIMEType)
			assert.Equal(t, config.Templates[0].Body, check.Body)
			assert.Equal(t, config.Templates[0].Attachments, check.Attachments)
		})

		t.Run("Existing credentials are untouched", func(t *testing.T) {
			var check model.SMTPCredentials
			require.NoError(t, db.Select(&check).Where("email_address<>?", newCredAddr).
				OrderBy("id", true).Run())
			require.Len(t, check, 2)
			assert.Equal(t, check[0], dbCred1)
			assert.Equal(t, check[1], dbCred2)
		})

		t.Run("Existing credentials are untouched", func(t *testing.T) {
			var check model.EmailTemplates
			require.NoError(t, db.Select(&check).Where("name<>?", newTemplateName).
				OrderBy("id", true).Run())
			require.Len(t, check, 2)
			assert.Equal(t, check[0], dbTemplate1)
			assert.Equal(t, check[1], dbTemplate2)
		})
	})

	t.Run("With reset", func(t *testing.T) {
		const (
			newCredAddr     = "foobar@example.com"
			newTemplateName = "template3"
		)

		config := &file.EmailConfig{
			Credentials: []*file.SMTPCredential{
				{
					EmailAddress:  newCredAddr,
					ServerAddress: "smtp.example.com:123",
					Login:         "foobar",
					Password:      "sesame",
				},
			},
			Templates: []*file.EmailTemplate{
				{
					Name:        newTemplateName,
					Subject:     "New subject",
					MIMEType:    "text/plain",
					Body:        "This is a new test email template",
					Attachments: []string{"attachment4.txt", "attachment5.txt"},
				},
			},
		}

		require.NoError(t, importEmailConf(logger, db, config, true))

		t.Run("New credentials are imported", func(t *testing.T) {
			var check model.SMTPCredential
			require.NoError(t, db.Get(&check, "email_address=?", newCredAddr).Run())
			assert.Equal(t, check.Password.String(), config.Credentials[0].Password)
		})

		t.Run("New templates are imported", func(t *testing.T) {
			var check model.EmailTemplate
			require.NoError(t, db.Get(&check, "name=?", newTemplateName).Run())

			assert.Equal(t, config.Templates[0].Name, check.Name)
			assert.Equal(t, config.Templates[0].Subject, check.Subject)
			assert.Equal(t, config.Templates[0].MIMEType, check.MIMEType)
			assert.Equal(t, config.Templates[0].Body, check.Body)
			assert.Equal(t, config.Templates[0].Attachments, check.Attachments)
		})

		t.Run("Existing credentials are gone", func(t *testing.T) {
			var check model.SMTPCredentials
			require.NoError(t, db.Select(&check).Where("email_address<>?", newCredAddr).
				Owner().OrderBy("id", true).Run())
			assert.Len(t, check, 0)
		})

		t.Run("Existing credentials are gone", func(t *testing.T) {
			var check model.EmailTemplates
			require.NoError(t, db.Select(&check).Where("name<>?", newTemplateName).
				OrderBy("id", true).Run())
			assert.Len(t, check, 0)
		})
	})
}

func TestExistingEmailConfImport(t *testing.T) {
	db := dbtest.TestDatabase(t)
	logger := testhelpers.GetTestLogger(t)

	dbTemplate1 := &model.EmailTemplate{
		Name:        "template1",
		Subject:     "Subject 1",
		MIMEType:    "text/plain",
		Body:        "This is a test email template",
		Attachments: []string{"attachment1.txt", "attachment2.txt"},
	}
	dbTemplate2 := &model.EmailTemplate{
		Name:        "template2",
		Subject:     "Subject 2",
		MIMEType:    "text/html",
		Body:        "This is another test email template",
		Attachments: []string{"attachment3.txt"},
	}

	require.NoError(t, db.Insert(dbTemplate1).Run())
	require.NoError(t, db.Insert(dbTemplate2).Run())

	dbCred1 := &model.SMTPCredential{
		EmailAddress:  "foo@example.com",
		ServerAddress: types.Addr("smtp.example.com", 111),
		Login:         "foo",
		Password:      "bar",
	}
	dbCred2 := &model.SMTPCredential{
		EmailAddress:  "fizz@example.com",
		ServerAddress: types.Addr("smtp.example.com", 222),
		Login:         "fizz",
		Password:      "buzz",
	}

	require.NoError(t, db.Insert(dbCred1).Run())
	require.NoError(t, dbtest.ChangeOwner("waarp", db.Insert(dbCred2).Run))

	t.Run("No reset", func(t *testing.T) {
		var (
			newCredAddr     = dbCred1.EmailAddress
			newTemplateName = dbTemplate1.Name
		)

		config := &file.EmailConfig{
			Credentials: []*file.SMTPCredential{{
				EmailAddress:  newCredAddr,
				ServerAddress: "smtp.new.com:123",
				Login:         "foobar",
				Password:      "sesame",
			}},
			Templates: []*file.EmailTemplate{
				{
					Name:        newTemplateName,
					Subject:     "New subject",
					MIMEType:    "text/plain",
					Body:        "This is a new test email template",
					Attachments: []string{"attachment4.txt", "attachment5.txt"},
				},
			},
		}

		require.NoError(t, importEmailConf(logger, db, config, false))

		t.Run("Affected credentials are updated", func(t *testing.T) {
			var check model.SMTPCredential
			require.NoError(t, db.Get(&check, "email_address=?", newCredAddr).Run())
			assert.Equal(t, check.Password.String(), config.Credentials[0].Password)
		})

		t.Run("Affected templates are updated", func(t *testing.T) {
			var check model.EmailTemplate
			require.NoError(t, db.Get(&check, "name=?", newTemplateName).Run())

			assert.Equal(t, config.Templates[0].Name, check.Name)
			assert.Equal(t, config.Templates[0].Subject, check.Subject)
			assert.Equal(t, config.Templates[0].MIMEType, check.MIMEType)
			assert.Equal(t, config.Templates[0].Body, check.Body)
			assert.Equal(t, config.Templates[0].Attachments, check.Attachments)
		})

		t.Run("Other credentials are untouched", func(t *testing.T) {
			var check model.SMTPCredentials
			require.NoError(t, db.Select(&check).Where("email_address<>?", newCredAddr).
				OrderBy("id", true).Run())
			require.Len(t, check, 1)
			assert.Equal(t, check[0], dbCred2)
		})

		t.Run("Other credentials are untouched", func(t *testing.T) {
			var check model.EmailTemplates
			require.NoError(t, db.Select(&check).Where("name<>?", newTemplateName).
				OrderBy("id", true).Run())
			require.Len(t, check, 1)
			assert.Equal(t, check[0], dbTemplate2)
		})
	})
}
