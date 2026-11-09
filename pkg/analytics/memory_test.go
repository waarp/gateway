package analytics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTotalMemory(t *testing.T) {
	total, ok := TotalMemory()
	assert.True(t, ok)
	assert.Positive(t, total)
}
