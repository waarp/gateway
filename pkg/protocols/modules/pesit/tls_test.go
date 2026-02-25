package pesit

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication/auth"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestTLS(t *testing.T) {
	db := gwtesting.Database(t)
	ctx := gwtesting.TestTransferCtx(t, db, PesitTLS, nil, nil, nil)

	ctx.AddCred(t, &model.Credential{
		Name:         "pesit_server_cert",
		LocalAgentID: utils.NewNullInt64(ctx.Server.ID),
		Type:         auth.TLSCertificate,
		Value2:       gwtesting.LocalhostKeyPEM,
		Value:        gwtesting.LocalhostCertPEM,
	})
	ctx.AddCred(t, &model.Credential{
		Name:          "pesit_partner_cert",
		RemoteAgentID: utils.NewNullInt64(ctx.Partner.ID),
		Type:          auth.TLSTrustedCertificate,
		Value:         gwtesting.LocalhostCertPEM,
	})

	t.Run("Given a PESIT pull transfer", func(t *testing.T) {
		t.Parallel()
		pip := ctx.PullPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.NoError(t, pip.Run(), "Then the transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPullTransferOK(t)
			})
		})
	})

	t.Run("Given a PESIT push client", func(t *testing.T) {
		t.Parallel()
		pip := ctx.PushPipeline(t)

		t.Run("When executing the transfer", func(t *testing.T) {
			require.NoError(t, pip.Run(), "Then the transfer should execute without error")

			t.Run("Then it should have finished both the client & the server transfers", func(t *testing.T) {
				ctx.CheckPushTransferOK(t)
			})
		})
	})
}
