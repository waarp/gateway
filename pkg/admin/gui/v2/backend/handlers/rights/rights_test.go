package rights

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestLetters(t *testing.T) {
	t.Parallel()

	servers := Domains[1]
	require.Equal(t, "servers", servers.Key)

	assert.Equal(t, "---", servers.Letters(0))
	assert.Equal(t, "r--", servers.Letters(model.PermServersRead))
	assert.Equal(t, "rw-", servers.Letters(model.PermServersRead|model.PermServersWrite))
	assert.Equal(t, "rwd", servers.Letters(model.PermAll))
	assert.Equal(t, "-w-", servers.Letters(model.PermServersWrite), "every combination of the mask is valid")
	assert.Equal(t, "---", servers.Letters(model.PermPartnersRead|model.PermPartnersWrite), "another domain")

	assert.Equal(t, 0, servers.Count(0))
	assert.Equal(t, 2, servers.Count(model.PermServersRead|model.PermServersDelete))
	assert.Equal(t, 3, servers.Count(model.PermAll))

	// The transfers have no delete right: the product never checks it.
	transfers := Domains[0]
	require.True(t, transfers.NoDelete)
	assert.Equal(t, "rw-", transfers.Letters(model.PermAll))
	assert.Equal(t, 2, transfers.Count(model.PermAll))
	assert.Equal(t, "rights.domain.transfers", transfers.LabelKey())
}

func TestRows(t *testing.T) {
	t.Parallel()

	translate := func(key string, _ ...any) string { return "<" + key + ">" }

	rows := Rows(translate, model.PermServersRead|model.PermServersWrite|model.PermUsersDelete)
	require.Len(t, rows, 6)

	assert.Equal(t, Row{Key: "transfers", Label: "<rights.domain.transfers>", NoDelete: true}, rows[0])
	assert.Equal(t, Row{Key: "servers", Label: "<rights.domain.servers>", Read: true, Write: true}, rows[1])
	assert.Equal(t, Row{Key: "users", Label: "<rights.domain.users>", Delete: true}, rows[4])
	assert.Equal(t, "admin", rows[5].Key)

	all := Rows(translate, model.PermAll)
	assert.Equal(t, Row{Key: "transfers", Label: "<rights.domain.transfers>", Read: true, Write: true, NoDelete: true}, all[0],
		"no delete right on the transfers, even with every right")

	for _, row := range all[1:] {
		assert.True(t, row.Read && row.Write && row.Delete, row.Key)
	}
}
