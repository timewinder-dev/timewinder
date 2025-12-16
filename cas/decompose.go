package cas

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/dgryski/go-farm"
	msgpack "github.com/shamaton/msgpack/v2"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// decomposeState recursively decomposes a State into a StateRef with nested hashes
// Each nested structure (StackFrame, Value, etc.) is stored separately in the CAS
// Returns the hash of the top-level StateRef
func decomposeState(c *MemoryCAS, s *interp.State) (Hash, error) {
	if s == nil {
		return 0, fmt.Errorf("cannot decompose nil State")
	}

	// Decompose globals
	globalsHash, err := decomposeStackFrame(c, s.Globals)
	if err != nil {
		return 0, fmt.Errorf("decomposing globals: %w", err)
	}

	// Decompose all thread stacks
	var stacksHashes [][]Hash
	for threadIdx, threadStack := range s.Stacks {
		var threadHashes []Hash
		for frameIdx, frame := range threadStack {
			h, err := decomposeStackFrame(c, frame)
			if err != nil {
				return 0, fmt.Errorf("decomposing thread %d frame %d: %w", threadIdx, frameIdx, err)
			}
			threadHashes = append(threadHashes, h)
		}
		stacksHashes = append(stacksHashes, threadHashes)
	}

	// Create and store StateRef
	ref := &StateRef{
		GlobalsHash:  globalsHash,
		StacksHashes: stacksHashes,
		PauseReasons: s.PauseReason,
	}

	return putDirect(c, ref)
}

// decomposeStackFrame recursively decomposes a StackFrame into a StackFrameRef with nested hashes
func decomposeStackFrame(c *MemoryCAS, f *interp.StackFrame) (Hash, error) {
	if f == nil {
		return 0, fmt.Errorf("cannot decompose nil StackFrame")
	}

	// Decompose stack values
	var stackHashes []Hash
	for i, v := range f.Stack {
		h, err := decomposeValue(c, v)
		if err != nil {
			return 0, fmt.Errorf("decomposing stack value %d: %w", i, err)
		}
		stackHashes = append(stackHashes, h)
	}

	// Decompose variable values into parallel sorted lists for deterministic ordering
	varNames := make([]string, 0, len(f.Variables))
	for name := range f.Variables {
		varNames = append(varNames, name)
	}
	sort.Strings(varNames) // Sort for deterministic ordering

	varHashes := make([]Hash, len(varNames))
	for i, name := range varNames {
		h, err := decomposeValue(c, f.Variables[name])
		if err != nil {
			return 0, fmt.Errorf("decomposing variable %s: %w", name, err)
		}
		varHashes[i] = h
	}

	// Decompose iterators
	var iterHashes []Hash
	for i, iter := range f.IteratorStack {
		h, err := decomposeIteratorState(c, iter)
		if err != nil {
			return 0, fmt.Errorf("decomposing iterator %d: %w", i, err)
		}
		iterHashes = append(iterHashes, h)
	}

	// Create and store StackFrameRef
	ref := &StackFrameRef{
		StackHashes:    stackHashes,
		PC:             f.PC,
		VariableNames:  varNames,
		VariableHashes: varHashes,
		IteratorHashes: iterHashes,
	}

	return putDirect(c, ref)
}

// decomposeIteratorState decomposes an IteratorState
func decomposeIteratorState(c *MemoryCAS, iter *interp.IteratorState) (Hash, error) {
	if iter == nil {
		return 0, fmt.Errorf("cannot decompose nil IteratorState")
	}

	// Decompose the iterator itself
	iterHash, err := decomposeIterator(c, iter.Iter)
	if err != nil {
		return 0, fmt.Errorf("decomposing iterator: %w", err)
	}

	// Create and store IteratorStateRef
	ref := &IteratorStateRef{
		Start:    iter.Start,
		End:      iter.End,
		IterHash: iterHash,
	}

	return putDirect(c, ref)
}

// decomposeIterator decomposes an Iterator
// For now, we store iterators directly since they're relatively small
func decomposeIterator(c *MemoryCAS, iter interp.Iterator) (Hash, error) {
	if iter == nil {
		return 0, fmt.Errorf("cannot decompose nil Iterator")
	}

	// Store iterator directly
	// Note: Iterator interface doesn't implement Hashable, so we need to handle this specially
	// For now, we'll just return 0 and handle this in recompose
	// TODO: Properly implement iterator serialization if needed
	return 0, nil
}

