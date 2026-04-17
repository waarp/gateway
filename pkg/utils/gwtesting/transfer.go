package gwtesting

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/gatewayd/services"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/tasks/taskstest"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

const (
	Login    = "foobar"
	Password = "sesame"

	PushFileContent = "push file content"
	PullFileContent = "pull file content"
)

var ErrTest = errors.New("intended test error")

type TestService interface {
	protocol.Server
	SetTracer(tracer func() pipeline.Trace)
}

type TransferCtx struct {
	DB *database.DB

	startOnce     sync.Once
	ClientService protocol.Client
	ServerService TestService

	TransferPush, TransferPull *model.Transfer

	Server       *model.LocalAgent
	LocalAccount *model.LocalAccount

	Client        *model.Client
	Partner       *model.RemoteAgent
	RemoteAccount *model.RemoteAccount

	ClientRulePush, ClientRulePull *model.Rule
	ServerRulePush, ServerRulePull *model.Rule
}

//nolint:funlen //function length is fine (for now)
func TestTransferCtxUnstarted(tb testing.TB, db *database.DB, proto string,
	serverProtoConfig, clientProtoConfig, partnerProtoConfig any,
) *TransferCtx {
	tb.Helper()
	tb.Cleanup(pipeline.List.Reset)

	ctx := &TransferCtx{DB: db}
	port := GetLocalPort(tb)

	conf.GlobalConfig.Paths = conf.PathsConfig{
		GatewayHome:   tb.TempDir(),
		DefaultInDir:  "in",
		DefaultOutDir: "out",
		DefaultTmpDir: "tmp",
		FilePerms:     0o600,
		DirPerms:      0o700,
	}

	paths := &conf.GlobalConfig.Paths
	inPath := filepath.Join(paths.GatewayHome, paths.DefaultInDir)
	outPath := filepath.Join(paths.GatewayHome, paths.DefaultOutDir)
	tmpPath := filepath.Join(paths.GatewayHome, paths.DefaultTmpDir)

	require.NoError(tb, os.Mkdir(inPath, 0o700), `Failed to create the test "in" directory`)
	require.NoError(tb, os.Mkdir(outPath, 0o700), `Failed to create the test "out" directory`)
	require.NoError(tb, os.Mkdir(tmpPath, 0o700), `Failed to create the test "tmp" directory`)

	ctx.makeServer(tb, proto, port, serverProtoConfig)
	ctx.makeClient(tb, proto, port, clientProtoConfig, partnerProtoConfig)
	ctx.makeServerRules(tb)
	ctx.makeClientRules(tb)

	pushFileName := filepath.Join("push_src_dir", "push.file")
	pullFileName := filepath.Join("pull_src_dir", "pull.file")

	pushFilePath := filepath.Join(outPath, pushFileName)
	pullFilePath := filepath.Join(outPath, pullFileName)
	require.NoError(tb, os.MkdirAll(filepath.Dir(pushFilePath), 0o700),
		"Failed to create test push directory")
	require.NoError(tb, os.MkdirAll(filepath.Dir(pullFilePath), 0o700),
		"Failed to create test pull directory")
	require.NoError(tb, os.WriteFile(pushFilePath, []byte(PushFileContent), 0o600),
		"Failed to create test push file")
	require.NoError(tb, os.WriteFile(pullFilePath, []byte(PullFileContent), 0o600),
		"Failed to create test pull file")

	ctx.TransferPush = &model.Transfer{
		RuleID:          ctx.ClientRulePush.ID,
		RemoteAccountID: utils.NewNullInt64(ctx.RemoteAccount.ID),
		ClientID:        utils.NewNullInt64(ctx.Client.ID),
		SrcFilename:     filepath.ToSlash(pushFileName),
		DestFilename:    "push_dst_dir/push.file",
	}
	require.NoError(tb, db.Insert(ctx.TransferPush).Run(), "Failed to insert the test push transfer")

	ctx.TransferPull = &model.Transfer{
		RuleID:          ctx.ClientRulePull.ID,
		RemoteAccountID: utils.NewNullInt64(ctx.RemoteAccount.ID),
		ClientID:        utils.NewNullInt64(ctx.Client.ID),
		SrcFilename:     filepath.ToSlash(pullFileName),
		DestFilename:    "pull_dst_dir/pull.file",
	}
	require.NoError(tb, db.Insert(ctx.TransferPull).Run(), "Failed to insert the test push transfer")

	return ctx
}

func TestTransferCtx(tb testing.TB, db *database.DB, proto string,
	serverProtoConfig, clientProtoConfig, partnerProtoConfig any,
) *TransferCtx {
	tb.Helper()

	ctx := TestTransferCtxUnstarted(tb, db, proto, serverProtoConfig,
		clientProtoConfig, partnerProtoConfig)
	ctx.StartServices(tb)

	return ctx
}

