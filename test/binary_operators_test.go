package test

import (
	"testing"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// TestModuloOperator tests the % (modulo) operator
func TestModuloOperator(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "positive modulo",
			code: `
result = 10 % 3
`,
			expected: vm.IntValue(1),
		},
		{
			name: "zero remainder",
			code: `
result = 8 % 4
`,
			expected: vm.IntValue(0),
		},
		{
			name: "small mod large",
			code: `
result = 3 % 10
`,
			expected: vm.IntValue(3),
		},
		{
			name: "modulo in expression",
			code: `
result = (7 + 3) % 4
`,
			expected: vm.IntValue(2),
		},
		{
			name: "negative modulo (Go semantics)",
			code: `
result = -7 % 3
`,
			expected: vm.IntValue(-1), // Go behavior: -7 % 3 = -1 (not 2 like Python)
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

// TestFloorDivisionOperator tests the // (floor division) operator
func TestFloorDivisionOperator(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "basic floor division",
			code: `
result = 10 // 3
`,
			expected: vm.IntValue(3),
		},
		{
			name: "exact division",
			code: `
result = 9 // 3
`,
			expected: vm.IntValue(3),
		},
		{
			name: "floor division with remainder",
			code: `
result = 7 // 2
`,
			expected: vm.IntValue(3),
		},
		{
			name: "floor division zero",
			code: `
result = 0 // 5
`,
			expected: vm.IntValue(0),
		},
		{
			name: "floor division in expression",
			code: `
result = (15 + 5) // 4
`,
			expected: vm.IntValue(5),
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

// TestInOperatorArrays tests the 'in' operator with arrays
func TestInOperatorArrays(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "element in array - found",
			code: `
arr = [1, 2, 3, 4, 5]
result = 3 in arr
`,
			expected: vm.BoolTrue,
		},
		{
			name: "element in array - not found",
			code: `
arr = [1, 2, 3, 4, 5]
result = 10 in arr
`,
			expected: vm.BoolFalse,
		},
		{
			name: "element in array - first position",
			code: `
arr = [1, 2, 3]
result = 1 in arr
`,
			expected: vm.BoolTrue,
		},
		{
			name: "element in array - last position",
			code: `
arr = [1, 2, 3]
result = 3 in arr
`,
			expected: vm.BoolTrue,
		},
		{
			name: "element in empty array",
			code: `
arr = []
result = 1 in arr
`,
			expected: vm.BoolFalse,
		},
		{
			name: "string in string array",
			code: `
arr = ["hello", "world"]
result = "hello" in arr
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

// TestInOperatorStrings tests the 'in' operator with strings (substring search)
func TestInOperatorStrings(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "substring found",
			code: `
text = "hello world"
result = "world" in text
`,
			expected: vm.BoolTrue,
		},
		{
			name: "substring not found",
			code: `
text = "hello world"
result = "xyz" in text
`,
			expected: vm.BoolFalse,
		},
		{
			name: "single character found",
			code: `
text = "hello"
result = "e" in text
`,
			expected: vm.BoolTrue,
		},
		{
			name: "empty substring",
			code: `
text = "hello"
result = "" in text
`,
			expected: vm.BoolTrue, // Empty string is in any string
		},
		{
			name: "substring at start",
			code: `
text = "hello world"
result = "hello" in text
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

// TestInOperatorDicts tests the 'in' operator with dictionaries (key lookup)
func TestInOperatorDicts(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "key in dict - found",
			code: `
d = {"a": 1, "b": 2, "c": 3}
result = "a" in d
`,
			expected: vm.BoolTrue,
		},
		{
			name: "key in dict - not found",
			code: `
d = {"a": 1, "b": 2, "c": 3}
result = "z" in d
`,
			expected: vm.BoolFalse,
		},
		{
			name: "key in empty dict",
			code: `
d = {}
result = "a" in d
`,
			expected: vm.BoolFalse,
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

// TestNotInOperator tests the 'not in' operator
func TestNotInOperator(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected vm.Value
	}{
		{
			name: "element not in array - true",
			code: `
arr = [1, 2, 3]
result = 5 not in arr
`,
			expected: vm.BoolTrue,
		},
		{
			name: "element not in array - false",
			code: `
arr = [1, 2, 3]
result = 2 not in arr
`,
			expected: vm.BoolFalse,
		},
		{
			name: "substring not in string - true",
			code: `
text = "hello world"
result = "xyz" not in text
`,
			expected: vm.BoolTrue,
		},
		{
			name: "substring not in string - false",
			code: `
text = "hello world"
result = "world" not in text
`,
			expected: vm.BoolFalse,
		},
		{
			name: "key not in dict - true",
			code: `
d = {"a": 1, "b": 2}
result = "z" not in d
`,
			expected: vm.BoolTrue,
		},
		{
			name: "key not in dict - false",
			code: `
d = {"a": 1, "b": 2}
result = "a" not in d
`,
			expected: vm.BoolFalse,
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

// TestModuloCircularIndex tests modulo for circular buffer use case
func TestModuloCircularIndex(t *testing.T) {
	code := `
# Simulate token ring - next process ID
N = 3
current = 0
next = (current + 1) % N
result = next
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

	result, ok := frame.Variables["result"]
	if !ok {
		t.Fatalf("Variable 'result' not found")
	}

	expected := vm.IntValue(1)
	if cmp, ok := result.Cmp(expected); !ok || cmp != 0 {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestInOperatorInControlFlow tests using 'in' in if statements
func TestInOperatorInControlFlow(t *testing.T) {
	code := `
arr = [1, 2, 3]
result = 0

if 2 in arr:
    result = 10
else:
    result = 20
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

	result, ok := frame.Variables["result"]
	if !ok {
		t.Fatalf("Variable 'result' not found")
	}

	expected := vm.IntValue(10)
	if cmp, ok := result.Cmp(expected); !ok || cmp != 0 {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
