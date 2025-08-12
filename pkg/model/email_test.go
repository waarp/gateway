package model

import (
	"testing"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
)

func TestEmailTemplateBeforeWrite(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	existing := &EmailTemplate{
		Name:    "existing",
		Subject: "existing subject",
		Body:    "existing body",
	}
	require.NoError(t, db.Insert(existing).Run())

	for _, test := range []struct {
		name     string
		template *EmailTemplate
		wantErr  error
	}{
		{
			name:     "Valid template",
			template: &EmailTemplate{Name: "new", Subject: "new subject", Body: "new body"},
			wantErr:  nil,
		}, {
			name:     "No name",
			template: &EmailTemplate{Subject: "subject", Body: "body"},
			wantErr:  ErrEmailTemplateNoName,
		}, {
			name:     "No subject",
			template: &EmailTemplate{Name: "test", Body: "body"},
			wantErr:  ErrEmailTemplateNoSubj,
		}, {
			name:     "No body",
			template: &EmailTemplate{Name: "test", Subject: "subject"},
			wantErr:  ErrEmailTemplateNoBody,
		}, {
			name:     "Invalid MIME type",
			template: &EmailTemplate{Name: "test", Subject: "subject", Body: "body", MIMEType: "invalid/"},
			wantErr:  ErrEmailTemplateInvalidMIME,
		}, {
			name:     "Duplicate name",
			template: &EmailTemplate{Name: "existing", Subject: "subject", Body: "body"},
			wantErr:  ErrEmailTemplateDuplicate,
		}, {
			name:     "Default MIME type",
			template: &EmailTemplate{Name: "test2", Subject: "subject", Body: "body"},
			wantErr:  nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if test.wantErr == nil {
				require.NoError(t, test.template.BeforeWrite(db))
			} else {
				require.ErrorIs(t, test.template.BeforeWrite(db), test.wantErr)
			}
		})
	}
}

func TestSMTPCredentialBeforeWrite(t *testing.T) {
	t.Parallel()

	db := dbtest.TestDatabase(t)
	existing := &SMTPCredential{
		EmailAddress:  "existing@example.com",
		ServerAddress: types.Addr("smtp.example.com", 123),
		Login:         "foobar",
		Password:      "sesame",
	}
	require.NoError(t, db.Insert(existing).Run())

	for _, test := range []struct {
		name    string
		cred    *SMTPCredential
		wantErr error
	}{
		{
			name: "Valid credential",
			cred: &SMTPCredential{
				EmailAddress:  "test@example.com",
				ServerAddress: types.Addr("smtp.example.com", 456),
			},
			wantErr: nil,
		}, {
			name: "No sender address",
			cred: &SMTPCredential{
				ServerAddress: types.Addr("smtp.example.com", 456),
			},
			wantErr: ErrSMTPCredentialNoSender,
		}, {
			name: "No server address",
			cred: &SMTPCredential{
				EmailAddress: "test@example.com",
			},
			wantErr: ErrSMTPCredentialNoAddr,
		}, {
			name: "Duplicate name",
			cred: &SMTPCredential{
				EmailAddress:  existing.EmailAddress,
				ServerAddress: types.Addr("smtp.example.com", 456),
			},
			wantErr: ErrSMTPCredentialDuplicate,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if test.wantErr == nil {
				require.NoError(t, test.cred.BeforeWrite(db))
			} else {
				require.ErrorIs(t, test.cred.BeforeWrite(db), test.wantErr)
			}
		})
	}
}