func (ctx *TransferCtx) StartServices(tb testing.TB) {
	tb.Helper()

	ctx.startOnce.Do(func() {
		ctx.startClient(tb)
		ctx.startServer(tb)
	})
}

func (ctx *TransferCtx) makeServer(tb testing.TB,
	proto string, port uint16, serverProtoConf any,
) {
	tb.Helper()

	rawServConf := map[string]any{}
	if serverProtoConf != nil {
		err := utils.JSONConvert(serverProtoConf, &rawServConf)
		require.NoErrorf(tb, err, "Failed to serialize %s server proto config %+v",
			proto, serverProtoConf)
	}

	ctx.Server = &model.LocalAgent{
		Name:        proto + "-server",
		Address:     types.Addr("", port),
		Protocol:    proto,
		ProtoConfig: rawServConf,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Server).Run(), "Failed to insert test server")

	ctx.LocalAccount = &model.LocalAccount{
		LocalAgentID: ctx.Server.ID,
		Login:        Login,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.LocalAccount).Run(), "Failed to insert test local account")

	locAccountCred := &model.Credential{
		LocalAccountID: utils.NewNullInt64(ctx.LocalAccount.ID),
		Name:           Login + "-password-hash",
		Type:           auth.Password,
		Value:          Password,
	}
	require.NoError(tb, ctx.DB.Insert(locAccountCred).Run(), "Failed to insert test local account password")
}

//nolint:dupl //factorizing would be too complicated for no gain in maintainability
func (ctx *TransferCtx) makeServerRules(tb testing.TB) {
	tb.Helper()

	ctx.ServerRulePush = &model.Rule{Name: "push", Comment: "server push", IsSend: false}
	require.NoError(tb, ctx.DB.Insert(ctx.ServerRulePush).Run(), "Failed to insert test server push rule")

	pushPreTask := &model.Task{RuleID: ctx.ServerRulePush.ID, Chain: model.ChainPre, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pushPreTask).Run(), "Failed to insert test server push pre-task")

	pushPostTask := &model.Task{RuleID: ctx.ServerRulePush.ID, Chain: model.ChainPost, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pushPostTask).Run(), "Failed to insert test server push post-task")

	pushErrTask := &model.Task{RuleID: ctx.ServerRulePush.ID, Chain: model.ChainError, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pushErrTask).Run(), "Failed to insert test server push error-task")

	ctx.ServerRulePull = &model.Rule{Name: "pull", Comment: "server pull", IsSend: true}
	require.NoError(tb, ctx.DB.Insert(ctx.ServerRulePull).Run(), "Failed to insert test server pull rule")

	pullPreTask := &model.Task{RuleID: ctx.ServerRulePull.ID, Chain: model.ChainPre, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pullPreTask).Run(), "Failed to insert test server pull pre-task")

	pullPostTask := &model.Task{RuleID: ctx.ServerRulePull.ID, Chain: model.ChainPost, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pullPostTask).Run(), "Failed to insert test server pull post-task")

	pullErrTask := &model.Task{RuleID: ctx.ServerRulePull.ID, Chain: model.ChainError, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pullErrTask).Run(), "Failed to insert test server pull error-task")
}

func (ctx *TransferCtx) makeClient(tb testing.TB,
	proto string, port uint16, clientProtoConf, partnerProtoConf any,
) {
	tb.Helper()

	rawClientConf, rawPartConf := map[string]any{}, map[string]any{}

	if clientProtoConf != nil {
		err := utils.JSONConvert(clientProtoConf, &rawClientConf)
		require.NoErrorf(tb, err, "Failed to serialize %s client proto config %+v",
			proto, clientProtoConf)
	}

	if partnerProtoConf != nil {
		err := utils.JSONConvert(partnerProtoConf, &rawPartConf)
		require.NoErrorf(tb, err, "Failed to serialize %s partner proto config %+v",
			proto, partnerProtoConf)
	}

	ctx.Client = &model.Client{
		Name:        proto + "-client",
		Protocol:    proto,
		ProtoConfig: rawClientConf,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Client).Run(), "Failed to insert test client")

	ctx.Partner = &model.RemoteAgent{
		Name:        proto + "-server",
		Protocol:    proto,
		Address:     types.Addr("127.0.0.1", port),
		ProtoConfig: rawPartConf,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.Partner).Run(), "Failed to insert test partner")

	ctx.RemoteAccount = &model.RemoteAccount{
		RemoteAgentID: ctx.Partner.ID,
		Login:         Login,
	}
	require.NoError(tb, ctx.DB.Insert(ctx.RemoteAccount).Run(), "Failed to insert test remote account")

	locAccountCred := &model.Credential{
		RemoteAccountID: utils.NewNullInt64(ctx.RemoteAccount.ID),
		Name:            Login + "-password",
		Type:            auth.Password,
		Value:           Password,
	}
	require.NoError(tb, ctx.DB.Insert(locAccountCred).Run(), "Failed to insert test remote account password")
}

