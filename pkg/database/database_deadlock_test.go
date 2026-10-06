package database

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testDeadlock simulates a classic deadlock: two transactions lock two rows
// in opposite order, so each one ends up waiting for the lock held by the
// other. The database detects the cycle and aborts one of them (error 1213 on
// MySQL/MariaDB, SQLSTATE 40P01 on PostgreSQL). The test then checks that
// DB.Transaction transparently retries the aborted transaction, so that both
// of them eventually succeed.
//
// The given DB must already be started. The function creates (and drops) its
// own scratch table.
func testDeadlock(t *testing.T, db *DB) {
	t.Helper()

	const testTimeout = 30 * time.Second

	t.Cleanup(func() {
		if err := db.Exec("DROP TABLE IF EXISTS deadlock_test"); err != nil {
			t.Logf("Failed to drop table: %v", err)
		}
	})

	setup := []string{
		"DROP TABLE IF EXISTS deadlock_test",
		"CREATE TABLE deadlock_test (id INT PRIMARY KEY, val INT NOT NULL)",
		"INSERT INTO deadlock_test (id, val) VALUES (1, 0), (2, 0)",
	}
	for _, stmt := range setup {
		if err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}

	// Both transactions update their "first" row, then wait for each other
	// (only on their first attempt), then update the other row. This guarantees
	// that each one holds the lock the other needs -> deadlock. The victim is
	// rolled back and retried by DB.Transaction; on the retry, the barrier is
	// skipped so the transaction simply runs to completion.
	barrier := make(chan struct{})
	ready := make(chan struct{}, 2)

	runTransaction := func(first, second int, attempts *atomic.Int32) error {
		return db.Transaction(func(ses *Session) error {
			attempt := attempts.Add(1)

			if err := ses.Exec("UPDATE deadlock_test SET val = ? WHERE id = ?",
				first, first); err != nil {
				return err
			}

			if attempt == 1 {
				ready <- struct{}{}
				<-barrier
			}

			return ses.Exec("UPDATE deadlock_test SET val = ? WHERE id = ?",
				first, second)
		})
	}

	var (
		attempts1, attempts2 atomic.Int32
		err1, err2           error
		wg                   sync.WaitGroup
	)

	wg.Go(func() { err1 = runTransaction(1, 2, &attempts1) })
	wg.Go(func() { err2 = runTransaction(2, 1, &attempts2) })

	// Wait for both transactions to hold their first lock, then release them
	// so that they collide.
	timeout := time.After(testTimeout)
	for range 2 {
		select {
		case <-ready:
		case <-timeout:
			t.Fatal("timed out waiting for the transactions to acquire their first lock")
		}
	}

	close(barrier)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-timeout:
		t.Fatal("timed out waiting for the deadlock to be resolved")
	}

	// Thanks to the retry mechanism, both transactions must have succeeded...
	if err1 != nil {
		t.Errorf("transaction 1 failed despite the retry mechanism: %v", err1)
	}

	if err2 != nil {
		t.Errorf("transaction 2 failed despite the retry mechanism: %v", err2)
	}

	// ...and exactly one of them (the deadlock victim) must have been retried.
	n1, n2 := attempts1.Load(), attempts2.Load()

	switch {
	case n1 == 2 && n2 == 1:
		t.Log("transaction 1 was the deadlock victim and was retried")
	case n1 == 1 && n2 == 2:
		t.Log("transaction 2 was the deadlock victim and was retried")
	default:
		t.Fatalf("expected exactly one transaction to be retried once, "+
			"got %d attempt(s) for tx1 and %d attempt(s) for tx2", n1, n2)
	}

	// Finally, the table must reflect the last committed transaction (the
	// retried one), i.e. both rows must hold the same value.
	var val1, val2 int

	if err := db.QueryRow("SELECT val FROM deadlock_test WHERE id = 1").Scan(&val1); err != nil {
		t.Fatal(err)
	}

	if err := db.QueryRow("SELECT val FROM deadlock_test WHERE id = 2").Scan(&val2); err != nil {
		t.Fatal(err)
	}

	if val1 != val2 {
		t.Errorf("expected both rows to hold the same value, got id=1: %d, id=2: %d", val1, val2)
	}
}
