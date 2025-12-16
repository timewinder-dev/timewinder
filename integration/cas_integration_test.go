package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// TestCAS_SimpleProgram tests CAS with a simple Starlark program
func TestCAS_SimpleProgram(t *testing.T) {
	// Simple program: x = 5 + 3
	code := `
x = 5 + 3
y = x * 2
`

	// Compile
	prog, err := vm.CompileLiteral(code)
	require.NoError(t, err)

	// Execute to completion
	frame := &interp.StackFrame{}
	_, err = interp.RunToEnd(prog, nil, frame)
	require.NoError(t, err)

	// Verify execution result
	xVal, exists := frame.Variables["x"]
	require.True(t, exists)
	assert.Equal(t, vm.IntValue(8), xVal)

	yVal, exists := frame.Variables["y"]
	require.True(t, exists)
	assert.Equal(t, vm.IntValue(16), yVal)

	// Create a state from this frame
	state := &interp.State{
		Globals: frame,
	}

	// Test CAS round-trip
	c := cas.NewMemoryCAS()

	// Decompose state (using internal function for testing)
	hash, err := cas.DecomposeStateForTest(c, state)
	require.NoError(t, err)
	assert.NotEqual(t, cas.Hash(0), hash)

	// Recompose state
	result, err := cas.RecomposeStateForTest(c, hash)
	require.NoError(t, err)

	// Verify the recomposed state matches
	require.NotNil(t, result.Globals)
	resultX, exists := result.Globals.Variables["x"]
	require.True(t, exists)
	assert.Equal(t, vm.IntValue(8), resultX)

	resultY, exists := result.Globals.Variables["y"]
	require.True(t, exists)
	assert.Equal(t, vm.IntValue(16), resultY)
}

// TestCAS_FunctionCall tests CAS with function calls
func TestCAS_FunctionCall(t *testing.T) {
	code := `
def add(a, b):
    return a + b

result = add(10, 20)
`

	// Compile and execute
	prog, err := vm.CompileLiteral(code)
	require.NoError(t, err)

	frame := &interp.StackFrame{}
	_, err = interp.RunToEnd(prog, nil, frame)
	require.NoError(t, err)

	// Verify result
	resultVal, exists := frame.Variables["result"]
	require.True(t, exists)
	assert.Equal(t, vm.IntValue(30), resultVal)

	// Test CAS round-trip
	state := &interp.State{Globals: frame}
	c := cas.NewMemoryCAS()

	hash, err := cas.DecomposeStateForTest(c, state)
	require.NoError(t, err)

	recovered, err := cas.RecomposeStateForTest(c, hash)
	require.NoError(t, err)

	// Verify
	recoveredResult, exists := recovered.Globals.Variables["result"]
	require.True(t, exists)
	assert.Equal(t, vm.IntValue(30), recoveredResult)
}

// TestCAS_DataStructures tests CAS with dicts and lists
func TestCAS_DataStructures(t *testing.T) {
	code := `
account = {"name": "Alice", "balance": 100}
items = [1, 2, 3, 4, 5]
nested = {"list": [10, 20], "value": 99}
`

	prog, err := vm.CompileLiteral(code)
	require.NoError(t, err)

	frame := &interp.StackFrame{}
	_, err = interp.RunToEnd(prog, nil, frame)
	require.NoError(t, err)

	// Verify execution
	account, exists := frame.Variables["account"]
	require.True(t, exists)
	accountStruct, ok := account.(vm.StructValue)
	require.True(t, ok)
	assert.Equal(t, vm.StrValue("Alice"), accountStruct["name"])

	// Test CAS round-trip
	state := &interp.State{Globals: frame}
	c := cas.NewMemoryCAS()

	hash, err := cas.DecomposeStateForTest(c, state)
	require.NoError(t, err)

	recovered, err := cas.RecomposeStateForTest(c, hash)
	require.NoError(t, err)

	// Verify account
	recoveredAccount, exists := recovered.Globals.Variables["account"]
	require.True(t, exists)
	recoveredStruct, ok := recoveredAccount.(vm.StructValue)
	require.True(t, ok)
	assert.Equal(t, vm.StrValue("Alice"), recoveredStruct["name"])
	assert.Equal(t, vm.IntValue(100), recoveredStruct["balance"])

	// Verify items list
	recoveredItems, exists := recovered.Globals.Variables["items"]
	require.True(t, exists)
	recoveredArray, ok := recoveredItems.(vm.ArrayValue)
	require.True(t, ok)
	assert.Equal(t, 5, len(recoveredArray))
	assert.Equal(t, vm.IntValue(1), recoveredArray[0])
	assert.Equal(t, vm.IntValue(5), recoveredArray[4])

	// Verify nested structure
	recoveredNested, exists := recovered.Globals.Variables["nested"]
	require.True(t, exists)
	nestedStruct, ok := recoveredNested.(vm.StructValue)
	require.True(t, ok)

	nestedList, ok := nestedStruct["list"].(vm.ArrayValue)
	require.True(t, ok)
	assert.Equal(t, 2, len(nestedList))
	assert.Equal(t, vm.IntValue(10), nestedList[0])
}

