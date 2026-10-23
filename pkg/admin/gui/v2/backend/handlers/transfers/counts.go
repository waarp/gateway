package transfers

import (
	"strings"
	"sync"
	"time"
)

// countCache keeps the counts of the states of a search and its filters for a
// moment (2026-10-22, the live list of the kit): the two parts of the list,
// read at the same time, count once (the second waits for the reading under
// way), and a page change reads the page alone. The counts change with the
// transfers: the part of the counts always reads them again, the part of the
// rows takes them when they are recent.
type countCache struct {
	mu      sync.Mutex
	entries map[string]*countEntry
}

type countEntry struct {
	done   chan struct{} // closed once the counts are read
	at     time.Time
	counts map[string]uint64
	err    error
}

// countsForget is how long an entry is kept at most.
const countsForget = time.Minute

func newCountCache() *countCache {
	return &countCache{entries: map[string]*countEntry{}}
}

// get gives the counts of a key: those read less than maxAge ago, else those
// of a reading under way, else a new reading by read.
func (c *countCache) get(key string, maxAge time.Duration, read func() (map[string]uint64, error),
) (map[string]uint64, error) {
	c.mu.Lock()

	if entry, ok := c.entries[key]; ok {
		select {
		case <-entry.done:
			if entry.err == nil && time.Since(entry.at) < maxAge {
				c.mu.Unlock()

				return entry.counts, nil
			}
		default: // a reading under way: its counts are the freshest there will be
			c.mu.Unlock()
			<-entry.done

			return entry.counts, entry.err
		}
	}

	entry := &countEntry{done: make(chan struct{})}
	c.entries[key] = entry
	c.forgetOld()
	c.mu.Unlock()

	entry.counts, entry.err = read()
	entry.at = time.Now()
	close(entry.done)

	return entry.counts, entry.err
}

// forgetOld drops the entries read long ago (the lock is held).
func (c *countCache) forgetOld() {
	for key, entry := range c.entries {
		select {
		case <-entry.done:
			if time.Since(entry.at) > countsForget {
				delete(c.entries, key)
			}
		default:
		}
	}
}

// countKey is what the counts of the states depend on: the search and the
// filters, not the state, the sort nor the page.
func countKey(q string, crit *filters) string {
	return strings.Join([]string{
		q, crit.Rule, crit.Dir, crit.Partner, crit.Account, crit.Protocol, crit.From, crit.To,
	}, "\x00")
}
