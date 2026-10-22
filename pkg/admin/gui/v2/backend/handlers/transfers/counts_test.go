package transfers

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errCount = errors.New("count failed")

func TestCountCache(t *testing.T) {
	t.Parallel()

	cache := newCountCache()

	var reads atomic.Int32

	release := make(chan struct{})
	read := func() (map[string]uint64, error) {
		reads.Add(1)
		<-release

		return map[string]uint64{"done": 3}, nil
	}

	// The two parts of the list, read together, count once: the second waits
	// for the reading under way.
	var wg sync.WaitGroup

	results := make([]map[string]uint64, 2)
	for i := range results {
		wg.Go(func() { results[i], _ = cache.get("key", 0, read) })
	}

	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	assert.Equal(t, int32(1), reads.Load(), "one reading for the two parts")
	assert.Equal(t, map[string]uint64{"done": 3}, results[0])
	assert.Equal(t, results[0], results[1])

	// A page change takes the counts read a moment before.
	counts, err := cache.get("key", time.Minute, read)
	require.NoError(t, err)
	assert.Equal(t, map[string]uint64{"done": 3}, counts)
	assert.Equal(t, int32(1), reads.Load(), "no new reading for a page change")

	// The part of the counts always reads them again.
	_, err = cache.get("key", 0, read)
	require.NoError(t, err)
	assert.Equal(t, int32(2), reads.Load())

	// Other filters, other counts.
	_, err = cache.get("other", time.Minute, read)
	require.NoError(t, err)
	assert.Equal(t, int32(3), reads.Load())

	// A failed reading is not kept: the next one reads again.
	_, err = cache.get("failing", time.Minute, func() (map[string]uint64, error) { return nil, errCount })
	require.ErrorIs(t, err, errCount)

	counts, err = cache.get("failing", time.Minute, read)
	require.NoError(t, err)
	assert.Equal(t, map[string]uint64{"done": 3}, counts)
}

func TestCountKey(t *testing.T) {
	t.Parallel()

	crit := &filters{State: "error", Rule: "releves", Partner: "banque-demo"}
	other := *crit
	other.State = "done"

	assert.Equal(t, countKey("paie", crit), countKey("paie", &other), "the counts do not depend on the state")
	assert.NotEqual(t, countKey("paie", crit), countKey("virements", crit), "they depend on the search")

	other.Account = "prod"
	assert.NotEqual(t, countKey("paie", crit), countKey("paie", &other), "and on the filters")
}
