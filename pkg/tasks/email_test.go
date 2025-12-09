//go:build manual_test

package tasks

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

type testFile struct {
	Credential struct {
		SenderAddress string `json:"senderAddress"`
		ServerAddress string `json:"serverAddress"`
		Login         string `json:"login"`
		Password      string `json:"password"`
	} `json:"credential"`
	Template struct {
		Subject     string   `json:"subject"`
		Body        string   `json:"body"`
		Attachments []string `json:"attachments"`
	} `json:"template"`
	Recipients []string `json:"recipients"`
}

func TestMail(t *testing.T) {
	// Path to the JSON file containing the task arguments.
	const argsJsonFile = ""

	logger := testhelpers.GetTestLogger(t)
	args := testFile{}

	cont, err := os.ReadFile(argsJsonFile)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(cont, &args))

	db := dbtest.TestDatabase(t)
	task := &emailTask{}
	ctx := t.Context()

	servAddr, sErr := types.NewAddress(args.Credential.ServerAddress)
	require.NoError(t, sErr)

	cred := model.SMTPCredential{
		EmailAddress:  args.Credential.SenderAddress,
		ServerAddress: *servAddr,
		Login:         args.Credential.Login,
		Password:      database.SecretText(args.Credential.Password),
	}
	require.NoError(t, db.Insert(&cred).Run())

	template := model.EmailTemplate{
		Name:        "test_template",
		Subject:     args.Template.Subject,
		Body:        args.Template.Body,
		Attachments: args.Template.Attachments,
	}
	require.NoError(t, db.Insert(&template).Run())

	taskParams := map[string]string{
		"sender":     cred.EmailAddress,
		"recipients": strings.Join(args.Recipients, ", "),
		"template":   template.Name,
	}

	tCtx := &model.TransferContext{
		Transfer: &model.Transfer{ID: 1234},
	}

	require.NoError(t, task.Run(ctx, taskParams, db, logger, tCtx, nil))
}
