package test

import (
	"testing"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// TestSimpleWhileLoop tests a basic while loop that modifies a global variable
// Converted from: testdata/test_simple.star
func TestSimpleWhileLoop(t *testing.T) {
	code := `
x = True

def foo():
    while x:
        x = False
`
	prog, err := vm.CompileLiteral(code)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	state := interp.NewState()
	frame := state.Globals

	_, err = interp.RunToEnd(prog, nil, frame)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Call the foo function
	fooPtr, ok := prog.Resolve("foo")
	if !ok {
		t.Fatalf("Function 'foo' not found")
	}

	fooFn := vm.FnPtrValue(fooPtr)

	// Execute the function
	callFrame := &interp.StackFrame{
		PC:    vm.ExecPtr(fooFn),
		Stack: []vm.Value{},
	}

	_, err = interp.RunToEnd(prog, frame, callFrame)
	if err != nil {
		t.Fatalf("Function execution failed: %v", err)
	}

	// Verify x is now False
	x, ok := frame.Variables["x"]
	if !ok {
		t.Fatalf("Variable 'x' not found")
	}

	if x != vm.BoolFalse {
		t.Errorf("Expected x to be False, got %v", x)
	}
}

// TestArrayAppend tests the append method on arrays
// Converted from: testdata/test_append.star
func TestArrayAppend(t *testing.T) {
	code := `
queue = []

def test():
    queue.append("msg")
`
	prog, err := vm.CompileLiteral(code)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	state := interp.NewState()
	frame := state.Globals

	_, err = interp.RunToEnd(prog, nil, frame)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Call the test function
	testPtr, ok := prog.Resolve("test")
	if !ok {
		t.Fatalf("Function 'test' not found")
	}

	testFn := vm.FnPtrValue(testPtr)

	// Execute the function
	callFrame := &interp.StackFrame{
		PC:    vm.ExecPtr(testFn),
		Stack: []vm.Value{},
	}

	_, err = interp.RunToEnd(prog, frame, callFrame)
	if err != nil {
		t.Fatalf("Function execution failed: %v", err)
	}

	// Verify queue has one element
	queue, ok := frame.Variables["queue"]
	if !ok {
		t.Fatalf("Variable 'queue' not found")
	}

	queueArr, ok := queue.(vm.ArrayValue)
	if !ok {
		t.Fatalf("'queue' is not an array, got %T", queue)
	}

	if len(queueArr) != 1 {
		t.Errorf("Expected queue length 1, got %d", len(queueArr))
	}

	if len(queueArr) > 0 {
		msgStr, ok := queueArr[0].(vm.StrValue)
		if !ok {
			t.Errorf("Expected first element to be string, got %T", queueArr[0])
		} else if string(msgStr) != "msg" {
			t.Errorf("Expected 'msg', got %q", string(msgStr))
		}
	}
}

// TestFunctionCalls tests calling user-defined functions
// Converted from: testdata/test_funcall.star
func TestFunctionCalls(t *testing.T) {
	code := `
counter = 0

def increment():
    counter = counter + 1

def test():
    increment()
    increment()
`
	prog, err := vm.CompileLiteral(code)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	state := interp.NewState()
	frame := state.Globals

	_, err = interp.RunToEnd(prog, nil, frame)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Call the test function
	testPtr, ok := prog.Resolve("test")
	if !ok {
		t.Fatalf("Function 'test' not found")
	}

	testFn := vm.FnPtrValue(testPtr)

	// Execute the function
	callFrame := &interp.StackFrame{
		PC:    vm.ExecPtr(testFn),
		Stack: []vm.Value{},
	}

	_, err = interp.RunToEnd(prog, frame, callFrame)
	if err != nil {
		t.Fatalf("Function execution failed: %v", err)
	}

	// Verify counter is 2
	counter, ok := frame.Variables["counter"]
	if !ok {
		t.Fatalf("Variable 'counter' not found")
	}

	counterInt, ok := counter.(vm.IntValue)
	if !ok {
		t.Fatalf("'counter' is not an int, got %T", counter)
	}

	if int(counterInt) != 2 {
		t.Errorf("Expected counter to be 2, got %d", int(counterInt))
	}
}

// TestWhileLoopCounter tests a while loop with counter
// Converted from: testdata/test_while.star
func TestWhileLoopCounter(t *testing.T) {
	code := `
counter = 0

def test():
    counter = 0
    while counter < 3:
        counter = counter + 1
`
	prog, err := vm.CompileLiteral(code)
	if err != nil {
		t.Fatalf("Compilation failed: %v", err)
	}

	state := interp.NewState()
	frame := state.Globals

	_, err = interp.RunToEnd(prog, nil, frame)
	if err != nil {
		t.Fatalf("Execution failed: %v", err)
	}

	// Call the test function
	testPtr, ok := prog.Resolve("test")
	if !ok {
		t.Fatalf("Function 'test' not found")
	}

	testFn := vm.FnPtrValue(testPtr)

	// Execute the function
	callFrame := &interp.StackFrame{
		PC:    vm.ExecPtr(testFn),
		Stack: []vm.Value{},
	}

	_, err = interp.RunToEnd(prog, frame, callFrame)
	if err != nil {
		t.Fatalf("Function execution failed: %v", err)
	}

	// Verify counter is 3
	counter, ok := frame.Variables["counter"]
	if !ok {
		t.Fatalf("Variable 'counter' not found")
	}

	counterInt, ok := counter.(vm.IntValue)
	if !ok {
		t.Fatalf("'counter' is not an int, got %T", counter)
	}

	if int(counterInt) != 3 {
		t.Errorf("Expected counter to be 3, got %d", int(counterInt))
	}
}

// TestBasicArithmetic tests basic arithmetic operations
func TestBasicArithmetic(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		varName  string
		expected vm.Value
	}{
		{
			name: "addition",
			code: `
result = 2 + 3
`,
			varName:  "result",
			expected: vm.IntValue(5),
		},
		{
			name: "subtraction",
			code: `
result = 10 - 4
`,
			varName:  "result",
			expected: vm.IntValue(6),
		},
		{
			name: "multiplication",
			code: `
result = 3 * 4
`,
			varName:  "result",
			expected: vm.IntValue(12),
		},
		{
			name: "division",
			code: `
result = 15 / 3
`,
			varName:  "result",
			expected: vm.IntValue(5),
		},
		{
			name: "complex expression",
			code: `
result = (2 + 3) * 4 - 1
`,
			varName:  "result",
			expected: vm.IntValue(19),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := vm.CompileLiteral(tt.code)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			state := interp.NewState()
			frame := state.Globals

			_, err = interp.RunToEnd(prog, nil, frame)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			result, ok := frame.Variables[tt.varName]
			if !ok {
				t.Fatalf("Variable '%s' not found", tt.varName)
			}

			if cmp, ok := result.Cmp(tt.expected); !ok || cmp != 0 {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestBooleanLogic tests boolean operations
func TestBooleanLogic(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "and true",
			code: `
result = True and True
`,
			expected: vm.BoolTrue,
		},
		{
			name: "and false",
			code: `
result = True and False
`,
			expected: vm.BoolFalse,
		},
		{
			name: "or true",
			code: `
result = True or False
`,
			expected: vm.BoolTrue,
		},
		{
			name: "or false",
			code: `
result = False or False
`,
			expected: vm.BoolFalse,
		},
		{
			name: "not true",
			code: `
result = not True
`,
			expected: vm.BoolFalse,
		},
		{
			name: "not false",
			code: `
result = not False
`,
			expected: vm.BoolTrue,
		},
		{
			name: "complex expression",
			code: `
result = (True or False) and not False
`,
			expected: vm.BoolTrue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := vm.CompileLiteral(tt.code)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			state := interp.NewState()
			frame := state.Globals

			_, err = interp.RunToEnd(prog, nil, frame)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			result, ok := frame.Variables["result"]
			if !ok {
				t.Fatalf("Variable 'result' not found")
			}

			if cmp, ok := result.Cmp(tt.expected); !ok || cmp != 0 {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// TestComparisons tests comparison operators
func TestComparisons(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "less than true",
			code: `
result = 3 < 5
`,
			expected: vm.BoolTrue,
		},
		{
			name: "less than false",
			code: `
result = 5 < 3
`,
			expected: vm.BoolFalse,
		},
		{
			name: "greater than true",
			code: `
result = 5 > 3
`,
			expected: vm.BoolTrue,
		},
		{
			name: "equal true",
			code: `
result = 5 == 5
`,
			expected: vm.BoolTrue,
		},
		{
			name: "equal false",
			code: `
result = 5 == 3
`,
			expected: vm.BoolFalse,
		},
		{
			name: "not equal true",
			code: `
result = 5 != 3
`,
			expected: vm.BoolTrue,
		},
		{
			name: "less than or equal",
			code: `
result = 5 <= 5
`,
			expected: vm.BoolTrue,
		},
		{
			name: "greater than or equal",
			code: `
result = 6 >= 5
`,
			expected: vm.BoolTrue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prog, err := vm.CompileLiteral(tt.code)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			state := interp.NewState()
			frame := state.Globals

			_, err = interp.RunToEnd(prog, nil, frame)
			if err != nil {
				t.Fatalf("Execution failed: %v", err)
			}

			result, ok := frame.Variables["result"]
			if !ok {
				t.Fatalf("Variable 'result' not found")
			}

			if cmp, ok := result.Cmp(tt.expected); !ok || cmp != 0 {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
