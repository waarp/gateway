package common

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestRefusal(t *testing.T) {
	t.Parallel()

	byTransfers := &database.ForeignKeyError{Msg: "held", Elems: []database.FKElement{&model.Transfer{ID: 7}}, Total: 1}
	byWatchers := &database.ForeignKeyError{Msg: "held", Elems: []database.FKElement{&model.FileWatcher{Flow: "f"}}, Total: 1}
	wrapped := fmt.Errorf("delete: %w", byTransfers)

	assert.True(t, Refused(byTransfers), "an object still used is a refusal")
	assert.True(t, Refused(wrapped), "even wrapped")
	assert.True(t, Refused(database.NewValidationError("no")), "a validation error is a refusal")
	assert.False(t, Refused(errors.New("disk full")), "anything else is a failure")
	assert.False(t, Refused(nil))

	assert.True(t, HeldBy[*model.Transfer](wrapped))
	assert.False(t, HeldBy[*model.FileWatcher](wrapped), "the transfers hold it, not the folder watchers")
	assert.True(t, HeldBy[*model.FileWatcher](byWatchers))
	assert.False(t, HeldBy[*model.Transfer](database.NewValidationError("no")), "a validation error names no holder")
}
