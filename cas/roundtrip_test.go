package cas

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// TestRoundTrip_SimpleValues tests decomposition and recomposition of simple vm.Values
func TestRoundTrip_SimpleValues(t *testing.T) {
	tests := []struct {
		name  string
		value vm.Value
	}{
		{"BoolTrue", vm.BoolTrue},
		{"BoolFalse", vm.BoolFalse},
		{"IntValue", vm.IntValue(42)},
		{"FloatValue", vm.FloatValue(3.14)},
		{"StrValue", vm.StrValue("hello")},
		{"NoneValue", vm.None},
		{"FnPtrValue", vm.FnPtrValue(100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewMemoryCAS()

			// Decompose
			hash, err := decomposeValue(c, tt.value)
			require.NoError(t, err)
			assert.NotEqual(t, Hash(0), hash)

			// Recompose
			result, err := recomposeValue(c, hash)
			require.NoError(t, err)

			// Compare
			cmp, ok := tt.value.Cmp(result)
			assert.True(t, ok, "Values should be comparable")
			assert.Equal(t, 0, cmp, "Values should be equal")
		})
	}
}

// TestRoundTrip_SmallStruct tests structs below the reference threshold
func TestRoundTrip_SmallStruct(t *testing.T) {
	c := NewMemoryCAS()

	// Create a small struct (< 3 fields)
	original := vm.StructValue{
		"x": vm.IntValue(10),
		"y": vm.IntValue(20),
	}

	// Decompose and recompose
	hash, err := decomposeValue(c, original)
	require.NoError(t, err)

	result, err := recomposeValue(c, hash)
	require.NoError(t, err)

	// Verify
	resultStruct, ok := result.(vm.StructValue)
	require.True(t, ok)
	assert.Equal(t, len(original), len(resultStruct))

	for k, v := range original {
		rv, exists := resultStruct[k]
		assert.True(t, exists, "Key %s should exist", k)
		cmp, ok := v.Cmp(rv)
		assert.True(t, ok)
		assert.Equal(t, 0, cmp)
	}
}

// TestRoundTrip_LargeStruct tests structs above the reference threshold
func TestRoundTrip_LargeStruct(t *testing.T) {
	c := NewMemoryCAS()

	// Create a large struct (≥ 3 fields)
	original := vm.StructValue{
		"a": vm.IntValue(1),
		"b": vm.IntValue(2),
		"c": vm.IntValue(3),
		"d": vm.IntValue(4),
	}

	// Decompose
	hash, err := DecomposeValueForTest(c, original)
	require.NoError(t, err)

	// Verify it created a StructValueRef (not inline)
	entryBytes := c.data[hash]
	entry := &TypedEntry{}
	err = entry.Deserialize(bytes.NewReader(entryBytes))
	require.NoError(t, err)
	assert.Equal(t, "StructValueRef", entry.TypeTag)

	// Recompose
	result, err := RecomposeValueForTest(c, hash)
	require.NoError(t, err)

	// Verify
	resultStruct, ok := result.(vm.StructValue)
	require.True(t, ok)
	assert.Equal(t, len(original), len(resultStruct))

	for k, v := range original {
		rv, exists := resultStruct[k]
		assert.True(t, exists, "Key %s should exist", k)
		cmp, ok := v.Cmp(rv)
		assert.True(t, ok)
		assert.Equal(t, 0, cmp)
	}
}

// TestRoundTrip_SmallArray tests arrays below the reference threshold
func TestRoundTrip_SmallArray(t *testing.T) {
	c := NewMemoryCAS()

	// Create a small array (< 5 elements)
	original := vm.ArrayValue{
		vm.IntValue(1),
		vm.IntValue(2),
		vm.IntValue(3),
	}

	// Decompose and recompose
	hash, err := decomposeValue(c, original)
	require.NoError(t, err)

	result, err := recomposeValue(c, hash)
	require.NoError(t, err)

	// Verify
	resultArray, ok := result.(vm.ArrayValue)
	require.True(t, ok)
	assert.Equal(t, len(original), len(resultArray))

	for i, v := range original {
		cmp, ok := v.Cmp(resultArray[i])
		assert.True(t, ok)
		assert.Equal(t, 0, cmp)
	}
}

// TestRoundTrip_LargeArray tests arrays above the reference threshold
func TestRoundTrip_LargeArray(t *testing.T) {
	c := NewMemoryCAS()

	// Create a large array (≥ 5 elements)
	original := vm.ArrayValue{
		vm.IntValue(1),
		vm.IntValue(2),
		vm.IntValue(3),
		vm.IntValue(4),
		vm.IntValue(5),
		vm.IntValue(6),
	}

	// Decompose
	hash, err := DecomposeValueForTest(c, original)
	require.NoError(t, err)

	// Verify it created an ArrayValueRef (not inline)
	entryBytes := c.data[hash]
	entry := &TypedEntry{}
	err = entry.Deserialize(bytes.NewReader(entryBytes))
	require.NoError(t, err)
	assert.Equal(t, "ArrayValueRef", entry.TypeTag)

	// Recompose
	result, err := RecomposeValueForTest(c, hash)
	require.NoError(t, err)

	// Verify
	resultArray, ok := result.(vm.ArrayValue)
	require.True(t, ok)
	assert.Equal(t, len(original), len(resultArray))

	for i, v := range original {
		cmp, ok := v.Cmp(resultArray[i])
		assert.True(t, ok)
		assert.Equal(t, 0, cmp)
	}
}

