package interp

import (
	"testing"

	"github.com/timewinder-dev/timewinder/vm"
)

func TestSliceIteratorSingleVar(t *testing.T) {
	values := []vm.Value{
		vm.IntValue(1),
		vm.IntValue(2),
		vm.IntValue(3),
	}

	iter := &SliceIterator{
		Values:   values,
		Index:    -1,
		VarCount: 1,
	}

	// Test initial state
	if iter.Index != -1 {
		t.Errorf("Expected initial index -1, got %d", iter.Index)
	}

	// First iteration
	if !iter.Next() {
		t.Fatal("Expected first Next() to return true")
	}
	if iter.Index != 0 {
		t.Errorf("Expected index 0, got %d", iter.Index)
	}
	if iter.Var1() != vm.IntValue(1) {
		t.Errorf("Expected Var1() = 1, got %v", iter.Var1())
	}

	// Second iteration
	if !iter.Next() {
		t.Fatal("Expected second Next() to return true")
	}
	if iter.Index != 1 {
		t.Errorf("Expected index 1, got %d", iter.Index)
	}
	if iter.Var1() != vm.IntValue(2) {
		t.Errorf("Expected Var1() = 2, got %v", iter.Var1())
	}

	// Third iteration
	if !iter.Next() {
		t.Fatal("Expected third Next() to return true")
	}
	if iter.Index != 2 {
		t.Errorf("Expected index 2, got %d", iter.Index)
	}
	if iter.Var1() != vm.IntValue(3) {
		t.Errorf("Expected Var1() = 3, got %v", iter.Var1())
	}

	// Should be exhausted
	if iter.Next() {
		t.Error("Expected Next() to return false after exhausting iterator")
	}
}

func TestSliceIteratorTwoVars(t *testing.T) {
	values := []vm.Value{
		vm.StrValue("a"),
		vm.StrValue("b"),
	}

	iter := &SliceIterator{
		Values:   values,
		Index:    -1,
		VarCount: 2,
	}

	// First iteration
	if !iter.Next() {
		t.Fatal("Expected first Next() to return true")
	}
	if iter.Var1() != vm.IntValue(0) {
		t.Errorf("Expected Var1() = 0 (index), got %v", iter.Var1())
	}
	if iter.Var2() != vm.StrValue("a") {
		t.Errorf("Expected Var2() = 'a' (element), got %v", iter.Var2())
	}

	// Second iteration
	if !iter.Next() {
		t.Fatal("Expected second Next() to return true")
	}
	if iter.Var1() != vm.IntValue(1) {
		t.Errorf("Expected Var1() = 1 (index), got %v", iter.Var1())
	}
	if iter.Var2() != vm.StrValue("b") {
		t.Errorf("Expected Var2() = 'b' (element), got %v", iter.Var2())
	}

	// Should be exhausted
	if iter.Next() {
		t.Error("Expected Next() to return false after exhausting iterator")
	}
}

func TestSliceIteratorEmpty(t *testing.T) {
	iter := &SliceIterator{
		Values:   []vm.Value{},
		Index:    -1,
		VarCount: 1,
	}

	// Should immediately return false
	if iter.Next() {
		t.Error("Expected Next() to return false for empty slice")
	}
}

func TestDictIteratorSingleVar(t *testing.T) {
	dict := vm.StructValue{
		"charlie": vm.IntValue(15),
		"alice":   vm.IntValue(10),
		"bob":     vm.IntValue(20),
	}

	keys := []string{"alice", "bob", "charlie"} // Sorted

	iter := &DictIterator{
		Dict:     dict,
		Keys:     keys,
		Index:    -1,
		VarCount: 1,
	}

	// First iteration - should get "alice"
	if !iter.Next() {
		t.Fatal("Expected first Next() to return true")
	}
	if iter.Var1() != vm.StrValue("alice") {
		t.Errorf("Expected Var1() = 'alice', got %v", iter.Var1())
	}

	// Second iteration - should get "bob"
	if !iter.Next() {
		t.Fatal("Expected second Next() to return true")
	}
	if iter.Var1() != vm.StrValue("bob") {
		t.Errorf("Expected Var1() = 'bob', got %v", iter.Var1())
	}

	// Third iteration - should get "charlie"
	if !iter.Next() {
		t.Fatal("Expected third Next() to return true")
	}
	if iter.Var1() != vm.StrValue("charlie") {
		t.Errorf("Expected Var1() = 'charlie', got %v", iter.Var1())
	}

	// Should be exhausted
	if iter.Next() {
		t.Error("Expected Next() to return false after exhausting iterator")
	}
}

