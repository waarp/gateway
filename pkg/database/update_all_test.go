package database

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateAll checks the update of several entries in one statement: the
// named columns alone are written, zero values included, to the entries the
// conditions match and to no other; the associations are left alone; an
// update without a column or without a condition is refused.
func TestUpdateAll(t *testing.T) {
	t.Parallel()

	logs := &bytes.Buffer{}
	db := newUpdateTestDB(t, logs)

	first := &updParent{Name: "first", Size: 12}
	second := &updParent{Name: "second", Size: 34}
	third := &updParent{Name: "third", Size: 56}
	require.NoError(t, db.engine.Create(first).Error)
	require.NoError(t, db.engine.Create(second).Error)
	require.NoError(t, db.engine.Create(third).Error)

	t.Run("the named columns of the matching entries", func(t *testing.T) {
		values := &updParent{Name: "renamed", Size: 0, Children: updChildren{{Name: "child"}}}

		logs.Reset()
		require.NoError(t, db.UpdateAll(values).Cols("size").In("id", first.ID, second.ID).Run())
		assert.NotContains(t, logs.String(), "ON CONFLICT")

		var after []updParent
		require.NoError(t, db.engine.Order("id").Find(&after).Error)
		require.Len(t, after, 3)
		assert.Equal(t, []updParent{
			{ID: first.ID, Name: "first", Size: 0},
			{ID: second.ID, Name: "second", Size: 0},
			{ID: third.ID, Name: "third", Size: 56},
		}, after, "the size alone, a zero value, on the two entries named")

		var children updChildren
		require.NoError(t, db.engine.Find(&children).Error)
		assert.Empty(t, children, "the update must not have written the association")
	})

	t.Run("a Where condition, in a transaction", func(t *testing.T) {
		require.NoError(t, db.Transaction(func(ses *Session) error {
			return ses.UpdateAll(&updParent{Name: "big"}).Cols("name").Where("size>?", 50).Run()
		}))

		var after []updParent
		require.NoError(t, db.engine.Order("id").Find(&after).Error)
		assert.Equal(t, []string{"first", "second", "big"}, []string{after[0].Name, after[1].Name, after[2].Name})
	})

	t.Run("refused without a column", func(t *testing.T) {
		err := db.UpdateAll(&updParent{Name: "x"}).In("id", first.ID).Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "names the columns")
	})

	t.Run("refused without a condition", func(t *testing.T) {
		err := db.UpdateAll(&updParent{Name: "x"}).Cols("name").Run()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "needs a condition")

		err = db.UpdateAll(&updParent{Name: "x"}).Cols("name").In("id").Run()
		require.Error(t, err, "an IN without any value is no condition")

		var after []updParent
		require.NoError(t, db.engine.Where("name=?", "x").Find(&after).Error)
		assert.Empty(t, after, "nothing written")
	})
}
