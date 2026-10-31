package rights

import (
	"fmt"
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

	assert.Equal(t, Row{
		Key: "transfers", Label: "<rights.domain.transfers>", NoDelete: true, Granted: "<rights.granted.none>", None: true,
	}, rows[0])
	assert.Equal(t, Row{
		Key: "servers", Label: "<rights.domain.servers>", Read: true, Write: true, Granted: "<rights.granted.rw>",
	}, rows[1])
	assert.Equal(t, Row{Key: "users", Label: "<rights.domain.users>", Delete: true, Granted: "<rights.granted.d>"}, rows[4])
	assert.Equal(t, "admin", rows[5].Key)

	all := Rows(translate, model.PermAll)
	assert.Equal(t, Row{
		Key: "transfers", Label: "<rights.domain.transfers>", Read: true, Write: true, NoDelete: true, Granted: "<rights.granted.rw>",
	}, all[0], "no delete right on the transfers, even with every right")

	for _, row := range all[1:] {
		assert.True(t, row.Read && row.Write && row.Delete, row.Key)
	}
}

// The summary of a folded rights card: every right, none, one or how many.
func TestMeta(t *testing.T) {
	t.Parallel()

	translate := func(key string, args ...any) string { return fmt.Sprintf("%s|%v", key, args) }
	all := []Row{{Read: true, Write: true, Delete: true}, {Read: true, Write: true, NoDelete: true}}

	assert.Equal(t, "rights.meta_all|[]", Meta(translate, all))
	assert.Equal(t, "rights.meta_none|[]", Meta(translate, []Row{{}, {NoDelete: true}}))
	assert.Equal(t, "rights.meta_one|[]", Meta(translate, []Row{{Read: true}, {}}))
	assert.Equal(t, "rights.meta|[n 3]", Meta(translate, []Row{{Read: true, Write: true}, {Read: true}}))
}