func TestDictIteratorTwoVars(t *testing.T) {
	dict := vm.StructValue{
		"bob":   vm.IntValue(20),
		"alice": vm.IntValue(10),
	}

	keys := []string{"alice", "bob"} // Sorted

	iter := &DictIterator{
		Dict:     dict,
		Keys:     keys,
		Index:    -1,
		VarCount: 2,
	}

	// First iteration
	if !iter.Next() {
		t.Fatal("Expected first Next() to return true")
	}
	if iter.Var1() != vm.StrValue("alice") {
		t.Errorf("Expected Var1() = 'alice', got %v", iter.Var1())
	}
	if iter.Var2() != vm.IntValue(10) {
		t.Errorf("Expected Var2() = 10, got %v", iter.Var2())
	}

	// Second iteration
	if !iter.Next() {
		t.Fatal("Expected second Next() to return true")
	}
	if iter.Var1() != vm.StrValue("bob") {
		t.Errorf("Expected Var1() = 'bob', got %v", iter.Var1())
	}
	if iter.Var2() != vm.IntValue(20) {
		t.Errorf("Expected Var2() = 20, got %v", iter.Var2())
	}

	// Should be exhausted
	if iter.Next() {
		t.Error("Expected Next() to return false after exhausting iterator")
	}
}

func TestDictIteratorEmpty(t *testing.T) {
	iter := &DictIterator{
		Dict:     vm.StructValue{},
		Keys:     []string{},
		Index:    -1,
		VarCount: 2,
	}

	// Should immediately return false
	if iter.Next() {
		t.Error("Expected Next() to return false for empty dict")
	}
}

func TestSliceIteratorClone(t *testing.T) {
	values := []vm.Value{
		vm.IntValue(1),
		vm.IntValue(2),
	}

	iter := &SliceIterator{
		Values:   values,
		Index:    -1,
		VarCount: 1,
	}

	// Advance iterator
	iter.Next()

	// Clone it
	cloned := iter.Clone().(*SliceIterator)

	// Cloned should have same state
	if cloned.Index != iter.Index {
		t.Errorf("Expected cloned index %d, got %d", iter.Index, cloned.Index)
	}
	if cloned.VarCount != iter.VarCount {
		t.Errorf("Expected cloned VarCount %d, got %d", iter.VarCount, cloned.VarCount)
	}
	if len(cloned.Values) != len(iter.Values) {
		t.Errorf("Expected cloned Values length %d, got %d", len(iter.Values), len(cloned.Values))
	}

	// Advance original
	iter.Next()

	// Cloned should not be affected
	if cloned.Index != 0 {
		t.Errorf("Expected cloned index still 0, got %d", cloned.Index)
	}
}

func TestDictIteratorClone(t *testing.T) {
	dict := vm.StructValue{
		"alice": vm.IntValue(10),
		"bob":   vm.IntValue(20),
	}

	keys := []string{"alice", "bob"}

	iter := &DictIterator{
		Dict:     dict,
		Keys:     keys,
		Index:    -1,
		VarCount: 2,
	}

	// Advance iterator
	iter.Next()

	// Clone it
	cloned := iter.Clone().(*DictIterator)

	// Cloned should have same state
	if cloned.Index != iter.Index {
		t.Errorf("Expected cloned index %d, got %d", iter.Index, cloned.Index)
	}
	if cloned.VarCount != iter.VarCount {
		t.Errorf("Expected cloned VarCount %d, got %d", iter.VarCount, cloned.VarCount)
	}

	// Advance original
	iter.Next()

	// Cloned should not be affected
	if cloned.Index != 0 {
		t.Errorf("Expected cloned index still 0, got %d", cloned.Index)
	}
}