// TestCAS_MultipleStates tests storing multiple states and verifying independence
func TestCAS_MultipleStates(t *testing.T) {
	c := cas.NewMemoryCAS()

	// Create first state
	state1 := &interp.State{
		Globals: &interp.StackFrame{
			Variables: map[string]vm.Value{
				"version": vm.IntValue(1),
				"data":    vm.StrValue("first"),
			},
		},
	}

	// Create second state (different data)
	state2 := &interp.State{
		Globals: &interp.StackFrame{
			Variables: map[string]vm.Value{
				"version": vm.IntValue(2),
				"data":    vm.StrValue("second"),
			},
		},
	}

	// Store both in CAS
	hash1, err := cas.DecomposeStateForTest(c, state1)
	require.NoError(t, err)

	hash2, err := cas.DecomposeStateForTest(c, state2)
	require.NoError(t, err)

	// Hashes should be different
	assert.NotEqual(t, hash1, hash2)

	// Retrieve both
	recovered1, err := cas.RecomposeStateForTest(c, hash1)
	require.NoError(t, err)

	recovered2, err := cas.RecomposeStateForTest(c, hash2)
	require.NoError(t, err)

	// Verify they're independent
	ver1 := recovered1.Globals.Variables["version"]
	ver2 := recovered2.Globals.Variables["version"]
	assert.Equal(t, vm.IntValue(1), ver1)
	assert.Equal(t, vm.IntValue(2), ver2)

	data1 := recovered1.Globals.Variables["data"]
	data2 := recovered2.Globals.Variables["data"]
	assert.Equal(t, vm.StrValue("first"), data1)
	assert.Equal(t, vm.StrValue("second"), data2)
}

// TestCAS_IdenticalStates tests that identical states produce the same hash
func TestCAS_IdenticalStates(t *testing.T) {
	c := cas.NewMemoryCAS()

	// Create two identical states
	createState := func() *interp.State {
		return &interp.State{
			Globals: &interp.StackFrame{
				Variables: map[string]vm.Value{
					"x": vm.IntValue(42),
					"y": vm.StrValue("test"),
				},
			},
		}
	}

	state1 := createState()
	state2 := createState()

	// Store both
	hash1, err := cas.DecomposeStateForTest(c, state1)
	require.NoError(t, err)

	hash2, err := cas.DecomposeStateForTest(c, state2)
	require.NoError(t, err)

	// They should produce the same hash (structural equality)
	assert.Equal(t, hash1, hash2, "Identical states should hash to the same value")
}

