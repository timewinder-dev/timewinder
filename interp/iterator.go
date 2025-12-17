package interp

import (
	"slices"

	"github.com/timewinder-dev/timewinder/vm"
)

// SliceIterator iterates over array/list values
type SliceIterator struct {
	Values   []vm.Value // The array being iterated
	Index    int        // Current position (-1 = not started)
	VarCount int        // 1 or 2 variables
}

// Clone creates a deep copy of the SliceIterator
func (s *SliceIterator) Clone() Iterator {
	return &SliceIterator{
		Values:   slices.Clone(s.Values),
		Index:    s.Index,
		VarCount: s.VarCount,
	}
}

// Next advances the iterator to the next element
// Returns true if there are more elements, false if exhausted
func (s *SliceIterator) Next() bool {
	s.Index++
	return s.Index < len(s.Values)
}

// Var1 returns the first loop variable value
// For 1-var loops: returns the element
// For 2-var loops: returns the index
func (s *SliceIterator) Var1() vm.Value {
	if s.VarCount == 1 {
		return s.Values[s.Index] // Just the element
	}
	return vm.IntValue(s.Index) // Index for 2-var loops
}

// Var2 returns the second loop variable value
// For 1-var loops: returns None (unused)
// For 2-var loops: returns the element
func (s *SliceIterator) Var2() vm.Value {
	if s.VarCount == 2 {
		return s.Values[s.Index] // Element
	}
	return vm.None // Unused for 1-var loops
}

// DictIterator iterates over dict/struct key-value pairs
type DictIterator struct {
	Dict     vm.StructValue // The dict being iterated
	Keys     []string       // Sorted keys for deterministic iteration
	Index    int            // Current position (-1 = not started)
	VarCount int            // Should be 2 (key, value) but can be 1
}

// Clone creates a deep copy of the DictIterator
func (d *DictIterator) Clone() Iterator {
	return &DictIterator{
		Dict:     d.Dict, // StructValue is already a map, ok to share
		Keys:     slices.Clone(d.Keys),
		Index:    d.Index,
		VarCount: d.VarCount,
	}
}

// Next advances the iterator to the next key-value pair
// Returns true if there are more entries, false if exhausted
func (d *DictIterator) Next() bool {
	d.Index++
	return d.Index < len(d.Keys)
}

// Var1 returns the first loop variable value (the key)
func (d *DictIterator) Var1() vm.Value {
	key := d.Keys[d.Index]
	return vm.StrValue(key)
}

// Var2 returns the second loop variable value (the value)
// For 1-var loops: returns None (unused)
// For 2-var loops: returns the dict value for the current key
func (d *DictIterator) Var2() vm.Value {
	if d.VarCount == 2 {
		key := d.Keys[d.Index]
		return d.Dict[key] // Value
	}
	return vm.None
}