func TestIterateNonIterableError(t *testing.T) {
	// Test that attempting to iterate over non-iterable types causes an error
	testCases := []struct {
		name  string
		value vm.Value
	}{
		{"int", vm.IntValue(42)},
		{"string", vm.StrValue("hello")},
		{"bool", vm.BoolValue(true)},
		{"float", vm.FloatValue(3.14)},
		{"none", vm.None},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a minimal program with an ITER_START instruction
			prog := &vm.Program{
				Main: &vm.Function{
					Bytecode: []vm.Op{
						{Code: vm.ITER_START, Arg: vm.IntValue(3)}, // Jump to offset 3 when done
						{Code: vm.PUSH, Arg: vm.IntValue(999)},     // Loop body (not reached)
						{Code: vm.ITER_NEXT},
						// offset 3: end of loop
					},
				},
			}

			// Create state with the non-iterable value and variable name on stack
			globals := &StackFrame{
				Variables: make(map[string]vm.Value),
			}

			frame := &StackFrame{
				PC:        vm.ExecPtr(0),
				Variables: make(map[string]vm.Value),
				Stack:     []vm.Value{},
			}

			// Push variable name and iterable onto stack (as ITER_START expects)
			frame.Push(vm.StrValue("x"))
			frame.Push(tc.value)

			// Execute the ITER_START instruction
			stepType, _, err := Step(prog, globals, []*StackFrame{frame})

			// Should get an error
			if err == nil {
				t.Errorf("Expected error when iterating over %s, got nil", tc.name)
			}

			if stepType != ErrorStep {
				t.Errorf("Expected ErrorStep, got %v", stepType)
			}

			// Error message should mention the type
			if err != nil {
				expectedMsg := "Cannot iterate over"
				if len(err.Error()) < len(expectedMsg) || err.Error()[:len(expectedMsg)] != expectedMsg {
					t.Errorf("Expected error message to start with '%s', got: %v", expectedMsg, err.Error())
				}
			}
		})
	}
}

func TestIterateNonIterableTwoVarsError(t *testing.T) {
	// Test ITER_START_2 with non-iterable
	prog := &vm.Program{
		Main: &vm.Function{
			Bytecode: []vm.Op{
				{Code: vm.ITER_START_2, Arg: vm.IntValue(3)},
				{Code: vm.PUSH, Arg: vm.IntValue(999)},
				{Code: vm.ITER_NEXT},
			},
		},
	}

	globals := &StackFrame{
		Variables: make(map[string]vm.Value),
	}

	frame := &StackFrame{
		PC:        vm.ExecPtr(0),
		Variables: make(map[string]vm.Value),
		Stack:     []vm.Value{},
	}

	// Push two variable names and non-iterable value
	frame.Push(vm.StrValue("key"))
	frame.Push(vm.StrValue("value"))
	frame.Push(vm.IntValue(123)) // Non-iterable

	// Execute ITER_START_2
	stepType, _, err := Step(prog, globals, []*StackFrame{frame})

	// Should get an error
	if err == nil {
		t.Error("Expected error when iterating over int with ITER_START_2, got nil")
	}

	if stepType != ErrorStep {
		t.Errorf("Expected ErrorStep, got %v", stepType)
	}

	if err != nil && len(err.Error()) > 0 {
		expectedMsg := "Cannot iterate over"
		if len(err.Error()) < len(expectedMsg) || err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Expected error message to start with '%s', got: %v", expectedMsg, err.Error())
		}
	}
}