// decomposeValue decomposes a vm.Value
// Simple values are stored inline, complex values are decomposed to Refs
func decomposeValue(c *MemoryCAS, v vm.Value) (Hash, error) {
	if v == nil {
		return 0, fmt.Errorf("cannot decompose nil Value")
	}

	switch val := v.(type) {
	case vm.BoolValue, vm.IntValue, vm.FloatValue, vm.StrValue, vm.NoneValue, vm.FnPtrValue:
		// Simple values: store directly using msgpack
		return putValueDirect(c, val)

	case vm.ArgValue:
		// ArgValue: store it directly
		return putValueDirect(c, val)

	case vm.StructValue:
		// Always use reference for structs to avoid msgpack deserialization issues
		// with vm.Value interface types
		// Use parallel sorted lists for deterministic ordering
		fieldNames := make([]string, 0, len(val))
		for k := range val {
			fieldNames = append(fieldNames, k)
		}
		sort.Strings(fieldNames) // Sort for deterministic ordering

		fieldHashes := make([]Hash, len(fieldNames))
		for i, k := range fieldNames {
			h, err := decomposeValue(c, val[k])
			if err != nil {
				return 0, fmt.Errorf("decomposing struct field %s: %w", k, err)
			}
			fieldHashes[i] = h
		}
		ref := &StructValueRef{
			FieldNames:  fieldNames,
			FieldHashes: fieldHashes,
		}
		return putDirect(c, ref)

	case vm.ArrayValue:
		// Always use reference for arrays to avoid msgpack deserialization issues
		// with vm.Value interface types
		var elemHashes []Hash
		for i, elem := range val {
			h, err := decomposeValue(c, elem)
			if err != nil {
				return 0, fmt.Errorf("decomposing array element %d: %w", i, err)
			}
			elemHashes = append(elemHashes, h)
		}
		ref := &ArrayValueRef{ElementHashes: elemHashes}
		return putDirect(c, ref)

	default:
		return 0, fmt.Errorf("unknown value type: %T", v)
	}
}

// putDirect stores an item directly in the CAS without further decomposition
// Returns the hash of the stored item
func putDirect(c *MemoryCAS, item Hashable) (Hash, error) {
	// Serialize the item
	var buf bytes.Buffer
	err := item.Serialize(&buf)
	if err != nil {
		return 0, fmt.Errorf("serializing item: %w", err)
	}

	// Compute hash
	data := buf.Bytes()
	h := Hash(farm.Hash64(data))

	// Check if already stored
	if _, ok := c.data[h]; ok {
		// Already exists, return existing hash
		return h, nil
	}

	// Wrap with type tag
	tag := getTypeTag(item)
	entry := &TypedEntry{
		TypeTag: tag,
		Data:    data,
	}

	// Serialize the typed entry
	var entryBuf bytes.Buffer
	err = entry.Serialize(&entryBuf)
	if err != nil {
		return 0, fmt.Errorf("serializing typed entry: %w", err)
	}

	// Store serialized entry bytes in CAS
	// Note: We use the hash of the original item, not the typed entry
	// This allows structural sharing based on content
	c.data[h] = entryBuf.Bytes()

	return h, nil
}

// putValueDirect stores a vm.Value directly using msgpack serialization
// This is needed because vm.Value types don't implement the full Hashable interface
func putValueDirect(c *MemoryCAS, v vm.Value) (Hash, error) {
	// Serialize using msgpack
	var buf bytes.Buffer
	err := msgpack.MarshalWrite(&buf, v)
	if err != nil {
		return 0, fmt.Errorf("serializing value: %w", err)
	}

	// Compute hash
	data := buf.Bytes()
	h := Hash(farm.Hash64(data))

	// Check if already stored
	if _, ok := c.data[h]; ok {
		return h, nil
	}

	// Get type tag
	tag := getValueTypeTag(v)

	// Wrap with type tag
	entry := &TypedEntry{
		TypeTag: tag,
		Data:    data,
	}

	// Serialize the typed entry
	var entryBuf bytes.Buffer
	err = entry.Serialize(&entryBuf)
	if err != nil {
		return 0, fmt.Errorf("serializing typed entry: %w", err)
	}

	// Store serialized entry bytes in CAS
	c.data[h] = entryBuf.Bytes()

	return h, nil
}

// getValueTypeTag returns the type tag for a vm.Value
func getValueTypeTag(v vm.Value) string {
	switch v.(type) {
	case vm.BoolValue:
		return "BoolValue"
	case vm.IntValue:
		return "IntValue"
	case vm.FloatValue:
		return "FloatValue"
	case vm.StrValue:
		return "StrValue"
	case vm.NoneValue:
		return "NoneValue"
	case vm.FnPtrValue:
		return "FnPtrValue"
	case vm.StructValue:
		return "StructValue"
	case vm.ArrayValue:
		return "ArrayValue"
	case vm.ArgValue:
		return "ArgValue"
	default:
		return fmt.Sprintf("%T", v)
	}
}
