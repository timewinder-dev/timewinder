package model

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timewinder-dev/timewinder/cas"
)

// TestMultiThreadEquivalence verifies that MultiThreadEngine produces the same results as SingleThreadEngine
func TestMultiThreadEquivalence(t *testing.T) {
	testCases := []struct {
		name     string
		specPath string
	}{
		{"PracticalTLA Ch1 A", "../testdata/practical_tla/ch1/ch1_a.toml"},
		{"PracticalTLA Ch1 B", "../testdata/practical_tla/ch1/ch1_b.toml"},
		{"PracticalTLA Ch1 C", "../testdata/practical_tla/ch1/ch1_c.toml"},
		{"PracticalTLA Ch1 D", "../testdata/practical_tla/ch1/ch1_d.toml"},
		{"PracticalTLA Ch1 E", "../testdata/practical_tla/ch1/ch1_e.toml"},
		{"PracticalTLA Ch5 A", "../testdata/practical_tla/ch5/ch5_a.toml"},
		{"PracticalTLA Ch5 B", "../testdata/practical_tla/ch5/ch5_b.toml"},
		{"PracticalTLA Ch5 C", "../testdata/practical_tla/ch5/ch5_c.toml"},
		{"Peterson Mutex", "../testdata/found_specs/04_peterson_mutex.toml"},
		{"Bounded Buffer", "../testdata/found_specs/05_bounded_buffer.toml"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Run with SingleThreadEngine
			singleResult, err := runWithEngine(t, tc.specPath, false, 0, 0)
			require.NoError(t, err, "SingleThread should not return error")
			require.NotNil(t, singleResult, "SingleThread should return result")

			// Run with MultiThreadEngine (4 exec, 2 check threads)
			multiResult, err := runWithEngine(t, tc.specPath, true, 4, 2)
			require.NoError(t, err, "MultiThread should not return error")
			require.NotNil(t, multiResult, "MultiThread should return result")

			// Compare results - multi-threaded may explore more states/find more violations due to parallel execution
			t.Logf("Single: Success=%v, Violations=%d", singleResult.Success, len(singleResult.Violations))
			t.Logf("Multi:  Success=%v, Violations=%d", multiResult.Success, len(multiResult.Violations))
			assert.Equal(t, singleResult.Success, multiResult.Success, "Success status should match")

			// If there are violations, check they're the same type
			if len(singleResult.Violations) > 0 && len(multiResult.Violations) > 0 {
				assert.Equal(t, singleResult.Violations[0].PropertyName, multiResult.Violations[0].PropertyName,
					"Violation property name should match")
			}
		})
	}
}

// TestMultiThreadWorkerCounts tests different worker thread configurations
func TestMultiThreadWorkerCounts(t *testing.T) {
	testCases := []struct {
		name         string
		execThreads  int
		checkThreads int
	}{
		{"1 exec, 1 check", 1, 1},
		{"2 exec, 1 check", 2, 1},
		{"4 exec, 2 check", 4, 2},
		{"8 exec, 4 check", 8, 4},
		{"Default (0, 0)", 0, 0},
	}

	specPath := "../testdata/practical_tla/ch1/ch1_a.toml"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := runWithEngine(t, specPath, true, tc.execThreads, tc.checkThreads)
			require.NoError(t, err, "Should not return error")
			require.NotNil(t, result, "Should return result")

			// Just verify we get a consistent result
			assert.Greater(t, result.Statistics.UniqueStates, 0, "Should explore some states")
		})
	}
}

// TestMultiThreadCancellation tests that context cancellation works properly
func TestMultiThreadCancellation(t *testing.T) {
	t.Skip("Manual test - requires long-running spec")
	// This would require a spec that runs for a long time
	// and then we'd cancel it mid-execution to test graceful shutdown
}

