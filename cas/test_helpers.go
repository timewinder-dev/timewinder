package cas

import (
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// Test helper functions to expose internal decompose/recompose functions
// These are used by unit tests and integration tests

// DecomposeStateForTest exposes decomposeState for testing
func DecomposeStateForTest(c *MemoryCAS, s *interp.State) (Hash, error) {
	return decomposeState(c, s)
}

// RecomposeStateForTest exposes recomposeState for testing
func RecomposeStateForTest(c *MemoryCAS, hash Hash) (*interp.State, error) {
	return recomposeState(c, hash)
}

// DecomposeStackFrameForTest exposes decomposeStackFrame for testing
func DecomposeStackFrameForTest(c *MemoryCAS, f *interp.StackFrame) (Hash, error) {
	return decomposeStackFrame(c, f)
}

// RecomposeStackFrameForTest exposes recomposeStackFrame for testing
func RecomposeStackFrameForTest(c *MemoryCAS, hash Hash) (*interp.StackFrame, error) {
	return recomposeStackFrame(c, hash)
}

// DecomposeValueForTest exposes decomposeValue for testing
func DecomposeValueForTest(c *MemoryCAS, v vm.Value) (Hash, error) {
	return decomposeValue(c, v)
}

// RecomposeValueForTest exposes recomposeValue for testing
func RecomposeValueForTest(c *MemoryCAS, hash Hash) (vm.Value, error) {
	return recomposeValue(c, hash)
}
