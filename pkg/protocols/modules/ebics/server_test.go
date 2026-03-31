package ebics

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

type ebicsConfigChecker struct{}

func (ebicsConfigChecker) IsValidProtocol(proto string) bool {
	return proto == EBICS
}

func (ebicsConfigChecker) CheckServerConfig(proto string, conf map[string]any) error {
	if proto != EBICS {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	structConf := defaultServerConfig()
	if err := utils.JSONConvert(conf, structConf); err != nil {
		return err
	}

	if err := structConf.ValidServer(); err != nil {
		return err
	}

	clear(conf)

	return utils.JSONConvert(structConf, &conf)
}

func (ebicsConfigChecker) CheckClientConfig(proto string, conf map[string]any) error {
	if proto != EBICS {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	structConf := defaultClientConfig()
	if err := utils.JSONConvert(conf, structConf); err != nil {
		return err
	}

	if err := structConf.ValidClient(); err != nil {
		return err
	}

	clear(conf)

	return utils.JSONConvert(structConf, &conf)
}

func (ebicsConfigChecker) CheckPartnerConfig(proto string, conf map[string]any) error {
	if proto != EBICS {
		return fmt.Errorf("unknown protocol %q", proto)
	}

	structConf := defaultPartnerConfig()
	if err := utils.JSONConvert(conf, structConf); err != nil {
		return err
	}

	if err := structConf.ValidPartner(); err != nil {
		return err
	}

	clear(conf)

	return utils.JSONConvert(structConf, &conf)
}

func setEBICSConfigChecker(t *testing.T) {
	t.Helper()

	oldChecker := model.ConfigChecker
	model.ConfigChecker = ebicsConfigChecker{}
	t.Cleanup(func() { model.ConfigChecker = oldChecker })
}

func TestServerStartStop(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	agent := insertTestEBICSServer(t, db, gwtesting.GetLocalPort(t))

	service := NewServer(db, agent)

	require.NoError(t, service.Start())
	code, message := service.State()
	require.Equal(t, utils.StateRunning, code)
	require.Empty(t, message)
	require.NotNil(t, service.listener)
	require.NotNil(t, service.httpServer)
	require.NotNil(t, service.ebicsServer)
	require.NotNil(t, service.providerStore)
	require.NotNil(t, service.config)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	require.NoError(t, service.Stop(ctx))
	code, message = service.State()
	require.Equal(t, utils.StateOffline, code)
	require.Empty(t, message)
	require.Nil(t, service.listener)
	require.Nil(t, service.httpServer)
	require.Nil(t, service.ebicsServer)
	require.Nil(t, service.providerStore)
	require.Nil(t, service.config)
}

func TestServerStartTwiceFails(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	agent := insertTestEBICSServer(t, db, gwtesting.GetLocalPort(t))

	service := NewServer(db, agent)
	require.NoError(t, service.Start())

	err := service.Start()
	require.ErrorIs(t, err, utils.ErrAlreadyRunning)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, service.Stop(ctx))
}

func TestServerStopWhenNotRunningFails(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	agent := insertTestEBICSServer(t, db, gwtesting.GetLocalPort(t))

	service := NewServer(db, agent)

	err := service.Stop(context.Background())
	require.ErrorIs(t, err, utils.ErrNotRunning)
}

func TestServerStartInvalidConfigSetsErrorState(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	service := NewServer(db, &model.LocalAgent{
		Name:     "ebics-invalid-config",
		Protocol: EBICS,
		Address:  types.Addr("localhost", gwtesting.GetLocalPort(t)),
		ProtoConfig: map[string]any{
			"requestTimeout": -1,
		},
	})

	err := service.Start()
	require.Error(t, err)
	require.ErrorContains(t, err, "requestTimeout")

	code, message := service.State()
	require.Equal(t, utils.StateError, code)
	require.Contains(t, message, "requestTimeout")
}

func TestServerStartListenFailureSetsErrorState(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)
	port := gwtesting.GetLocalPort(t)
	agent := insertTestEBICSServer(t, db, port)

	blocker, err := net.Listen("tcp", agent.Address.String())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, blocker.Close()) })

	service := NewServer(db, agent)

	err = service.Start()
	require.Error(t, err)
	require.ErrorContains(t, err, "start EBICS TLS listener")

	code, message := service.State()
	require.Equal(t, utils.StateError, code)
	require.Contains(t, message, "start EBICS TLS listener")
}

func TestResolveEBICSXSDDirUsesEnvVarWhenDirectoryExists(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("EBICS_XSD_DIR", dir)

	resolved, ok := resolveEBICSXSDDir()
	require.True(t, ok)
	require.Equal(t, dir, resolved)
}

func TestResolveEBICSXSDDirRejectsMissingEnvDir(t *testing.T) {
	t.Setenv("EBICS_XSD_DIR", t.TempDir()+"\\missing")

	resolved, ok := resolveEBICSXSDDir()
	require.False(t, ok)
	require.Empty(t, resolved)
}

func TestDirExists(t *testing.T) {
	require.True(t, dirExists(t.TempDir()))
	require.False(t, dirExists(t.TempDir()+"\\missing"))
}

func insertTestEBICSServer(t *testing.T, db *database.DB, port uint16) *model.LocalAgent {
	t.Helper()

	agent := &model.LocalAgent{
		Name:     "ebics-test-server",
		Protocol: EBICS,
		Address:  types.Addr("localhost", port),
	}
	require.NoError(t, db.Insert(agent).Run())

	cred := &model.Credential{
		LocalAgentID: utils.NewNullInt64(agent.ID),
		Type:         auth.TLSCertificate,
		Value:        testhelpers.LocalhostCert,
		Value2:       testhelpers.LocalhostKey,
	}
	require.NoError(t, db.Insert(cred).Run())

	return agent
}
