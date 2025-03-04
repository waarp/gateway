package gwtesting

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

const (
	LocalhostCert = testhelpers.LocalhostCert
	LocalhostKey  = testhelpers.LocalhostKey
)

func (ctx *TransferCtx) AddCred(tb testing.TB, cred *model.Credential) {
	tb.Helper()

	require.NoError(tb, ctx.db.Insert(cred).Run())
}