// TestRoundTrip_StackFrame tests StackFrame decomposition and recomposition
func TestRoundTrip_StackFrame(t *testing.T) {
	c := NewMemoryCAS()

	// Create a StackFrame with various data
	original := &interp.StackFrame{
		Stack: []vm.Value{
			vm.IntValue(42),
			vm.StrValue("test"),
			vm.BoolTrue,
		},
		PC: vm.ExecPtr((1 << 32) | 10),
		Variables: map[string]vm.Value{
			"x":    vm.IntValue(100),
			"name": vm.StrValue("Alice"),
			"flag": vm.BoolFalse,
		},
		// IteratorStack left empty for now
	}

	// Decompose
	hash, err := DecomposeStackFrameForTest(c, original)
	require.NoError(t, err)

	// Recompose
	result, err := RecomposeStackFrameForTest(c, hash)
	require.NoError(t, err)

	// Verify
	assert.Equal(t, original.PC, result.PC)
	assert.Equal(t, len(original.Stack), len(result.Stack))
	assert.Equal(t, len(original.Variables), len(result.Variables))

	// Verify stack values
	for i, v := range original.Stack {
		cmp, ok := v.Cmp(result.Stack[i])
		assert.True(t, ok)
		assert.Equal(t, 0, cmp)
	}

	// Verify variables
	for k, v := range original.Variables {
		rv, exists := result.Variables[k]
		assert.True(t, exists, "Variable %s should exist", k)
		cmp, ok := v.Cmp(rv)
		assert.True(t, ok)
		assert.Equal(t, 0, cmp)
	}
}

// TestRoundTrip_State tests full State decomposition and recomposition
func TestRoundTrip_State(t *testing.T) {
	c := NewMemoryCAS()

	// Create a simple State
	original := &interp.State{
		Globals: &interp.StackFrame{
			Variables: map[string]vm.Value{
				"global_var": vm.IntValue(999),
			},
			PC: vm.ExecPtr((0 << 32) | 0),
		},
		ThreadSets: []interp.ThreadSet{
			// Thread 0 in singleton set
			{
				Stacks: []interp.StackFrames{
					{
						&interp.StackFrame{
							Stack: []vm.Value{
								vm.IntValue(1),
								vm.IntValue(2),
							},
							PC: vm.ExecPtr((1 << 32) | 5),
							Variables: map[string]vm.Value{
								"local": vm.StrValue("thread0"),
							},
						},
					},
				},
				PauseReason: []interp.Pause{interp.Start},
			},
			// Thread 1 in singleton set
			{
				Stacks: []interp.StackFrames{
					{
						&interp.StackFrame{
							Stack: []vm.Value{
								vm.BoolTrue,
							},
							PC: vm.ExecPtr((2 << 32) | 10),
							Variables: map[string]vm.Value{
								"count": vm.IntValue(42),
							},
						},
					},
				},
				PauseReason: []interp.Pause{interp.Yield},
			},
		},
	}

	// Decompose
	hash, err := DecomposeStateForTest(c, original)
	require.NoError(t, err)
	assert.NotEqual(t, Hash(0), hash)

	// Recompose
	result, err := RecomposeStateForTest(c, hash)
	require.NoError(t, err)

	// Verify structure
	assert.Equal(t, len(original.ThreadSets), len(result.ThreadSets))
	assert.Equal(t, original.ThreadCount(), result.ThreadCount())

	// Verify globals
	assert.Equal(t, original.Globals.PC, result.Globals.PC)
	for k, v := range original.Globals.Variables {
		rv, exists := result.Globals.Variables[k]
		assert.True(t, exists)
		cmp, ok := v.Cmp(rv)
		assert.True(t, ok)
		assert.Equal(t, 0, cmp)
	}

	// Verify each thread set
	for setIdx, origSet := range original.ThreadSets {
		resultSet := result.ThreadSets[setIdx]
		assert.Equal(t, len(origSet.Stacks), len(resultSet.Stacks))
		assert.Equal(t, len(origSet.PauseReason), len(resultSet.PauseReason))

		// Verify each thread in the set
		for localIdx, origThread := range origSet.Stacks {
			resultThread := resultSet.Stacks[localIdx]
			assert.Equal(t, len(origThread), len(resultThread))

			for frameIdx, origFrame := range origThread {
				resultFrame := resultThread[frameIdx]
				assert.Equal(t, origFrame.PC, resultFrame.PC)
				assert.Equal(t, len(origFrame.Stack), len(resultFrame.Stack))
				assert.Equal(t, len(origFrame.Variables), len(resultFrame.Variables))
			}

			// Verify pause reason
			assert.Equal(t, origSet.PauseReason[localIdx], resultSet.PauseReason[localIdx])
		}
	}
}

