package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReservedInfoKeyUserSets(t *testing.T) {
	t.Parallel()

	assert.True(t, ReservedInfoKey{SetBy: InfoSetByUser}.UserSets())
	assert.True(t, ReservedInfoKey{SetBy: InfoSetByBoth}.UserSets())
	assert.False(t, ReservedInfoKey{SetBy: InfoSetByGateway}.UserSets())
}