// TestCAS_StateWithThreads tests multi-threaded state
func TestCAS_StateWithThreads(t *testing.T) {
	c := cas.NewMemoryCAS()

	// Create a state with multiple threads
	state := &interp.State{
		Globals: &interp.StackFrame{
			Variables: map[string]vm.Value{
				"shared": vm.IntValue(0),
			},
		},
		Stacks: []interp.StackFrames{
			// Thread 0
			{
				&interp.StackFrame{
					PC: vm.ExecPtr((1 << 32) | 10),
					Variables: map[string]vm.Value{
						"local0": vm.StrValue("thread0"),
					},
					Stack: []vm.Value{
						vm.IntValue(1),
						vm.IntValue(2),
					},
				},
			},
			// Thread 1
			{
				&interp.StackFrame{
					PC: vm.ExecPtr((2 << 32) | 20),
					Variables: map[string]vm.Value{
						"local1": vm.StrValue("thread1"),
					},
					Stack: []vm.Value{
						vm.BoolTrue,
					},
				},
			},
		},
		PauseReason: []interp.Pause{interp.Yield, interp.Start},
	}

	// Test round-trip
	hash, err := cas.DecomposeStateForTest(c, state)
	require.NoError(t, err)

	recovered, err := cas.RecomposeStateForTest(c, hash)
	require.NoError(t, err)

	// Verify structure
	assert.Equal(t, 2, len(recovered.Stacks))
	assert.Equal(t, 2, len(recovered.PauseReason))

	// Verify thread 0
	thread0 := recovered.Stacks[0][0]
	assert.Equal(t, vm.ExecPtr((1 << 32) | 10), thread0.PC)
	assert.Equal(t, vm.StrValue("thread0"), thread0.Variables["local0"])
	assert.Equal(t, 2, len(thread0.Stack))

	// Verify thread 1
	thread1 := recovered.Stacks[1][0]
	assert.Equal(t, vm.ExecPtr((2 << 32) | 20), thread1.PC)
	assert.Equal(t, vm.StrValue("thread1"), thread1.Variables["local1"])
	assert.Equal(t, 1, len(thread1.Stack))

	// Verify pause reasons
	assert.Equal(t, interp.Yield, recovered.PauseReason[0])
	assert.Equal(t, interp.Start, recovered.PauseReason[1])
}

// TestCAS_StateEvolution tests storing states as program evolves
func TestCAS_StateEvolution(t *testing.T) {
	code := `
counter = 0
`

	prog, err := vm.CompileLiteral(code)
	require.NoError(t, err)

	// Execute and create initial state
	frame := &interp.StackFrame{}
	_, err = interp.RunToEnd(prog, nil, frame)
	require.NoError(t, err)

	c := cas.NewMemoryCAS()
	var hashes []cas.Hash

	// Simulate state evolution
	for i := 0; i < 5; i++ {
		// Modify counter
		frame.Variables["counter"] = vm.IntValue(i)

		state := &interp.State{Globals: frame.Clone()}

		// Store in CAS
		hash, err := cas.DecomposeStateForTest(c, state)
		require.NoError(t, err)
		hashes = append(hashes, hash)
	}

	// All hashes should be different
	for i := 0; i < len(hashes); i++ {
		for j := i + 1; j < len(hashes); j++ {
			assert.NotEqual(t, hashes[i], hashes[j], "States %d and %d should have different hashes", i, j)
		}
	}

	// Retrieve and verify each state
	for i, hash := range hashes {
		recovered, err := cas.RecomposeStateForTest(c, hash)
		require.NoError(t, err)

		counter := recovered.Globals.Variables["counter"]
		assert.Equal(t, vm.IntValue(i), counter, "State %d should have counter=%d", i, i)
	}
}

// TestCAS_LargeState tests CAS with a larger, more complex state
func TestCAS_LargeState(t *testing.T) {
	c := cas.NewMemoryCAS()

	// Create a large struct
	largeStruct := vm.StructValue{}
	for i := 0; i < 100; i++ {
		key := "field" + string(rune('0'+i%10)) + string(rune('0'+i/10))
		largeStruct[key] = vm.IntValue(i)
	}

	// Create state
	state := &interp.State{
		Globals: &interp.StackFrame{
			Variables: map[string]vm.Value{
				"large": largeStruct,
			},
		},
	}

	// Test round-trip
	hash, err := cas.DecomposeStateForTest(c, state)
	require.NoError(t, err)

	recovered, err := cas.RecomposeStateForTest(c, hash)
	require.NoError(t, err)

	// Verify
	recoveredLarge, exists := recovered.Globals.Variables["large"]
	require.True(t, exists)
	recoveredStruct, ok := recoveredLarge.(vm.StructValue)
	require.True(t, ok)
	assert.Equal(t, 100, len(recoveredStruct))

	// Spot check a few values
	assert.Equal(t, vm.IntValue(0), recoveredStruct["field00"])
	assert.Equal(t, vm.IntValue(42), recoveredStruct["field24"])
	assert.Equal(t, vm.IntValue(99), recoveredStruct["field99"])
}
