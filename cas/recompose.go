package cas

import (
	"bytes"
	"fmt"

	msgpack "github.com/shamaton/msgpack/v2"
	"github.com/timewinder-dev/timewinder/interp"
	"github.com/timewinder-dev/timewinder/vm"
)

// recomposeState reconstructs a State from a StateRef stored in the CAS
func recomposeState(c *MemoryCAS, hash Hash) (*interp.State, error) {
	// Retrieve the StateRef
	ref, err := getDirect[*StateRef](c, hash)
	if err != nil {
		return nil, fmt.Errorf("retrieving StateRef: %w", err)
	}

	// Reconstruct globals
	globals, err := recomposeStackFrame(c, ref.GlobalsHash)
	if err != nil {
		return nil, fmt.Errorf("recomposing globals: %w", err)
	}

	// Reconstruct all thread stacks
	var stacks []interp.StackFrames
	for threadIdx, threadHashes := range ref.StacksHashes {
		var threadStack []*interp.StackFrame
		for frameIdx, h := range threadHashes {
			frame, err := recomposeStackFrame(c, h)
			if err != nil {
				return nil, fmt.Errorf("recomposing thread %d frame %d: %w", threadIdx, frameIdx, err)
			}
			threadStack = append(threadStack, frame)
		}
		stacks = append(stacks, threadStack)
	}

	// Reconstruct State
	return &interp.State{
		Globals:     globals,
		Stacks:      stacks,
		PauseReason: ref.PauseReasons,
	}, nil
}

// recomposeStackFrame reconstructs a StackFrame from a StackFrameRef
func recomposeStackFrame(c *MemoryCAS, hash Hash) (*interp.StackFrame, error) {
	// Retrieve the StackFrameRef
	ref, err := getDirect[*StackFrameRef](c, hash)
	if err != nil {
		return nil, fmt.Errorf("retrieving StackFrameRef: %w", err)
	}

	frame := &interp.StackFrame{
		PC:        ref.PC,
		Variables: make(map[string]vm.Value),
	}

	// Reconstruct stack values
	for i, h := range ref.StackHashes {
		v, err := recomposeValue(c, h)
		if err != nil {
			return nil, fmt.Errorf("recomposing stack value %d: %w", i, err)
		}
		frame.Stack = append(frame.Stack, v)
	}

	// Reconstruct variable values from parallel lists
	for i, name := range ref.VariableNames {
		h := ref.VariableHashes[i]
		v, err := recomposeValue(c, h)
		if err != nil {
			return nil, fmt.Errorf("recomposing variable %s: %w", name, err)
		}
		frame.Variables[name] = v
	}

	// Reconstruct iterators
	for i, h := range ref.IteratorHashes {
		iter, err := recomposeIteratorState(c, h)
		if err != nil {
			return nil, fmt.Errorf("recomposing iterator %d: %w", i, err)
		}
		frame.IteratorStack = append(frame.IteratorStack, iter)
	}

	return frame, nil
}

// recomposeIteratorState reconstructs an IteratorState from an IteratorStateRef
func recomposeIteratorState(c *MemoryCAS, hash Hash) (*interp.IteratorState, error) {
	// Retrieve the IteratorStateRef
	ref, err := getDirect[*IteratorStateRef](c, hash)
	if err != nil {
		return nil, fmt.Errorf("retrieving IteratorStateRef: %w", err)
	}

	// Reconstruct iterator
	// TODO: Handle iterator reconstruction properly
	// For now, we return nil since we didn't store it properly in decompose
	var iter interp.Iterator = nil

	return &interp.IteratorState{
		Start: ref.Start,
		End:   ref.End,
		Iter:  iter,
	}, nil
}