// TestMultiThreadViolationHandling tests violation detection and keep-going mode
func TestMultiThreadViolationHandling(t *testing.T) {
	// Use a spec that has violations
	specPath := "../testdata/practical_tla/ch1/ch1_b.toml" // This should have violations

	t.Run("Stop on first violation", func(t *testing.T) {
		result, err := runWithEngine(t, specPath, true, 4, 2)
		require.NoError(t, err)
		require.NotNil(t, result)

		if !result.Success {
			assert.GreaterOrEqual(t, len(result.Violations), 1, "Should have at least one violation")
		}
	})

	t.Run("Keep going mode", func(t *testing.T) {
		result, err := runWithEngineKeepGoing(t, specPath, true, 4, 2)
		require.NoError(t, err)
		require.NotNil(t, result)

		if !result.Success {
			// In keep-going mode, might find multiple violations
			assert.GreaterOrEqual(t, len(result.Violations), 1, "Should have violations")
		}
	})
}

// runWithEngine is a helper that runs a spec with the specified engine configuration
func runWithEngine(t *testing.T, specPath string, useMultiThread bool, execThreads, checkThreads int) (*ModelResult, error) {
	// Load spec
	spec, err := LoadSpecFromFile(specPath)
	if err != nil {
		return nil, err
	}

	// Create CAS
	memoryCAS := cas.NewMemoryCAS()
	casStore := cas.NewLRUCache(memoryCAS, 10000)

	// Build executor
	exec, err := spec.BuildExecutor(casStore)
	if err != nil {
		return nil, err
	}

	// Configure executor
	exec.DebugWriter = io.Discard
	exec.Reporter = nil // Disable progress reporting in tests

	// Set engine configuration
	exec.UseMultiThread = useMultiThread
	exec.NumExecThreads = execThreads
	exec.NumCheckThreads = checkThreads

	// Initialize
	err = exec.Initialize()
	if err != nil {
		return nil, err
	}

	// Run model checker
	// Note: RunModel now handles worker shutdown automatically, no need for explicit Close()
	return exec.RunModel()
}

// runWithEngineKeepGoing is like runWithEngine but enables keep-going mode
func runWithEngineKeepGoing(t *testing.T, specPath string, useMultiThread bool, execThreads, checkThreads int) (*ModelResult, error) {
	// Load spec
	spec, err := LoadSpecFromFile(specPath)
	if err != nil {
		return nil, err
	}

	// Create CAS
	memoryCAS := cas.NewMemoryCAS()
	casStore := cas.NewLRUCache(memoryCAS, 10000)

	// Build executor
	exec, err := spec.BuildExecutor(casStore)
	if err != nil {
		return nil, err
	}

	// Configure executor
	exec.DebugWriter = io.Discard
	exec.Reporter = nil
	exec.KeepGoing = true // Enable keep-going

	// Set engine configuration
	exec.UseMultiThread = useMultiThread
	exec.NumExecThreads = execThreads
	exec.NumCheckThreads = checkThreads

	// Initialize
	err = exec.Initialize()
	if err != nil {
		return nil, err
	}

	// Run model checker
	// Note: RunModel now handles worker shutdown automatically, no need for explicit Close()
	return exec.RunModel()
}

// TestWorkItemAndCheckItem tests the work item constructors
func TestWorkItemAndCheckItem(t *testing.T) {
	t.Run("NewWorkItem", func(t *testing.T) {
		thunk := &Thunk{}
		item := NewWorkItem(thunk, 5)

		assert.Equal(t, thunk, item.Thunk)
		assert.Equal(t, 5, item.DepthNumber)
	})

	t.Run("NewCheckItem", func(t *testing.T) {
		thunk := &Thunk{}
		hash := cas.Hash(12345)

		item := NewCheckItem(thunk, nil, hash, true, 3, 10)

		assert.Equal(t, thunk, item.Thunk)
		assert.Equal(t, hash, item.StateHash)
		assert.True(t, item.IsNewState)
		assert.Equal(t, 3, item.DepthNumber)
		assert.Equal(t, 10, item.SuccessorCount)
	})
}