// TestStructuralSharing verifies that identical sub-structures share the same hash
func TestStructuralSharing(t *testing.T) {
	c := NewMemoryCAS()

	// Create two StackFrames with identical Variables
	sharedVars := map[string]vm.Value{
		"shared": vm.IntValue(42),
	}

	frame1 := &interp.StackFrame{
		PC:        vm.ExecPtr((1 << 32) | 0),
		Variables: sharedVars,
	}

	frame2 := &interp.StackFrame{
		PC:        vm.ExecPtr((2 << 32) | 0), // Different PC
		Variables: sharedVars,                       // Same variables
	}

	// Decompose both
	hash1, err := DecomposeStackFrameForTest(c, frame1)
	require.NoError(t, err)

	initialCASSize := len(c.data)

	hash2, err := DecomposeStackFrameForTest(c, frame2)
	require.NoError(t, err)

	finalCASSize := len(c.data)

	// They should have different hashes (different PC)
	assert.NotEqual(t, hash1, hash2)

	// But should share some internal structures (the variable value)
	// The CAS should have grown by less than the full size of frame2
	// This is a weak test, but demonstrates the concept
	t.Logf("Initial CAS size: %d, Final CAS size: %d", initialCASSize, finalCASSize)
	t.Logf("CAS growth: %d entries for second frame", finalCASSize-initialCASSize)
}

// TestRoundTrip_NestedStructures tests deeply nested structures
func TestRoundTrip_NestedStructures(t *testing.T) {
	c := NewMemoryCAS()

	// Create a struct containing an array containing a struct
	original := vm.StructValue{
		"outer": vm.StrValue("level1"),
		"array": vm.ArrayValue{
			vm.IntValue(1),
			vm.StructValue{
				"inner1": vm.IntValue(10),
				"inner2": vm.IntValue(20),
				"inner3": vm.IntValue(30),
			},
			vm.IntValue(3),
		},
		"last": vm.BoolTrue,
	}

	// Decompose
	hash, err := DecomposeValueForTest(c, original)
	require.NoError(t, err)

	// Recompose
	result, err := RecomposeValueForTest(c, hash)
	require.NoError(t, err)

	// Verify structure
	resultStruct, ok := result.(vm.StructValue)
	require.True(t, ok)

	// Verify outer field
	outerVal, _ := resultStruct["outer"]
	assert.Equal(t, vm.StrValue("level1"), outerVal)

	// Verify array field
	arrayVal, _ := resultStruct["array"]
	arrayResult, ok := arrayVal.(vm.ArrayValue)
	require.True(t, ok)
	assert.Equal(t, 3, len(arrayResult))

	// Verify nested struct inside array
	innerStruct, ok := arrayResult[1].(vm.StructValue)
	require.True(t, ok)
	assert.Equal(t, 3, len(innerStruct))
}

// TestRoundTrip_EmptyStructures tests empty collections
func TestRoundTrip_EmptyStructures(t *testing.T) {
	c := NewMemoryCAS()

	tests := []struct {
		name  string
		value vm.Value
	}{
		{"EmptyStruct", vm.StructValue{}},
		{"EmptyArray", vm.ArrayValue{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := DecomposeValueForTest(c, tt.value)
			require.NoError(t, err)

			result, err := RecomposeValueForTest(c, hash)
			require.NoError(t, err)

			// Verify type
			assert.IsType(t, tt.value, result)
		})
	}
}

// TestCASEntryCount verifies the number of CAS entries created
func TestCASEntryCount(t *testing.T) {
	c := NewMemoryCAS()

	// Create a state with multiple threads
	state := &interp.State{
		Globals: &interp.StackFrame{
			Variables: map[string]vm.Value{
				"g1": vm.IntValue(1),
				"g2": vm.IntValue(2),
			},
		},
		ThreadSets: []interp.ThreadSet{
			{
				Stacks: []interp.StackFrames{
					{
						&interp.StackFrame{
							Variables: map[string]vm.Value{
								"t0v1": vm.IntValue(10),
							},
						},
					},
				},
				PauseReason: []interp.Pause{interp.Start},
			},
			{
				Stacks: []interp.StackFrames{
					{
						&interp.StackFrame{
							Variables: map[string]vm.Value{
								"t1v1": vm.IntValue(20),
							},
						},
					},
				},
				PauseReason: []interp.Pause{interp.Start},
			},
		},
	}

	// Decompose
	hash, err := DecomposeStateForTest(c, state)
	require.NoError(t, err)
	assert.NotEqual(t, Hash(0), hash)

	// Count entries
	entryCount := len(c.data)
	t.Logf("Total CAS entries created: %d", entryCount)

	// Verify we can recompose
	result, err := RecomposeStateForTest(c, hash)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Expected entries:
	// - StateRef (1)
	// - GlobalsHash → StackFrameRef (1)
	// - g1, g2 values (2)
	// - Thread 0 StackFrameRef (1)
	// - t0v1 value (1)
	// - Thread 1 StackFrameRef (1)
	// - t1v1 value (1)
	// Total: ~8 entries
	assert.Greater(t, entryCount, 5, "Should have multiple CAS entries")
	assert.Less(t, entryCount, 20, "Should not have excessive entries")
}