//nolint:dupl //factorizing would be too complicated for no gain in maintainability
func (ctx *TransferCtx) makeClientRules(tb testing.TB) {
	tb.Helper()

	features := protocolsList[ctx.Server.Protocol]

	ctx.ClientRulePush = &model.Rule{Name: "push", Comment: "client push", IsSend: true}
	if !features.RuleName {
		ctx.ClientRulePush.RemoteDir = "push"
	}

	require.NoError(tb, ctx.DB.Insert(ctx.ClientRulePush).Run(), "Failed to insert test client push rule")

	pushPreTask := &model.Task{RuleID: ctx.ClientRulePush.ID, Chain: model.ChainPre, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pushPreTask).Run(), "Failed to insert test client push pre-task")

	pushPostTask := &model.Task{RuleID: ctx.ClientRulePush.ID, Chain: model.ChainPost, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pushPostTask).Run(), "Failed to insert test client push post-task")

	pushErrTask := &model.Task{RuleID: ctx.ClientRulePush.ID, Chain: model.ChainError, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pushErrTask).Run(), "Failed to insert test client push error-task")

	ctx.ClientRulePull = &model.Rule{Name: "pull", Comment: "client pull", IsSend: false}
	if !features.RuleName {
		ctx.ClientRulePull.RemoteDir = "pull"
	}

	require.NoError(tb, ctx.DB.Insert(ctx.ClientRulePull).Run(), "Failed to insert test client pull rule")

	pullPreTask := &model.Task{RuleID: ctx.ClientRulePull.ID, Chain: model.ChainPre, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pullPreTask).Run(), "Failed to insert test client pull pre-task")

	pullPostTask := &model.Task{RuleID: ctx.ClientRulePull.ID, Chain: model.ChainPost, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pullPostTask).Run(), "Failed to insert test client pull post-task")

	pullErrTask := &model.Task{RuleID: ctx.ClientRulePull.ID, Chain: model.ChainError, Type: taskstest.TaskOK}
	require.NoError(tb, ctx.DB.Insert(pullErrTask).Run(), "Failed to insert test client pull error-task")
}

func (ctx *TransferCtx) startClient(tb testing.TB) {
	tb.Helper()

	module := protocolsList[ctx.Client.Protocol]
	ctx.ClientService = module.NewClient(ctx.DB, ctx.Client)
	services.Clients.Add(ctx.Client, ctx.ClientService)

	require.NoError(tb, ctx.ClientService.Start(), "Failed to start the client")

	tb.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.NoError(tb, ctx.ClientService.Stop(stopCtx), "Failed to stop the client")
	})
}

func (ctx *TransferCtx) startServer(tb testing.TB) {
	tb.Helper()

	module := protocolsList[ctx.Server.Protocol]
	service := module.NewServer(ctx.DB, ctx.Server)
	services.Servers.Add(ctx.Server, service)

	require.Implements(tb, (*TestService)(nil), service,
		"The service must implement the interface for test services")

	ctx.ServerService = service.(TestService) //nolint:forcetypeassert,errcheck //type is checked above

	require.NoError(tb, service.Start(), "Failed to start the server")
	tb.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		require.NoError(tb, service.Stop(stopCtx), "Failed to stop the server")
	})
}

func (ctx *TransferCtx) AddTaskError(tb testing.TB, rule *model.Rule, chain model.Chain) {
	tb.Helper()

	require.NoError(tb, ctx.DB.Insert(&model.Task{
		RuleID: rule.ID,
		Chain:  chain,
		Rank:   1,
		Type:   taskstest.TaskErr,
	}).Run())
}

func AddClientDataError(tb testing.TB, pip *Pipeline) {
	tb.Helper()

	if pip.Pip.TransCtx.Rule.IsSend {
		pip.Pip.Trace.OnRead = func(int64) error { return ErrTest }
	} else {
		pip.Pip.Trace.OnWrite = func(int64) error { return ErrTest }
	}
}

func (ctx *TransferCtx) AddServerDataError(tb testing.TB, rule *model.Rule) {
	tb.Helper()

	if rule.IsSend {
		ctx.ServerService.SetTracer(func() pipeline.Trace {
			return pipeline.Trace{
				OnRead: func(int64) error { return ErrTest },
			}
		})
	} else {
		ctx.ServerService.SetTracer(func() pipeline.Trace {
			return pipeline.Trace{
				OnWrite: func(int64) error { return ErrTest },
			}
		})
	}
}

func (ctx *TransferCtx) AddCred(tb testing.TB, cred *model.Credential) {
	tb.Helper()

	require.NoError(tb, ctx.DB.Insert(cred).Run())
}
