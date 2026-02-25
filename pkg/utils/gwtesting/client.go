package gwtesting

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/controller"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type TestClientCtx struct {
	Root string
	DB   *database.DB

	Client   *model.Client
	Partner  *model.RemoteAgent
	Account  *model.RemoteAccount
	RulePush *model.Rule
	RulePull *model.Rule

	Service services.Client
}

func NewTestClientCtx(tb testing.TB, protocol, partnerAddr string,
	clientConf, serverConf map[string]any,
) *TestClientCtx {
	tb.Helper()
	require.Contains(tb, Protocols, protocol)

	ctx := &TestClientCtx{
		Root: tb.TempDir(),
		DB:   Database(tb),
	}

	conf.GlobalConfig.Paths = conf.PathsConfig{
		GatewayHome: ctx.Root,
		FilePerms:   0o600,
		DirPerms:    0o700,
	}

	ctx.Client = &model.Client{
		Name:        protocol + "-test-client",
		Protocol:    protocol,
		ProtoConfig: clientConf,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Client).Run())

	ctx.Partner = &model.RemoteAgent{
		Name:        protocol + "-test-partner",
		Address:     Addr(tb, partnerAddr),
		Protocol:    protocol,
		ProtoConfig: serverConf,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Partner).Run())

	ctx.Account = &model.RemoteAccount{
		RemoteAgentID: ctx.Partner.ID,
		Login:         "test-partner-account",
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Account).Run())

	ctx.RulePush = &model.Rule{Name: "push", LocalDir: "push", IsSend: true}
	require.NoError(tb, ctx.DB.Insert(ctx.RulePush).Run())
	require.NoError(tb, fs.MkdirAll(fs.JoinPath(ctx.Root, ctx.RulePush.LocalDir)))

	ctx.RulePull = &model.Rule{Name: "pull", LocalDir: "pull", IsSend: false}
	require.NoError(tb, ctx.DB.Insert(ctx.RulePull).Run())
	require.NoError(tb, fs.MkdirAll(fs.JoinPath(ctx.Root, ctx.RulePull.LocalDir)))

	ctx.Service = Protocols[protocol].MakeClient(ctx.DB, ctx.Client)
	require.NoError(tb, ctx.Service.Start())

	tb.Cleanup(func() {
		require.NoError(tb, ctx.Service.Stop(tb.Context()))
	})

	return ctx
}

func (ctx *TestClientCtx) AddPassword(tb testing.TB, pswd string) {
	tb.Helper()

	cred := &model.Credential{
		RemoteAccountID: utils.NewNullInt64(ctx.Account.ID),
		Type:            auth.Password,
		Value:           pswd,
	}
	require.NoError(tb, ctx.DB.Insert(cred).Run())
}

func (ctx *TestClientCtx) AddCert(tb testing.TB, cert string) {
	tb.Helper()

	cred := &model.Credential{
		RemoteAgentID: utils.NewNullInt64(ctx.Account.ID),
		Type:          auth.TLSTrustedCertificate,
		Value:         cert,
	}
	require.NoError(tb, ctx.DB.Insert(cred).Run())
}

func (ctx *TestClientCtx) makeTransfer(tb testing.TB, file string,
	rule *model.Rule,
) *model.Transfer {
	tb.Helper()

	trans := &model.Transfer{
		RuleID:          rule.ID,
		ClientID:        utils.NewNullInt64(ctx.Client.ID),
		RemoteAccountID: utils.NewNullInt64(ctx.Account.ID),
		SrcFilename:     file,
	}
	require.NoError(tb, ctx.DB.Insert(trans).Run())

	return trans
}

func (ctx *TestClientCtx) run(tb testing.TB, file string, rule *model.Rule) error {
	tb.Helper()
	trans := ctx.makeTransfer(tb, file, rule)

	logger := Logger(tb)
	transCtx, ctxErr := model.GetTransferContext(ctx.DB, logger, trans)
	require.NoError(tb, ctxErr)

	pip, cErr := pipeline.NewClientPipeline(ctx.DB, logger, transCtx, nil)
	requireNoError(tb, cErr)

	transClient, tErr := ctx.Service.InitTransfer(pip)
	requireNoError(tb, tErr)

	clientPip := &controller.ClientPipeline{
		Pip:    pip,
		Client: transClient,
	}

	//nolint:wrapcheck //no need to wrap here
	return clientPip.Run()
}

func (ctx *TestClientCtx) RunUpload(tb testing.TB, file string) error {
	tb.Helper()

	return ctx.run(tb, file, ctx.RulePush)
}

func (ctx *TestClientCtx) RunDownload(tb testing.TB, file string) error {
	tb.Helper()

	return ctx.run(tb, file, ctx.RulePull)
}

func addTaskError(tb testing.TB, db database.Access, rule *model.Rule, chain model.Chain,
) (types.TransferErrorCode, string) {
	tb.Helper()

	task := &model.Task{
		RuleID: rule.ID,
		Chain:  chain,
		Rank:   1,
		Type:   taskstest.TaskErr,
	}
	require.NoError(tb, db.Insert(task).Run())

	return types.TeExternalOperation, fmt.Sprintf("Task %s @ %s %s[%d]: %v",
		task.Type, rule.Name, chain, task.Rank, taskstest.ErrTaskFailed)
}

func (ctx *TestClientCtx) AddPushPreTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePush, model.ChainPre)
}

func (ctx *TestClientCtx) AddPullPreTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePull, model.ChainPre)
}

func (ctx *TestClientCtx) AddPostTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePush, model.ChainPost)
}

func (ctx *TestClientCtx) AddPullPostTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePull, model.ChainPost)
}
