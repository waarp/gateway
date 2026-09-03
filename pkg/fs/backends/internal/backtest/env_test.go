package backtest

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// recorderTB records the outcome instead of reporting it. A failure cannot be
// observed on a real *testing.T without failing the parent test, so the CI
// branch needs a stand-in. Both outcomes abort like the real ones do.
type recorderTB struct {
	testing.TB

	failed  bool
	skipped bool
	message string
}

func (r *recorderTB) Helper() {}

func (r *recorderTB) Skipf(format string, args ...any) {
	r.skipped = true
	r.message = fmt.Sprintf(format, args...)

	runtime.Goexit()
}

func (r *recorderTB) Fatalf(format string, args ...any) {
	r.failed = true
	r.message = fmt.Sprintf(format, args...)

	runtime.Goexit()
}

// runEnvOrSkip calls EnvOrSkip on a goroutine of its own, so that the abort
// performed by the recorder does not take the test down with it.
func runEnvOrSkip(recorder *recorderTB, name string) string {
	var value string

	done := make(chan struct{})

	go func() {
		defer close(done)

		value = EnvOrSkip(recorder, name)
	}()

	<-done

	return value
}

//nolint:paralleltest // cannot be parallelized because it calls t.Setenv
func TestEnvOrSkip(t *testing.T) {
	const varName = "WAARP_BACKTEST_VARIABLE"

	// The literal is deliberate: the constant alone would let a rename pass.
	t.Setenv("CI", "")

	// A value that does not parse as a boolean still means CI, but the tools
	// that set CI=false mean the opposite.
	ciValues := map[string]bool{
		"true": true, "True": true, "1": true, "gitlab": true,
		"false": false, "FALSE": false, "F": false, "0": false, "": false,
	}

	for ciValue, expFail := range ciValues {
		outcome := "skips"
		if expFail {
			outcome = "fails"
		}

		t.Run(fmt.Sprintf("%s when CI=%q", outcome, ciValue), func(t *testing.T) {
			t.Setenv("CI", ciValue)

			recorder := &recorderTB{TB: t}

			value := runEnvOrSkip(recorder, "WAARP_BACKTEST_UNSET_VARIABLE")

			assert.Equal(t, expFail, recorder.failed, "wrong failure outcome")
			assert.Equal(t, !expFail, recorder.skipped, "wrong skip outcome")
			assert.Contains(t, recorder.message, "WAARP_BACKTEST_UNSET_VARIABLE")
			assert.Empty(t, value)
		})
	}

	t.Run("returns the value in CI when the variable is set", func(t *testing.T) {
		t.Setenv("CI", "true")
		t.Setenv(varName, "some-value")

		recorder := &recorderTB{TB: t}

		value := runEnvOrSkip(recorder, varName)

		assert.Equal(t, "some-value", value)
		assert.False(t, recorder.failed, "EnvOrSkip should not have failed the test")
		assert.False(t, recorder.skipped, "EnvOrSkip should not have skipped the test")
	})

	t.Run("returns the value when the variable is set", func(t *testing.T) {
		t.Setenv(varName, "some-value")

		// A skip aborts the subtest, so the check must survive it.
		t.Cleanup(func() {
			assert.False(t, t.Skipped(), "EnvOrSkip should not have skipped the test")
		})

		value := EnvOrSkip(t, varName)

		assert.Equal(t, "some-value", value)
	})

	t.Run("skips when the variable is empty", func(t *testing.T) {
		t.Setenv(varName, "")

		EnvOrSkip(t, varName)
		t.Error("EnvOrSkip should have skipped the test")
	})

	t.Run("skips when the variable is unset", func(t *testing.T) {
		EnvOrSkip(t, "WAARP_BACKTEST_UNSET_VARIABLE")
		t.Error("EnvOrSkip should have skipped the test")
	})
}
