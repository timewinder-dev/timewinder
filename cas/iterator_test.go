package cas

import (
	"testing"

	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

func TestDecomposeRecomposeSliceIterator(t *testing.T) {
	c := NewMemoryCAS()

	// Create a SliceIterator
	values := []vm.Value{
		vm.IntValue(1),
		vm.IntValue(2),
		vm.IntValue(3),
	}

	original := &interp.SliceIterator{
		Values:   values,
		Index:    1, // Mid-iteration
		VarCount: 1,
	}

	// Decompose it
	hash, err := decomposeIterator(c, original)
	if err != nil {
		t.Fatalf("Failed to decompose SliceIterator: %v", err)
	}

	// Recompose it
	recomposed, err := recomposeIterator(c, hash)
	if err != nil {
		t.Fatalf("Failed to recompose SliceIterator: %v", err)
	}

	// Verify it's the right type
	sliceIter, ok := recomposed.(*interp.SliceIterator)
	if !ok {
		t.Fatalf("Expected *interp.SliceIterator, got %T", recomposed)
	}

	// Verify state
	if sliceIter.Index != original.Index {
		t.Errorf("Expected Index %d, got %d", original.Index, sliceIter.Index)
	}
	if sliceIter.VarCount != original.VarCount {
		t.Errorf("Expected VarCount %d, got %d", original.VarCount, sliceIter.VarCount)
	}
	if len(sliceIter.Values) != len(original.Values) {
		t.Errorf("Expected %d values, got %d", len(original.Values), len(sliceIter.Values))
	}

	// Verify values
	for i, v := range original.Values {
		if sliceIter.Values[i] != v {
			t.Errorf("Value[%d]: expected %v, got %v", i, v, sliceIter.Values[i])
		}
	}
}

func TestDecomposeRecomposeDictIterator(t *testing.T) {
	c := NewMemoryCAS()

	// Create a DictIterator
	dict := vm.StructValue{
		"alice": vm.IntValue(10),
		"bob":   vm.IntValue(20),
	}

	keys := []string{"alice", "bob"}

	original := &interp.DictIterator{
		Dict:     dict,
		Keys:     keys,
		Index:    0, // At first key
		VarCount: 2,
	}

	// Decompose it
	hash, err := decomposeIterator(c, original)
	if err != nil {
		t.Fatalf("Failed to decompose DictIterator: %v", err)
	}

	// Recompose it
	recomposed, err := recomposeIterator(c, hash)
	if err != nil {
		t.Fatalf("Failed to recompose DictIterator: %v", err)
	}

	// Verify it's the right type
	dictIter, ok := recomposed.(*interp.DictIterator)
	if !ok {
		t.Fatalf("Expected *interp.DictIterator, got %T", recomposed)
	}

	// Verify state
	if dictIter.Index != original.Index {
		t.Errorf("Expected Index %d, got %d", original.Index, dictIter.Index)
	}
	if dictIter.VarCount != original.VarCount {
		t.Errorf("Expected VarCount %d, got %d", original.VarCount, dictIter.VarCount)
	}

	// Verify keys
	if len(dictIter.Keys) != len(original.Keys) {
		t.Errorf("Expected %d keys, got %d", len(original.Keys), len(dictIter.Keys))
	}
	for i, k := range original.Keys {
		if dictIter.Keys[i] != k {
			t.Errorf("Key[%d]: expected %s, got %s", i, k, dictIter.Keys[i])
		}
	}

	// Verify dict values
	for k, v := range original.Dict {
		if dictIter.Dict[k] != v {
			t.Errorf("Dict[%s]: expected %v, got %v", k, v, dictIter.Dict[k])
		}
	}
}

func TestDecomposeRecomposeIteratorState(t *testing.T) {
	c := NewMemoryCAS()

	// Create an IteratorState with a SliceIterator
	values := []vm.Value{
		vm.IntValue(10),
		vm.IntValue(20),
	}

	iter := &interp.SliceIterator{
		Values:   values,
		Index:    0,
		VarCount: 1,
	}

	original := &interp.IteratorState{
		Start:    vm.ExecPtr(100),
		End:      vm.ExecPtr(200),
		Iter:     iter,
		VarNames: []string{"x"},
	}

	// Decompose it
	hash, err := decomposeIteratorState(c, original)
	if err != nil {
		t.Fatalf("Failed to decompose IteratorState: %v", err)
	}

	// Recompose it
	recomposed, err := recomposeIteratorState(c, hash)
	if err != nil {
		t.Fatalf("Failed to recompose IteratorState: %v", err)
	}

	// Verify ExecPtrs
	if recomposed.Start != original.Start {
		t.Errorf("Expected Start %v, got %v", original.Start, recomposed.Start)
	}
	if recomposed.End != original.End {
		t.Errorf("Expected End %v, got %v", original.End, recomposed.End)
	}

	// Verify VarNames
	if len(recomposed.VarNames) != len(original.VarNames) {
		t.Errorf("Expected %d VarNames, got %d", len(original.VarNames), len(recomposed.VarNames))
	}
	for i, name := range original.VarNames {
		if recomposed.VarNames[i] != name {
			t.Errorf("VarName[%d]: expected %s, got %s", i, name, recomposed.VarNames[i])
		}
	}

	// Verify iterator was recomposed
	if recomposed.Iter == nil {
		t.Fatal("Expected non-nil Iter")
	}

	sliceIter, ok := recomposed.Iter.(*interp.SliceIterator)
	if !ok {
		t.Fatalf("Expected *interp.SliceIterator, got %T", recomposed.Iter)
	}

	if sliceIter.Index != iter.Index {
		t.Errorf("Expected iterator Index %d, got %d", iter.Index, sliceIter.Index)
	}
}

func TestDecomposeIteratorStateWithDictIterator(t *testing.T) {
	c := NewMemoryCAS()

	// Create an IteratorState with a DictIterator
	dict := vm.StructValue{
		"x": vm.IntValue(1),
		"y": vm.IntValue(2),
	}

	iter := &interp.DictIterator{
		Dict:     dict,
		Keys:     []string{"x", "y"},
		Index:    1,
		VarCount: 2,
	}

	original := &interp.IteratorState{
		Start:    vm.ExecPtr(50),
		End:      vm.ExecPtr(100),
		Iter:     iter,
		VarNames: []string{"key", "value"},
	}

	// Decompose and recompose
	hash, err := decomposeIteratorState(c, original)
	if err != nil {
		t.Fatalf("Failed to decompose IteratorState: %v", err)
	}

	recomposed, err := recomposeIteratorState(c, hash)
	if err != nil {
		t.Fatalf("Failed to recompose IteratorState: %v", err)
	}

	// Verify
	dictIter, ok := recomposed.Iter.(*interp.DictIterator)
	if !ok {
		t.Fatalf("Expected *interp.DictIterator, got %T", recomposed.Iter)
	}

	if dictIter.Index != iter.Index {
		t.Errorf("Expected Index %d, got %d", iter.Index, dictIter.Index)
	}
	if dictIter.VarCount != iter.VarCount {
		t.Errorf("Expected VarCount %d, got %d", iter.VarCount, dictIter.VarCount)
	}
	if len(recomposed.VarNames) != 2 {
		t.Errorf("Expected 2 VarNames, got %d", len(recomposed.VarNames))
	}
}

func TestIteratorStructuralSharing(t *testing.T) {
	c := NewMemoryCAS()

	// Create two SliceIterators with the same values
	values := []vm.Value{
		vm.IntValue(100),
		vm.IntValue(200),
	}

	iter1 := &interp.SliceIterator{
		Values:   values,
		Index:    0,
		VarCount: 1,
	}

	iter2 := &interp.SliceIterator{
		Values:   values,
		Index:    1, // Different index
		VarCount: 1,
	}

	// Decompose both
	hash1, err := decomposeIterator(c, iter1)
	if err != nil {
		t.Fatalf("Failed to decompose iter1: %v", err)
	}

	hash2, err := decomposeIterator(c, iter2)
	if err != nil {
		t.Fatalf("Failed to decompose iter2: %v", err)
	}

	// They should have different hashes (different Index)
	if hash1 == hash2 {
		t.Error("Expected different hashes for iterators with different Index")
	}

	// But the underlying value hashes should be shared in the CAS
	// This is implicit - we can't easily test it without inspecting internal state
}
