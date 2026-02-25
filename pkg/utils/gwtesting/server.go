package gwtesting

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type TestServerCtx struct {
	Root string
	DB   *database.DB

	Server   *model.LocalAgent
	Account  *model.LocalAccount
	RulePush *model.Rule
	RulePull *model.Rule

	Service services.Server

	endChan chan bool
}

func NewTestServerCtx(tb testing.TB, protocol string, serverConf map[string]any) *TestServerCtx {
	tb.Helper()
	require.Contains(tb, Protocols, protocol)
	const chanBuf = 10

	port := GetLocalPort(tb)
	ctx := &TestServerCtx{
		Root:    tb.TempDir(),
		DB:      Database(tb),
		endChan: make(chan bool, chanBuf),
	}

	conf.GlobalConfig.Paths = conf.PathsConfig{
		GatewayHome: ctx.Root,
		FilePerms:   0o600,
		DirPerms:    0o700,
	}

	ctx.Server = &model.LocalAgent{
		Name:        protocol + "-test-server",
		Address:     types.Addr("localhost", port),
		Protocol:    protocol,
		ProtoConfig: serverConf,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Server).Run())

	ctx.Account = &model.LocalAccount{
		LocalAgentID: ctx.Server.ID,
		Login:        "test-server-account",
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Account).Run())

	ctx.RulePush = &model.Rule{Name: "push", LocalDir: "push", IsSend: false}
	require.NoError(tb, ctx.DB.Insert(ctx.RulePush).Run())
	require.NoError(tb, fs.MkdirAll(fs.JoinPath(ctx.Root, ctx.RulePush.LocalDir)))

	ctx.RulePull = &model.Rule{Name: "pull", LocalDir: "pull", IsSend: true}
	require.NoError(tb, ctx.DB.Insert(ctx.RulePull).Run())
	require.NoError(tb, fs.MkdirAll(fs.JoinPath(ctx.Root, ctx.RulePull.LocalDir)))

	service, ok := Protocols[protocol].MakeServer(ctx.DB, ctx.Server).(TestService)
	require.True(tb, ok)
	service.SetTracer(func() pipeline.Trace {
		return pipeline.Trace{
			OnTransferEnd: func() { ctx.endChan <- true },
		}
	})

	require.NoError(tb, service.Start())
	tb.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		require.NoError(tb, service.Stop(stopCtx))
	})

	return ctx
}

func (ctx *TestServerCtx) WaitEnd(tb testing.TB) {
	tb.Helper()

	select {
	case <-ctx.endChan:
	case <-tb.Context().Done():
		tb.Fatal("timeout waiting for transfer to end")
	}
}

func (ctx *TestServerCtx) AddPassword(tb testing.TB, pswd string) {
	tb.Helper()

	cred := &model.Credential{
		LocalAccountID: utils.NewNullInt64(ctx.Account.ID),
		Type:           auth.Password,
		Value:          pswd,
	}
	require.NoError(tb, ctx.DB.Insert(cred).Run())
}

func (ctx *TestServerCtx) AddCert(tb testing.TB, cert, key string) {
	tb.Helper()

	cred := &model.Credential{
		LocalAgentID: utils.NewNullInt64(ctx.Server.ID),
		Type:         auth.TLSCertificate,
		Value2:       key,
		Value:        cert,
	}
	require.NoError(tb, ctx.DB.Insert(cred).Run())
}

func (ctx *TestServerCtx) AddPushPreTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePush, model.ChainPre)
}

func (ctx *TestServerCtx) AddPullPreTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePull, model.ChainPre)
}

func (ctx *TestServerCtx) AddPushPostTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePush, model.ChainPost)
}

func (ctx *TestServerCtx) AddPullPostTaskError(tb testing.TB) (types.TransferErrorCode, string) {
	tb.Helper()

	return addTaskError(tb, ctx.DB, ctx.RulePull, model.ChainPost)
}
