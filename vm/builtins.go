package vm

import (
	"fmt"
)

// BuiltinRegistry maps builtin function names to their implementations
var BuiltinRegistry = map[string]func(args []Value) (Value, error){
	"range": builtinRange,
	"oneof": builtinOneof,
}

// AllBuiltins contains the BuiltinValue instances to inject into global scope
var AllBuiltins = map[string]BuiltinValue{
	"range": {Name: "range"},
	"oneof": {Name: "oneof"},
}

// builtinRange implements Python-like range() function
// Supports 3 forms:
// - range(stop): returns [0, 1, ..., stop-1]
// - range(start, stop): returns [start, start+1, ..., stop-1]
// - range(start, stop, step): returns [start, start+step, ..., < stop]
func builtinRange(args []Value) (Value, error) {
	var start, stop, step int64

	// Parse arguments based on count
	switch len(args) {
	case 1:
		// range(stop)
		stopVal, ok := args[0].(IntValue)
		if !ok {
			return nil, fmt.Errorf("range() argument must be an integer, got %T", args[0])
		}
		start = 0
		stop = int64(stopVal)
		step = 1

	case 2:
		// range(start, stop)
		startVal, ok := args[0].(IntValue)
		if !ok {
			return nil, fmt.Errorf("range() start must be an integer, got %T", args[0])
		}
		stopVal, ok := args[1].(IntValue)
		if !ok {
			return nil, fmt.Errorf("range() stop must be an integer, got %T", args[1])
		}
		start = int64(startVal)
		stop = int64(stopVal)
		step = 1

	case 3:
		// range(start, stop, step)
		startVal, ok := args[0].(IntValue)
		if !ok {
			return nil, fmt.Errorf("range() start must be an integer, got %T", args[0])
		}
		stopVal, ok := args[1].(IntValue)
		if !ok {
			return nil, fmt.Errorf("range() stop must be an integer, got %T", args[1])
		}
		stepVal, ok := args[2].(IntValue)
		if !ok {
			return nil, fmt.Errorf("range() step must be an integer, got %T", args[2])
		}
		start = int64(startVal)
		stop = int64(stopVal)
		step = int64(stepVal)

		if step == 0 {
			return nil, fmt.Errorf("range() step argument must not be zero")
		}

	default:
		return nil, fmt.Errorf("range() takes 1 to 3 arguments, got %d", len(args))
	}

	// Build the range array
	var result ArrayValue

	if step > 0 {
		// Ascending range
		for i := start; i < stop; i += step {
			result = append(result, IntValue(i))
		}
	} else {
		// Descending range
		for i := start; i > stop; i += step {
			result = append(result, IntValue(i))
		}
	}

	return result, nil
}

// builtinOneof implements non-deterministic choice
// Takes a single array argument and returns a NonDetValue containing all choices
// The model checker will expand this into multiple execution branches
func builtinOneof(args []Value) (Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("oneof() takes exactly 1 argument, got %d", len(args))
	}

	arr, ok := args[0].(ArrayValue)
	if !ok {
		return nil, fmt.Errorf("oneof() argument must be an array, got %T", args[0])
	}

	if len(arr) == 0 {
		// Return None for empty arrays - non-deterministic "no choice"
		return None, nil
	}

	// Return NonDetValue - will trigger immediate expansion in the model checker
	return NonDetValue{Choices: arr}, nil
}
