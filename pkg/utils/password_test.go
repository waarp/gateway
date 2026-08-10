package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHash(t *testing.T) {
	t.Parallel()

	const password = "testpassword"

	hash, err := HashPassword(bcrypt.MinCost, password)
	require.NoError(t, err)

	assert.True(t, IsHash(hash))
	assert.True(t, IsHashOf(hash, password))
}
