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

func TestEmailConfigExport(t *testing.T) {
	logger := testhelpers.GetTestLogger(t)
	db := dbtest.TestDatabase(t)

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
	require.NoError(t, dbtest.ChangeOwner("other_owner", db.Insert(dbCred2).Run))

	res, err := exportEmailConf(logger, db)
	require.NoError(t, err)

	t.Run("Templates exported", func(t *testing.T) {
		t.Parallel()

		require.Len(t, res.Templates, 2)
		assert.Equal(t, &file.EmailTemplate{
			Name:        dbTemplate1.Name,
			Subject:     dbTemplate1.Subject,
			MIMEType:    dbTemplate1.MIMEType,
			Body:        dbTemplate1.Body,
			Attachments: dbTemplate1.Attachments,
		}, res.Templates[0])
		assert.Equal(t, &file.EmailTemplate{
			Name:        dbTemplate2.Name,
			Subject:     dbTemplate2.Subject,
			MIMEType:    dbTemplate2.MIMEType,
			Body:        dbTemplate2.Body,
			Attachments: dbTemplate2.Attachments,
		}, res.Templates[1])
	})

	t.Run("Credentials exported", func(t *testing.T) {
		t.Parallel()

		require.Len(t, res.Credentials, 1)
		assert.Equal(t, &file.SMTPCredential{
			EmailAddress:  dbCred1.EmailAddress,
			ServerAddress: dbCred1.ServerAddress.String(),
			Login:         dbCred1.Login,
			Password:      dbCred1.Password.String(),
		}, res.Credentials[0])
	})
}