// recomposeValue reconstructs a vm.Value from the CAS
// Handles both inline values and Ref types
func recomposeValue(c *MemoryCAS, hash Hash) (vm.Value, error) {
	// Retrieve the raw bytes
	entryBytes, ok := c.data[hash]
	if !ok {
		return nil, fmt.Errorf("hash not found in CAS: %d", hash)
	}

	// Deserialize TypedEntry from bytes
	typedEntry := &TypedEntry{}
	entryBuf := bytes.NewReader(entryBytes)
	err := typedEntry.Deserialize(entryBuf)
	if err != nil {
		return nil, fmt.Errorf("deserializing TypedEntry: %w", err)
	}

	// Check if it's a Ref type that needs recomposition
	switch typedEntry.TypeTag {
	case "StructValueRef":
		// Reconstruct from StructValueRef using parallel lists
		ref, err := getDirect[*StructValueRef](c, hash)
		if err != nil {
			return nil, fmt.Errorf("retrieving StructValueRef: %w", err)
		}

		result := make(vm.StructValue)
		for i, name := range ref.FieldNames {
			h := ref.FieldHashes[i]
			v, err := recomposeValue(c, h)
			if err != nil {
				return nil, fmt.Errorf("recomposing struct field %s: %w", name, err)
			}
			result[name] = v
		}
		return result, nil

	case "ArrayValueRef":
		// Reconstruct from ArrayValueRef
		ref, err := getDirect[*ArrayValueRef](c, hash)
		if err != nil {
			return nil, fmt.Errorf("retrieving ArrayValueRef: %w", err)
		}

		var result vm.ArrayValue
		for i, h := range ref.ElementHashes {
			v, err := recomposeValue(c, h)
			if err != nil {
				return nil, fmt.Errorf("recomposing array element %d: %w", i, err)
			}
			result = append(result, v)
		}
		return result, nil

	default:
		// Not a Ref type, deserialize directly
		return deserializeValue(typedEntry)
	}
}

// deserializeValue deserializes a vm.Value from a TypedEntry
func deserializeValue(entry *TypedEntry) (vm.Value, error) {
	// vm.Value types are serialized directly with msgpack, not using the Hashable interface
	// We need to deserialize into the specific concrete type based on the type tag

	buf := bytes.NewReader(entry.Data)

	// Deserialize into the appropriate concrete type based on the type tag
	switch entry.TypeTag {
	case "BoolValue":
		var v vm.BoolValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "IntValue":
		var v vm.IntValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "FloatValue":
		var v vm.FloatValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "StrValue":
		var v vm.StrValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "NoneValue":
		var v vm.NoneValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "FnPtrValue":
		var v vm.FnPtrValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "StructValue":
		var v vm.StructValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "ArrayValue":
		var v vm.ArrayValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	case "ArgValue":
		var v vm.ArgValue
		err := msgpack.UnmarshalRead(buf, &v)
		return v, err
	default:
		return nil, fmt.Errorf("unknown value type tag: %s", entry.TypeTag)
	}
}

// getDirect retrieves an item from the CAS and deserializes it to type T
func getDirect[T Hashable](c *MemoryCAS, hash Hash) (T, error) {
	var zero T

	// Retrieve bytes from CAS
	entryBytes, ok := c.data[hash]
	if !ok {
		return zero, fmt.Errorf("hash not found in CAS: %d", hash)
	}

	// Deserialize TypedEntry from bytes
	typedEntry := &TypedEntry{}
	entryBuf := bytes.NewReader(entryBytes)
	err := typedEntry.Deserialize(entryBuf)
	if err != nil {
		return zero, fmt.Errorf("deserializing TypedEntry: %w", err)
	}

	// Create instance
	instance, err := createInstance(typedEntry.TypeTag)
	if err != nil {
		return zero, fmt.Errorf("creating instance: %w", err)
	}

	// Deserialize
	dataBuf := bytes.NewReader(typedEntry.Data)
	err = instance.Deserialize(dataBuf)
	if err != nil {
		return zero, fmt.Errorf("deserializing: %w", err)
	}

	// Type assert
	result, ok := instance.(T)
	if !ok {
		return zero, fmt.Errorf("type mismatch: expected %T, got %T", zero, instance)
	}

	return result, nil
}
