package pesit

import (
	"testing"

	"code.waarp.fr/lib/pesit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/utils/gwtesting"
)

func TestGetRuleByNameNotFound(t *testing.T) {
	db := gwtesting.Database(t)
	handler := &transferHandler{db: db, logger: gwtesting.Logger(t)}

	_, err := handler.getRuleByName("does-not-exist", true)
	require.Error(t, err)

	var diag pesit.Diagnostic

	require.ErrorAs(t, err, &diag)

	// An unknown rule is a mistake of the caller, not a failure of the
	// server: the diagnostic must say so, rather than "database error".
	assert.Equal(t, pesit.CodeParameterError, diag.GetCode())
}
